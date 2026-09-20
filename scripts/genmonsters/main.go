// Generates migrations/014_catalog_monsters.sql from 5e14 bestiary index + 5eapi 2014.
// 5eapi monsters supply fightable stats (HP, AC, CR, DEX, resist, actions, SRD specials).
// 5e14 cards supply numeric IDs, RU names/URLs, and CR/type from index + filter JSON.
// 5e14-only cards become stubs (name, URL, CR/type). No article-body scrape.
package main

import (
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
)

const (
	dnd14ListURL  = "https://5e14.dnd.su/piece/bestiary/index-list/"
	dnd14PageURL  = "https://5e14.dnd.su/bestiary/"
	apiListURL    = "https://www.dnd5eapi.co/api/2014/monsters"
	apiMonsterURL = "https://www.dnd5eapi.co/api/2014/monsters/"
	apiOnlyIDBase = 1000000
	homebrewSrc   = 310
	maxHomebrew   = 400
)

type apiList struct {
	Results []struct {
		Index string `json:"index"`
		Name  string `json:"name"`
		URL   string `json:"url"`
	} `json:"results"`
}

type namedRef struct {
	Index string `json:"index"`
	Name  string `json:"name"`
}

type apiMonster struct {
	Index      string `json:"index"`
	Name       string `json:"name"`
	Size       string `json:"size"`
	Type       string `json:"type"`
	ArmorClass []struct {
		Type  string `json:"type"`
		Value int    `json:"value"`
	} `json:"armor_class"`
	HitPoints             int             `json:"hit_points"`
	HitDice               string          `json:"hit_dice"`
	Speed                 map[string]any  `json:"speed"`
	Dexterity             int             `json:"dexterity"`
	ChallengeRating       float64         `json:"challenge_rating"`
	XP                    int             `json:"xp"`
	DamageVulnerabilities []string        `json:"damage_vulnerabilities"`
	DamageResistances     []string        `json:"damage_resistances"`
	DamageImmunities      []string        `json:"damage_immunities"`
	ConditionImmunities   []namedRef      `json:"condition_immunities"`
	SpecialAbilities      []apiAction     `json:"special_abilities"`
	Actions               []apiAction     `json:"actions"`
	LegendaryActions      []apiAction     `json:"legendary_actions"`
	Desc                  json.RawMessage `json:"desc"`
}

type apiAction struct {
	Name        string `json:"name"`
	Desc        string `json:"desc"`
	AttackBonus int    `json:"attack_bonus"`
	Damage      []struct {
		DamageDice string   `json:"damage_dice"`
		DamageType namedRef `json:"damage_type"`
	} `json:"damage"`
}

type card14 struct {
	ID       int
	TitleRU  string
	TitleEN  string
	Link     string
	CRLabel  string
	Sources  []int
	TypeID   int
	SizeID   int
	DangerID int
}

type filterEntry struct {
	Type   []string `json:"type"`
	Size   []string `json:"size"`
	Danger []string `json:"danger"`
	Source []string `json:"source"`
	NPC    []string `json:"npc"`
}

type filterMaps struct {
	Type     map[int]string
	Size     map[int]string
	Danger   map[int]crInfo
	WotC     map[int]bool
	Homebrew map[int]bool
}

type crInfo struct {
	Label string
	CR    float64
	XP    int
}

type seedRow struct {
	ID                     int64
	Slug, NameEN, NameRU   string
	Size, Type             string
	ArmorClass, HitPoints  int
	HitDice, Speed         string
	Dexterity              int
	CR                     float64
	CRLabel                string
	XP                     int
	Resist, Immune, Vuln   []string
	CondImmune             []string
	Actions                string
	DescEN                 string
	SourceURL, SourceURLRU string
	IsStub                 bool
}

func main() {
	root, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	if filepath.Base(root) == "genmonsters" {
		root = filepath.Join(root, "..", "..")
	}
	outPath := filepath.Join(root, "api", "migrations", "014_catalog_monsters.sql")

	client := &http.Client{Timeout: 45 * time.Second}

	fmt.Println("fetching 5e14 bestiary listing (filter maps)…")
	pageRaw, err := fetch(client, dnd14PageURL)
	if err != nil {
		panic(err)
	}
	maps := parseFilterMaps(pageRaw)
	fmt.Printf("filter maps: types %d sizes %d cr %d wotc-sources %d homebrew-sources %d\n",
		len(maps.Type), len(maps.Size), len(maps.Danger), len(maps.WotC), len(maps.Homebrew))

	fmt.Println("fetching 5e14 bestiary index…")
	raw14, err := fetch(client, dnd14ListURL)
	if err != nil {
		panic(err)
	}
	cards, err := parse14(raw14)
	if err != nil {
		panic(err)
	}
	fmt.Printf("5e14 cards: %d\n", len(cards))

	byNorm := map[string]card14{}
	bySlug := map[string]card14{}
	for _, c := range cards {
		if n := norm(c.TitleEN); n != "" {
			if _, ok := byNorm[n]; !ok {
				byNorm[n] = c
			}
		}
		if n := norm(c.TitleRU); n != "" {
			if _, ok := byNorm[n]; !ok {
				byNorm[n] = c
			}
		}
		sg := slugFromLink(c.Link)
		if sg != "" {
			if _, ok := bySlug[sg]; !ok {
				bySlug[sg] = c
			}
			if _, ok := byNorm[norm(sg)]; !ok {
				byNorm[norm(sg)] = c
			}
		}
	}

	fmt.Println("fetching 5eapi 2014 monsters list…")
	list := mustList(client, apiListURL)
	fmt.Printf("5eapi monsters: %d\n", len(list.Results))

	type job struct {
		idx  string
		name string
		card *card14
	}
	jobs := make([]job, 0, len(list.Results))
	matchedID := map[int]bool{}
	for _, r := range list.Results {
		c := matchCard(byNorm, bySlug, r.Index, r.Name)
		var cp *card14
		if c != nil {
			cc := *c
			cp = &cc
			if c.ID > 0 {
				matchedID[c.ID] = true
			}
		}
		jobs = append(jobs, job{idx: r.Index, name: r.Name, card: cp})
	}

	details := make([]*apiMonster, len(jobs))
	var wg sync.WaitGroup
	sem := make(chan struct{}, 12)
	for i, j := range jobs {
		wg.Add(1)
		go func(i int, j job) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			body, err := fetch(client, apiMonsterURL+j.idx)
			if err != nil {
				fmt.Printf("skip monster %s: %v\n", j.idx, err)
				return
			}
			var m apiMonster
			if err := json.Unmarshal(body, &m); err != nil {
				fmt.Printf("parse monster %s: %v\n", j.idx, err)
				return
			}
			details[i] = &m
		}(i, j)
	}
	wg.Wait()

	var rows []seedRow
	usedID := map[int64]bool{}
	usedSlug := map[string]bool{}
	var apiOnly []seedRow
	matched := 0
	for i, j := range jobs {
		if details[i] == nil {
			continue
		}
		row := buildRow(details[i], j.card)
		if row.Slug == "" || row.HitPoints < 1 {
			continue
		}
		if usedSlug[row.Slug] {
			continue
		}
		if j.card != nil && j.card.ID > 0 && !usedID[int64(j.card.ID)] {
			row.ID = int64(j.card.ID)
			usedID[row.ID] = true
			usedSlug[row.Slug] = true
			rows = append(rows, row)
			matched++
			continue
		}
		apiOnly = append(apiOnly, row)
	}
	sort.Slice(apiOnly, func(i, j int) bool { return apiOnly[i].Slug < apiOnly[j].Slug })
	for i, row := range apiOnly {
		row.ID = int64(apiOnlyIDBase + i)
		for usedID[row.ID] {
			row.ID++
		}
		usedID[row.ID] = true
		usedSlug[row.Slug] = true
		rows = append(rows, row)
	}
	srd := len(rows)

	var stubCards []card14
	for _, c := range cards {
		if c.ID == 0 || matchedID[c.ID] || usedID[int64(c.ID)] {
			continue
		}
		sg := slugFromLink(c.Link)
		if sg != "" && usedSlug[sg] {
			continue
		}
		stubCards = append(stubCards, c)
	}
	kept := pickStubs(stubCards, maps)
	stubN := 0
	var spiderURL string
	for _, c := range kept {
		row := buildStub(c, maps)
		if row.Slug == "" || usedSlug[row.Slug] || usedID[row.ID] {
			continue
		}
		usedID[row.ID] = true
		usedSlug[row.Slug] = true
		rows = append(rows, row)
		stubN++
		if row.Slug == "spiderdragon" {
			spiderURL = row.SourceURLRU
		}
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].NameEN < rows[j].NameEN })
	fmt.Printf("seed rows: %d (5eapi/srd %d, 5e14 ids %d, stubs %d)\n", len(rows), srd, matched, stubN)
	if spiderURL == "" {
		fmt.Println("WARNING: Spiderdragon stub not found")
	} else {
		fmt.Println("Spiderdragon:", spiderURL)
	}

	if err := os.WriteFile(outPath, []byte(renderSQL(rows, matched, srd, stubN)), 0o644); err != nil {
		panic(err)
	}
	fmt.Println("wrote", outPath)
}

func pickStubs(cards []card14, maps filterMaps) []card14 {
	var official, homebrew, always []card14
	seen := map[int]bool{}
	for _, c := range cards {
		if c.ID == 0 || seen[c.ID] {
			continue
		}
		seen[c.ID] = true
		if alwaysKeep(c) {
			always = append(always, c)
			continue
		}
		if isHomebrewCard(c, maps) {
			homebrew = append(homebrew, c)
			continue
		}
		official = append(official, c)
	}
	out := append([]card14{}, official...)
	// 5e14 2014 bestiary is WotC books (MM, MotM, adventures…). Include all of those.
	// If a Homebrew group appears and would dump thousands of user cards, keep
	// only named uniques (Spiderdragon and similar) plus a small cap.
	if len(homebrew) > 0 && len(homebrew) <= maxHomebrew && len(official)+len(homebrew) < 800 {
		out = append(out, homebrew...)
		homebrew = nil
	}
	keptHB := 0
	for _, c := range homebrew {
		if alwaysKeep(c) || uniqueNamed(c) {
			out = append(out, c)
			keptHB++
		}
	}
	for _, c := range always {
		found := false
		for _, x := range out {
			if x.ID == c.ID {
				found = true
				break
			}
		}
		if !found {
			out = append(out, c)
		}
	}
	if keptHB > 0 || len(always) > 0 {
		fmt.Printf("stubs kept: official %d, unique/homebrew %d, always %d (dropped homebrew %d)\n",
			len(official), keptHB, len(always), len(homebrew)-keptHB)
	}
	return out
}

func alwaysKeep(c card14) bool {
	if norm(c.TitleEN) == "spiderdragon" || slugFromLink(c.Link) == "spiderdragon" {
		return true
	}
	return false
}

func uniqueNamed(c card14) bool {
	if alwaysKeep(c) {
		return true
	}
	en := strings.TrimSpace(c.TitleEN)
	if en == "" || strings.Contains(en, "(") {
		return false
	}
	words := strings.Fields(en)
	if len(words) == 0 || len(words) > 3 {
		return false
	}
	for _, w := range words {
		r := []rune(w)
		if len(r) == 0 || !unicode.IsUpper(r[0]) {
			return false
		}
	}
	return true
}

func isHomebrewCard(c card14, maps filterMaps) bool {
	if len(c.Sources) == 0 {
		return false
	}
	wotc := false
	hb := false
	for _, s := range c.Sources {
		if maps.Homebrew[s] || s == homebrewSrc {
			hb = true
		}
		if maps.WotC[s] {
			wotc = true
		}
	}
	if wotc {
		return false
	}
	return hb
}

func matchCard(byNorm map[string]card14, bySlug map[string]card14, idx, name string) *card14 {
	if c, ok := bySlug[idx]; ok {
		return &c
	}
	if c, ok := byNorm[norm(idx)]; ok {
		return &c
	}
	if c, ok := byNorm[norm(name)]; ok {
		return &c
	}
	return nil
}

func buildRow(m *apiMonster, c *card14) seedRow {
	if m == nil {
		return seedRow{}
	}
	r := seedRow{
		Slug: m.Index, NameEN: m.Name, Size: strings.ToLower(m.Size), Type: strings.ToLower(m.Type),
		HitPoints: m.HitPoints, HitDice: m.HitDice, Dexterity: m.Dexterity,
		CR: m.ChallengeRating, CRLabel: crLabel(m.ChallengeRating), XP: m.XP,
		Resist: nilToEmpty(m.DamageResistances), Immune: nilToEmpty(m.DamageImmunities),
		Vuln:      nilToEmpty(m.DamageVulnerabilities),
		SourceURL: apiMonsterURL + m.Index,
	}
	if r.Dexterity == 0 {
		r.Dexterity = 10
	}
	if len(m.ArmorClass) > 0 {
		r.ArmorClass = m.ArmorClass[0].Value
	}
	r.Speed = formatSpeed(m.Speed)
	for _, ci := range m.ConditionImmunities {
		if ci.Index != "" {
			r.CondImmune = append(r.CondImmune, ci.Index)
		} else if ci.Name != "" {
			r.CondImmune = append(r.CondImmune, strings.ToLower(ci.Name))
		}
	}
	if r.CondImmune == nil {
		r.CondImmune = []string{}
	}
	r.Actions = mustJSON(slimActions(m.Actions))
	r.DescEN = parseDesc(m.Desc)
	if r.DescEN == "" {
		r.DescEN = joinSpecials(m.SpecialAbilities)
	}
	if c != nil {
		r.NameRU = strings.TrimSpace(c.TitleRU)
		r.SourceURLRU = "https://5e14.dnd.su" + c.Link
	}
	if r.NameRU == "" {
		r.NameRU = r.NameEN
	}
	return r
}

func buildStub(c card14, maps filterMaps) seedRow {
	slug := slugFromLink(c.Link)
	nameEN := strings.TrimSpace(c.TitleEN)
	nameRU := strings.TrimSpace(c.TitleRU)
	if nameEN == "" {
		nameEN = nameRU
	}
	if nameRU == "" {
		nameRU = nameEN
	}
	if slug == "" {
		slug = slugify(nameEN)
	}
	info := crFromCard(c, maps)
	r := seedRow{
		ID: int64(c.ID), Slug: slug, NameEN: nameEN, NameRU: nameRU,
		Size: maps.Size[c.SizeID], Type: maps.Type[c.TypeID],
		ArmorClass: 10, HitPoints: 0, Dexterity: 10,
		CR: info.CR, CRLabel: info.Label, XP: info.XP,
		Resist: []string{}, Immune: []string{}, Vuln: []string{}, CondImmune: []string{},
		Actions: "[]", SourceURLRU: "https://5e14.dnd.su" + c.Link, IsStub: true,
	}
	if r.CRLabel == "" {
		r.CRLabel = "0"
	}
	return r
}

func crFromCard(c card14, maps filterMaps) crInfo {
	if lab := strings.TrimSpace(c.CRLabel); lab != "" {
		info := parseCRLabel(lab)
		if d, ok := maps.Danger[c.DangerID]; ok && info.XP == 0 {
			info.XP = d.XP
		}
		if info.XP == 0 {
			info.XP = xpForCR(info.CR, info.Label)
		}
		return info
	}
	if d, ok := maps.Danger[c.DangerID]; ok {
		return d
	}
	return crInfo{Label: "0"}
}

func parseCRLabel(lab string) crInfo {
	lab = strings.TrimSpace(lab)
	info := crInfo{Label: lab}
	switch lab {
	case "1/8":
		info.CR = 0.125
	case "1/4":
		info.CR = 0.25
	case "1/2":
		info.CR = 0.5
	default:
		if n, err := strconv.ParseFloat(lab, 64); err == nil {
			info.CR = n
		}
	}
	info.XP = xpForCR(info.CR, lab)
	return info
}

func xpForCR(cr float64, label string) int {
	switch label {
	case "0":
		return 10
	case "1/8":
		return 25
	case "1/4":
		return 50
	case "1/2":
		return 100
	}
	table := map[int]int{
		1: 200, 2: 450, 3: 700, 4: 1100, 5: 1800, 6: 2300, 7: 2900, 8: 3900, 9: 5000,
		10: 5900, 11: 7200, 12: 8400, 13: 10000, 14: 11500, 15: 13000, 16: 15000, 17: 18000,
		18: 20000, 19: 22000, 20: 25000, 21: 33000, 22: 41000, 23: 50000, 24: 62000, 25: 75000,
		26: 90000, 27: 105000, 28: 120000, 29: 135000, 30: 155000,
	}
	if xp, ok := table[int(cr+0.001)]; ok && cr == float64(int(cr)) {
		return xp
	}
	return 0
}

type slimAction struct {
	Name        string `json:"name"`
	Desc        string `json:"desc,omitempty"`
	AttackBonus int    `json:"attack_bonus,omitempty"`
	Damage      string `json:"damage,omitempty"`
}

func slimActions(in []apiAction) []slimAction {
	out := make([]slimAction, 0, len(in))
	for _, a := range in {
		s := slimAction{Name: a.Name, Desc: a.Desc, AttackBonus: a.AttackBonus}
		var dmg []string
		for _, d := range a.Damage {
			part := d.DamageDice
			if d.DamageType.Index != "" {
				part += " " + d.DamageType.Index
			}
			if part != "" {
				dmg = append(dmg, part)
			}
		}
		s.Damage = strings.Join(dmg, ", ")
		out = append(out, s)
	}
	return out
}

func formatSpeed(m map[string]any) string {
	if len(m) == 0 {
		return ""
	}
	order := []string{"walk", "fly", "swim", "climb", "burrow", "hover"}
	seen := map[string]bool{}
	var parts []string
	for _, k := range order {
		if v, ok := m[k]; ok {
			seen[k] = true
			parts = append(parts, k+" "+fmt.Sprint(v))
		}
	}
	var extra []string
	for k, v := range m {
		if seen[k] {
			continue
		}
		extra = append(extra, k+" "+fmt.Sprint(v))
	}
	sort.Strings(extra)
	parts = append(parts, extra...)
	return strings.Join(parts, ", ")
}

func crLabel(v float64) string {
	switch {
	case v == 0:
		return "0"
	case almost(v, 0.125):
		return "1/8"
	case almost(v, 0.25):
		return "1/4"
	case almost(v, 0.5):
		return "1/2"
	}
	if v == float64(int(v)) {
		return strconv.Itoa(int(v))
	}
	return strconv.FormatFloat(v, 'g', -1, 64)
}

func almost(a, b float64) bool {
	d := a - b
	if d < 0 {
		d = -d
	}
	return d < 0.0001
}

func parseDesc(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	var many []string
	if err := json.Unmarshal(raw, &many); err == nil {
		return joinDesc(many)
	}
	var one string
	if err := json.Unmarshal(raw, &one); err == nil {
		return strings.TrimSpace(one)
	}
	return ""
}

func joinDesc(d []string) string {
	var parts []string
	for _, s := range d {
		s = strings.TrimSpace(s)
		if s != "" {
			parts = append(parts, s)
		}
	}
	return strings.Join(parts, "\n")
}

func joinSpecials(a []apiAction) string {
	var parts []string
	for _, s := range a {
		line := strings.TrimSpace(s.Name)
		if s.Desc != "" {
			if line != "" {
				line += ". "
			}
			line += strings.TrimSpace(s.Desc)
		}
		if line != "" {
			parts = append(parts, line)
		}
	}
	return strings.Join(parts, "\n")
}

func renderSQL(rows []seedRow, matched, srd, stubs int) string {
	var b strings.Builder
	b.WriteString(`-- Generated catalog monsters. Do not edit by hand; rerun scripts/genmonsters.
-- 5e14 IDs from GET https://5e14.dnd.su/piece/bestiary/index-list/ (HTML cards + window.filterItems).
-- Filter maps (CR/type/size/source) from https://5e14.dnd.su/bestiary/ listing JSON, not article HTML.
-- 5eapi 2014 monsters; URLs stored only when the detail endpoint returned 200.
-- 5e14-only cards are stubs (is_stub=1): name, 5e14 URL, CR/type from the index. No lore scrape.
-- Existing DBs that already applied an older 014 will not reseed on restart; re-run this file's
-- DELETE+INSERT (after ALTER TABLE catalog_monsters ADD COLUMN is_stub INTEGER NOT NULL DEFAULT 0
-- if that column is missing). Catalog only — live battle_units snapshot names/HP.
-- 5eapi-only IDs start at 1000000 (stable by slug sort).
-- monster_count: `)
	b.WriteString(strconv.Itoa(len(rows)))
	b.WriteString("\n-- matched_5e14: ")
	b.WriteString(strconv.Itoa(matched))
	b.WriteString("\n-- srd_count: ")
	b.WriteString(strconv.Itoa(srd))
	b.WriteString("\n-- stub_count: ")
	b.WriteString(strconv.Itoa(stubs))
	b.WriteString("\n\n")
	b.WriteString(`CREATE TABLE IF NOT EXISTS catalog_monsters (
    id INTEGER PRIMARY KEY,
    slug TEXT NOT NULL UNIQUE,
    name_en TEXT NOT NULL,
    name_ru TEXT NOT NULL DEFAULT '',
    size TEXT NOT NULL DEFAULT '',
    type TEXT NOT NULL DEFAULT '',
    armor_class INTEGER NOT NULL DEFAULT 10,
    hit_points INTEGER NOT NULL DEFAULT 0,
    hit_dice TEXT NOT NULL DEFAULT '',
    speed TEXT NOT NULL DEFAULT '',
    dexterity INTEGER NOT NULL DEFAULT 10,
    cr REAL NOT NULL DEFAULT 0,
    cr_label TEXT NOT NULL DEFAULT '0',
    xp INTEGER NOT NULL DEFAULT 0,
    damage_resistances TEXT NOT NULL DEFAULT '[]',
    damage_immunities TEXT NOT NULL DEFAULT '[]',
    damage_vulnerabilities TEXT NOT NULL DEFAULT '[]',
    condition_immunities TEXT NOT NULL DEFAULT '[]',
    actions TEXT NOT NULL DEFAULT '[]',
    desc_en TEXT NOT NULL DEFAULT '',
    source_url TEXT NOT NULL DEFAULT '',
    source_url_ru TEXT NOT NULL DEFAULT '',
    is_stub INTEGER NOT NULL DEFAULT 0
);

DELETE FROM catalog_monsters;

INSERT INTO catalog_monsters (
    id, slug, name_en, name_ru, size, type, armor_class, hit_points, hit_dice, speed,
    dexterity, cr, cr_label, xp, damage_resistances, damage_immunities, damage_vulnerabilities,
    condition_immunities, actions, desc_en, source_url, source_url_ru, is_stub
) VALUES
`)
	for i, r := range rows {
		if i > 0 {
			b.WriteString(",\n")
		}
		fmt.Fprintf(&b, "(%d, %s, %s, %s, %s, %s, %d, %d, %s, %s, %d, %s, %s, %d, %s, %s, %s, %s, %s, %s, %s, %s, %d)",
			r.ID, q(r.Slug), q(r.NameEN), q(r.NameRU), q(r.Size), q(r.Type),
			r.ArmorClass, r.HitPoints, q(r.HitDice), q(r.Speed), r.Dexterity,
			fmt.Sprintf("%g", r.CR), q(r.CRLabel), r.XP,
			q(mustJSON(r.Resist)), q(mustJSON(r.Immune)), q(mustJSON(r.Vuln)),
			q(mustJSON(r.CondImmune)), q(r.Actions), q(r.DescEN), q(r.SourceURL), q(r.SourceURLRU),
			btoi(r.IsStub),
		)
	}
	b.WriteString(";\n")
	return b.String()
}

func mustList(c *http.Client, url string) apiList {
	body, err := fetch(c, url)
	if err != nil {
		panic(err)
	}
	var list apiList
	if err := json.Unmarshal(body, &list); err != nil {
		panic(err)
	}
	return list
}

func fetch(c *http.Client, url string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "DnDManager/1.0 (catalog seed)")
	req.Header.Set("Accept", "application/json, text/html;q=0.9")
	res, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("%s: HTTP %d", url, res.StatusCode)
	}
	return body, nil
}

func parse14(raw []byte) ([]card14, error) {
	s := string(raw)
	filters := map[string]filterEntry{}
	if i := strings.Index(s, "window.filterItems"); i >= 0 {
		eq := strings.Index(s[i:], "=")
		if eq >= 0 {
			dec := json.NewDecoder(strings.NewReader(strings.TrimSpace(s[i+eq+1:])))
			_ = dec.Decode(&filters)
		}
	}
	blockRe := regexp.MustCompile(`(?s)data-search='([^']*)'\s+data-id='(\d+)'[^>]*>.*?href='(/bestiary/[^']+/)'(?:.*?<span class='list-mark__danger'>\[<span>([^<]*)</span>\])?`)
	matches := blockRe.FindAllStringSubmatch(s, -1)
	if len(matches) == 0 {
		return nil, fmt.Errorf("5e14 bestiary: no cards")
	}
	var out []card14
	seen := map[int]bool{}
	for _, m := range matches {
		id, _ := strconv.Atoi(m[2])
		if id == 0 || seen[id] {
			continue
		}
		seen[id] = true
		ru, en := splitSearch(m[1])
		c := card14{ID: id, TitleRU: ru, TitleEN: en, Link: m[3], CRLabel: strings.TrimSpace(m[4])}
		if f, ok := filters[strconv.Itoa(id)]; ok {
			c.Sources = atoiAll(f.Source)
			if len(f.Type) > 0 {
				c.TypeID, _ = strconv.Atoi(f.Type[0])
			}
			if len(f.Size) > 0 {
				c.SizeID, _ = strconv.Atoi(f.Size[0])
			}
			if len(f.Danger) > 0 {
				c.DangerID, _ = strconv.Atoi(f.Danger[0])
			}
		}
		out = append(out, c)
	}
	return out, nil
}

func parseFilterMaps(raw []byte) filterMaps {
	m := filterMaps{
		Type:     map[int]string{},
		Size:     map[int]string{},
		Danger:   map[int]crInfo{},
		WotC:     map[int]bool{},
		Homebrew: map[int]bool{},
	}
	s := string(raw)
	re := regexp.MustCompile(`name="(type|size|danger|source)"[^>]*data-list="([^"]*)"`)
	for _, g := range re.FindAllStringSubmatch(s, -1) {
		kind, payload := g[1], html.UnescapeString(g[2])
		switch kind {
		case "type":
			for id, title := range parseIDTitles(payload) {
				m.Type[id] = typeSlug(title)
			}
		case "size":
			for id, title := range parseIDTitles(payload) {
				m.Size[id] = sizeSlug(title)
			}
		case "danger":
			for id, title := range parseIDTitles(payload) {
				m.Danger[id] = parseDangerTitle(title)
			}
		case "source":
			wotc, hb := parseSourceGroups(payload)
			for id := range wotc {
				m.WotC[id] = true
			}
			for id := range hb {
				m.Homebrew[id] = true
			}
		}
	}
	if len(m.Type) == 0 {
		m.Type = fallbackTypes()
	}
	if len(m.Size) == 0 {
		m.Size = fallbackSizes()
	}
	return m
}

type titledValue struct {
	Title string `json:"title"`
	Value int    `json:"value"`
}

func parseIDTitles(payload string) map[int]string {
	out := map[int]string{}
	var groups []struct {
		Title string          `json:"title"`
		List  json.RawMessage `json:"list"`
	}
	if err := json.Unmarshal([]byte(payload), &groups); err == nil {
		for _, g := range groups {
			for id, title := range titledList(g.List) {
				out[id] = title
			}
		}
		if len(out) > 0 {
			return out
		}
	}
	var obj map[string]struct {
		Title string          `json:"title"`
		List  json.RawMessage `json:"list"`
	}
	if err := json.Unmarshal([]byte(payload), &obj); err == nil {
		for _, g := range obj {
			for id, title := range titledList(g.List) {
				out[id] = title
			}
		}
	}
	return out
}

func titledList(raw json.RawMessage) map[int]string {
	out := map[int]string{}
	if len(raw) == 0 {
		return out
	}
	var arr []titledValue
	if err := json.Unmarshal(raw, &arr); err == nil {
		for _, v := range arr {
			if v.Value != 0 {
				out[v.Value] = v.Title
			}
		}
		return out
	}
	var obj map[string]titledValue
	if err := json.Unmarshal(raw, &obj); err == nil {
		for _, v := range obj {
			if v.Value != 0 {
				out[v.Value] = v.Title
			}
		}
	}
	return out
}

func parseSourceGroups(payload string) (wotc, homebrew map[int]bool) {
	wotc, homebrew = map[int]bool{}, map[int]bool{}
	var obj map[string]struct {
		Title string          `json:"title"`
		List  json.RawMessage `json:"list"`
	}
	if err := json.Unmarshal([]byte(payload), &obj); err != nil {
		for id := range parseIDTitles(payload) {
			wotc[id] = true
		}
		return wotc, homebrew
	}
	for _, g := range obj {
		ids := titledList(g.List)
		title := strings.ToLower(g.Title)
		hb := strings.Contains(title, "homebrew") || strings.Contains(title, "домаш")
		for id := range ids {
			if hb {
				homebrew[id] = true
			} else {
				wotc[id] = true
			}
		}
	}
	return wotc, homebrew
}

func parseDangerTitle(title string) crInfo {
	title = strings.TrimSpace(title)
	lab := title
	if i := strings.Index(title, " "); i > 0 {
		lab = title[:i]
	}
	if i := strings.Index(lab, "-"); i > 0 {
		lab = strings.TrimSpace(lab[:i])
	}
	info := parseCRLabel(lab)
	if n := regexp.MustCompile(`(\d[\d\s]*)\s*(опыта|xp)`).FindStringSubmatch(strings.ToLower(title)); len(n) > 1 {
		xp, _ := strconv.Atoi(strings.ReplaceAll(n[1], " ", ""))
		if xp > 0 {
			info.XP = xp
		}
	}
	return info
}

func typeSlug(title string) string {
	t := strings.ToLower(strings.TrimSpace(title))
	pairs := []struct{ ru, en string }{
		{"аберрац", "aberration"}, {"великан", "giant"}, {"гуманоид", "humanoid"},
		{"дракон", "dragon"}, {"зверь", "beast"}, {"исчад", "fiend"},
		{"конструкт", "construct"}, {"монстр", "monstrosity"}, {"небожител", "celestial"},
		{"нежить", "undead"}, {"растен", "plant"}, {"слизь", "ooze"}, {"фея", "fey"},
		{"элементал", "elemental"}, {"рой", "swarm"}, {"объект", "object"},
	}
	for _, p := range pairs {
		if strings.Contains(t, p.ru) || strings.Contains(t, p.en) {
			return p.en
		}
	}
	return t
}

func sizeSlug(title string) string {
	t := strings.ToLower(strings.TrimSpace(title))
	switch {
	case strings.Contains(t, "крошеч") || strings.Contains(t, "tiny"):
		return "tiny"
	case strings.Contains(t, "гигант") || strings.Contains(t, "gargantuan"):
		return "gargantuan"
	case strings.Contains(t, "огром") || strings.Contains(t, "huge"):
		return "huge"
	case strings.Contains(t, "больш") || strings.Contains(t, "large"):
		return "large"
	case strings.Contains(t, "малень") || strings.Contains(t, "small"):
		return "small"
	case strings.Contains(t, "средн") || strings.Contains(t, "medium"):
		return "medium"
	}
	return t
}

func fallbackTypes() map[int]string {
	return map[int]string{
		19: "humanoid", 20: "undead", 21: "dragon", 22: "beast", 23: "monstrosity",
		24: "aberration", 25: "giant", 26: "plant", 27: "ooze", 28: "celestial",
		29: "construct", 30: "elemental", 31: "fiend", 32: "fey", 33: "swarm", 34: "object",
	}
}

func fallbackSizes() map[int]string {
	return map[int]string{1: "tiny", 2: "small", 3: "medium", 4: "large", 5: "huge", 6: "gargantuan"}
}

func splitSearch(s string) (ru, en string) {
	parts := strings.Split(s, ",")
	if len(parts) > 0 {
		ru = strings.TrimSpace(parts[0])
	}
	if len(parts) > 1 {
		en = strings.TrimSpace(parts[1])
	}
	return ru, en
}

func atoiAll(ss []string) []int {
	var out []int
	for _, s := range ss {
		n, err := strconv.Atoi(strings.TrimSpace(s))
		if err == nil {
			out = append(out, n)
		}
	}
	return out
}

func slugFromLink(link string) string {
	link = strings.Trim(link, "/")
	parts := strings.Split(link, "/")
	if len(parts) == 0 {
		return ""
	}
	last := parts[len(parts)-1]
	if i := strings.Index(last, "-"); i >= 0 {
		return last[i+1:]
	}
	return last
}

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	prevDash := false
	for _, r := range s {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			b.WriteRune(r)
			prevDash = false
			continue
		}
		if !prevDash && b.Len() > 0 {
			b.WriteByte('-')
			prevDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}

func norm(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	for _, r := range s {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func nilToEmpty(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

func mustJSON(v any) string {
	b, _ := json.Marshal(v)
	if len(b) == 0 {
		return "[]"
	}
	return string(b)
}

func q(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

func btoi(v bool) int {
	if v {
		return 1
	}
	return 0
}
