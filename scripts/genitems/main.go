// Generates migrations/010_items_catalog.sql from 5e14 items index + 5eapi 2014.
// 5eapi equipment + magic-items supply stats and SRD desc. 5e14 DMG-only cards
// that 5eapi 404s (Moonblade, …) become stubs with name + RU URL, no lore scrape.
package main

import (
	"encoding/json"
	"fmt"
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
)

const (
	dnd14ListURL   = "https://5e14.dnd.su/piece/items/index-list/"
	apiEquipList   = "https://www.dnd5eapi.co/api/2014/equipment"
	apiEquipURL    = "https://www.dnd5eapi.co/api/2014/equipment/"
	apiMagicList   = "https://www.dnd5eapi.co/api/2014/magic-items"
	apiMagicURL    = "https://www.dnd5eapi.co/api/2014/magic-items/"
	dmgSource      = 101
	homebrewSource = 310
	apiOnlyIDBase  = 1000000
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

type apiCost struct {
	Quantity float64 `json:"quantity"`
	Unit     string  `json:"unit"`
}

type apiDamage struct {
	DamageDice string   `json:"damage_dice"`
	DamageType namedRef `json:"damage_type"`
}

type apiEquip struct {
	Index               string   `json:"index"`
	Name                string   `json:"name"`
	Desc                []string `json:"desc"`
	EquipmentCategory   namedRef `json:"equipment_category"`
	ArmorCategory       string   `json:"armor_category"`
	WeaponCategory      string   `json:"weapon_category"`
	Weight              float64  `json:"weight"`
	Cost                apiCost  `json:"cost"`
	StrMinimum          int      `json:"str_minimum"`
	StealthDisadvantage bool     `json:"stealth_disadvantage"`
	ArmorClass          *struct {
		Base     int  `json:"base"`
		DexBonus bool `json:"dex_bonus"`
		MaxBonus *int `json:"max_bonus"`
	} `json:"armor_class"`
	Damage          *apiDamage `json:"damage"`
	TwoHandedDamage *apiDamage `json:"two_handed_damage"`
	Range           *struct {
		Normal int `json:"normal"`
		Long   int `json:"long"`
	} `json:"range"`
	ThrowRange *struct {
		Normal int `json:"normal"`
		Long   int `json:"long"`
	} `json:"throw_range"`
	Properties []namedRef `json:"properties"`
}

type apiMagic struct {
	Index             string   `json:"index"`
	Name              string   `json:"name"`
	Desc              []string `json:"desc"`
	EquipmentCategory namedRef `json:"equipment_category"`
	Rarity            struct {
		Name string `json:"name"`
	} `json:"rarity"`
	Variant bool `json:"variant"`
}

type card14 struct {
	ID         int
	TitleRU    string
	TitleEN    string
	Link       string
	Sources    []int
	Attunement bool
	TypeID     int
	QualityID  int
}

type filterEntry struct {
	Type       []string `json:"type"`
	Quality    []string `json:"quality"`
	Attunement []string `json:"attunement"`
	Source     []string `json:"source"`
}

type job struct {
	kind string // "equipment" | "magic"
	idx  string
	card *card14
}

type seedRow struct {
	ID                                     int64
	Slug, NameEN, NameRU, Kind             string
	CostGP, WeightLB                       float64
	DamageDice, DamageType, ArmorClass     string
	Properties                             []string
	Rarity                                 string
	SourceURL, SourceURLRU, DescEN         string
	ArmorCategory                          string
	ACBase, DexMax                         int
	StealthDisadv                          bool
	StrMin                                 int
	WeaponCategory, VersatileDice          string
	RangeNormal, RangeLong                 int
	SuggestedSlot                          string
	RequiresAttunement, Consumable, IsStub bool
	ChargesMax                             int
}

func main() {
	root, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	if filepath.Base(root) == "genitems" {
		root = filepath.Join(root, "..", "..")
	}
	outPath := filepath.Join(root, "api", "migrations", "010_items_catalog.sql")

	client := &http.Client{Timeout: 45 * time.Second}

	fmt.Println("fetching 5e14 items index…")
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
	var dmgOnly []card14
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
		if hasSource(c, dmgSource) && !hasSource(c, homebrewSource) {
			dmgOnly = append(dmgOnly, c)
		}
	}

	fmt.Println("fetching 5eapi 2014 equipment + magic-items lists…")
	equipList := mustList(client, apiEquipList)
	magicList := mustList(client, apiMagicList)
	fmt.Printf("5eapi equipment: %d  magic-items: %d\n", len(equipList.Results), len(magicList.Results))

	var jobs []job
	seenAPI := map[string]bool{}
	addJob := func(kind, idx, name string) {
		c := matchCard(byNorm, bySlug, idx, name)
		var cp *card14
		if c != nil {
			cc := *c
			cp = &cc
		}
		jobs = append(jobs, job{kind: kind, idx: idx, card: cp})
		seenAPI[norm(idx)] = true
		seenAPI[norm(name)] = true
	}
	for _, r := range equipList.Results {
		addJob("equipment", r.Index, r.Name)
	}
	for _, r := range magicList.Results {
		if seenAPI[norm(r.Index)] {
			continue
		}
		addJob("magic", r.Index, r.Name)
	}

	seenStub := map[string]bool{}
	for n := range seenAPI {
		seenStub[n] = true
	}
	var stubJobs []job
	for _, c := range dmgOnly {
		n := norm(c.TitleEN)
		sg := slugFromLink(c.Link)
		if n != "" && seenStub[n] {
			continue
		}
		if sg != "" && (seenStub[norm(sg)] || seenAPI[norm(sg)]) {
			continue
		}
		if n != "" {
			seenStub[n] = true
		}
		if sg != "" {
			seenStub[norm(sg)] = true
		}
		cc := c
		stubJobs = append(stubJobs, job{card: &cc})
	}

	fmt.Printf("detail fetches: %d  dmg stubs: %d\n", len(jobs), len(stubJobs))

	equips := make([]*apiEquip, len(jobs))
	magics := make([]*apiMagic, len(jobs))
	var wg sync.WaitGroup
	sem := make(chan struct{}, 12)
	for i, j := range jobs {
		wg.Add(1)
		go func(i int, j job) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			switch j.kind {
			case "equipment":
				body, err := fetch(client, apiEquipURL+j.idx)
				if err != nil {
					fmt.Printf("skip equip %s: %v\n", j.idx, err)
					return
				}
				var eq apiEquip
				if err := json.Unmarshal(body, &eq); err != nil {
					fmt.Printf("parse equip %s: %v\n", j.idx, err)
					return
				}
				equips[i] = &eq
			case "magic":
				body, err := fetch(client, apiMagicURL+j.idx)
				if err != nil {
					fmt.Printf("skip magic %s: %v\n", j.idx, err)
					return
				}
				var mg apiMagic
				if err := json.Unmarshal(body, &mg); err != nil {
					fmt.Printf("parse magic %s: %v\n", j.idx, err)
					return
				}
				magics[i] = &mg
			}
		}(i, j)
	}
	wg.Wait()

	var rows []seedRow
	usedID := map[int64]bool{}
	usedSlug := map[string]bool{}
	var apiOnly []seedRow
	srd, stubs := 0, 0
	for i, j := range jobs {
		row := buildAPIRow(j, equips[i], magics[i])
		if row.Slug == "" {
			continue
		}
		if usedSlug[row.Slug] {
			row.Slug = row.Slug + "-" + j.kind
		}
		if j.card != nil && j.card.ID > 0 && !usedID[int64(j.card.ID)] {
			row.ID = int64(j.card.ID)
		} else {
			apiOnly = append(apiOnly, row)
			continue
		}
		usedID[row.ID] = true
		usedSlug[row.Slug] = true
		rows = append(rows, row)
		srd++
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
		srd++
	}
	for _, j := range stubJobs {
		row := buildStub(j.card)
		if row.Slug == "" || usedSlug[row.Slug] {
			continue
		}
		if row.ID == 0 || usedID[row.ID] {
			continue
		}
		usedID[row.ID] = true
		usedSlug[row.Slug] = true
		rows = append(rows, row)
		stubs++
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Kind != rows[j].Kind {
			return rows[i].Kind < rows[j].Kind
		}
		return rows[i].NameEN < rows[j].NameEN
	})
	fmt.Printf("seed rows: %d (srd %d, stubs %d)\n", len(rows), srd, stubs)

	if err := os.WriteFile(outPath, []byte(renderSQL(rows, srd, stubs)), 0o644); err != nil {
		panic(err)
	}
	fmt.Println("wrote", outPath)
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

func buildAPIRow(j job, eq *apiEquip, mg *apiMagic) seedRow {
	r := seedRow{DexMax: -1, Properties: []string{}}
	if eq != nil {
		r.Slug = eq.Index
		r.NameEN = eq.Name
		r.Kind = kindFromCategory(eq.EquipmentCategory.Index, eq.ArmorCategory)
		r.CostGP = costGP(eq.Cost)
		r.WeightLB = eq.Weight
		r.DescEN = joinDesc(eq.Desc)
		r.ArmorCategory = strings.ToLower(eq.ArmorCategory)
		r.WeaponCategory = strings.ToLower(eq.WeaponCategory)
		r.StrMin = eq.StrMinimum
		r.StealthDisadv = eq.StealthDisadvantage
		if eq.ArmorClass != nil {
			r.ACBase = eq.ArmorClass.Base
			if !eq.ArmorClass.DexBonus {
				r.DexMax = 0
			} else if eq.ArmorClass.MaxBonus != nil {
				r.DexMax = *eq.ArmorClass.MaxBonus
			} else {
				r.DexMax = -1
			}
			r.ArmorClass = armorClassString(r.ArmorCategory, r.ACBase, eq.ArmorClass.DexBonus, r.DexMax)
		}
		if eq.Damage != nil {
			r.DamageDice = eq.Damage.DamageDice
			r.DamageType = strings.ToLower(eq.Damage.DamageType.Index)
		}
		if eq.TwoHandedDamage != nil {
			r.VersatileDice = eq.TwoHandedDamage.DamageDice
		}
		if eq.Range != nil {
			r.RangeNormal, r.RangeLong = eq.Range.Normal, eq.Range.Long
		}
		if eq.ThrowRange != nil {
			if r.RangeNormal == 0 {
				r.RangeNormal = eq.ThrowRange.Normal
			}
			if r.RangeLong == 0 {
				r.RangeLong = eq.ThrowRange.Long
			}
		}
		for _, p := range eq.Properties {
			if p.Index != "" {
				r.Properties = append(r.Properties, p.Index)
			}
		}
		r.SourceURL = apiEquipURL + eq.Index
		r.Consumable = r.Kind == "potion" || r.Kind == "scroll" || r.Kind == "ammo" || r.Kind == "consumable"
	} else if mg != nil {
		r.Slug = mg.Index
		r.NameEN = mg.Name
		r.Kind = kindFromCategory(mg.EquipmentCategory.Index, "")
		r.Rarity = strings.ToLower(mg.Rarity.Name)
		r.DescEN = joinDesc(mg.Desc)
		r.SourceURL = apiMagicURL + mg.Index
		r.RequiresAttunement = descAttunement(mg.Desc)
		r.ChargesMax = descCharges(mg.Desc)
		r.Consumable = r.Kind == "potion" || r.Kind == "scroll" || r.Kind == "ammo"
		if dice := healDiceFor(mg.Index, mg.Name, mg.Desc); dice != "" {
			r.DamageDice = dice
			r.DamageType = "healing"
		}
	} else {
		return seedRow{}
	}
	if j.card != nil {
		applyCard(&r, j.card)
	}
	if r.NameRU == "" {
		r.NameRU = r.NameEN
	}
	applyHealOverlay(&r)
	r.SuggestedSlot = suggestSlot(r)
	return r
}

func buildStub(c *card14) seedRow {
	if c == nil {
		return seedRow{}
	}
	r := seedRow{
		ID: int64(c.ID), DexMax: -1, IsStub: true, Properties: []string{},
		Slug: slugFromLink(c.Link), NameEN: titleCase(c.TitleEN), NameRU: strings.TrimSpace(c.TitleRU),
		Kind: kindFromTypeID(c.TypeID), Rarity: rarityFromQuality(c.QualityID),
		SourceURLRU: "https://5e14.dnd.su" + c.Link, RequiresAttunement: c.Attunement,
	}
	if r.Slug == "" {
		r.Slug = kebab(c.TitleEN)
	}
	if r.NameEN == "" {
		r.NameEN = r.NameRU
	}
	if r.NameRU == "" {
		r.NameRU = r.NameEN
	}
	r.Consumable = r.Kind == "potion" || r.Kind == "scroll"
	r.SuggestedSlot = suggestSlot(r)
	return r
}

func applyCard(r *seedRow, c *card14) {
	r.NameRU = strings.TrimSpace(c.TitleRU)
	r.SourceURLRU = "https://5e14.dnd.su" + c.Link
	if c.Attunement {
		r.RequiresAttunement = true
	}
	if r.Rarity == "" {
		r.Rarity = rarityFromQuality(c.QualityID)
	}
	if r.Kind == "" || r.Kind == "gear" {
		if k := kindFromTypeID(c.TypeID); k != "" && k != "wondrous" {
			r.Kind = k
		}
	}
}

func applyHealOverlay(r *seedRow) {
	if dice := healDiceFor(r.Slug, r.NameEN, nil); dice != "" {
		r.DamageDice = dice
		if r.DamageType == "" {
			r.DamageType = "healing"
		}
		r.Consumable = true
		if r.Kind == "" || r.Kind == "gear" {
			r.Kind = "potion"
		}
	}
}

func healDiceFor(slug, name string, desc []string) string {
	switch slug {
	case "potion-of-healing", "potion-of-healing-common":
		return "2d4+2"
	case "potion-of-healing-greater", "potion-of-greater-healing":
		return "4d4+4"
	case "potion-of-healing-superior", "potion-of-superior-healing":
		return "8d4+8"
	case "potion-of-healing-supreme", "potion-of-supreme-healing":
		return "10d4+20"
	}
	n := strings.ToLower(name)
	if strings.Contains(n, "potion of healing") && !strings.Contains(n, "greater") &&
		!strings.Contains(n, "superior") && !strings.Contains(n, "supreme") {
		return "2d4+2"
	}
	joined := strings.ToLower(strings.Join(desc, " "))
	re := regexp.MustCompile(`regain\s+(\d+d\d+(?:\s*\+\s*\d+)?)\s+hit points`)
	if m := re.FindStringSubmatch(joined); len(m) == 2 {
		return strings.ReplaceAll(m[1], " ", "")
	}
	return ""
}

func suggestSlot(r seedRow) string {
	for _, p := range r.Properties {
		if p == "shield" {
			return "shield"
		}
	}
	cat := strings.ToLower(r.ArmorCategory)
	if cat == "shield" || r.Kind == "shield" {
		return "shield"
	}
	if r.Kind == "armor" || cat == "light" || cat == "medium" || cat == "heavy" {
		return "armor"
	}
	switch r.Kind {
	case "weapon", "wand", "staff", "rod":
		return "main_hand"
	case "potion", "scroll", "ammo", "gear", "tool", "pack":
		return ""
	}
	h := " " + strings.ToLower(strings.NewReplacer("-", " ", "'", "", "/", " ").Replace(r.NameEN+" "+r.Slug+" "+r.Kind)) + " "
	switch {
	case strings.Contains(h, " ring "):
		return "ring"
	case strings.Contains(h, " cloak "), strings.Contains(h, " cape "), strings.Contains(h, " mantle "), strings.Contains(h, " robe "):
		return "cloak"
	case strings.Contains(h, " boot "), strings.Contains(h, " boots "), strings.Contains(h, " shoe "):
		return "boots"
	case strings.Contains(h, " glove "), strings.Contains(h, " gloves "), strings.Contains(h, " gauntlet "):
		return "gloves"
	case strings.Contains(h, " belt "), strings.Contains(h, " girdle "):
		return "belt"
	case strings.Contains(h, " helm "), strings.Contains(h, " helmet "), strings.Contains(h, " headband "),
		strings.Contains(h, " hat "), strings.Contains(h, " circlet "), strings.Contains(h, " crown "),
		strings.Contains(h, " cap "):
		return "head"
	case strings.Contains(h, " amulet "), strings.Contains(h, " necklace "), strings.Contains(h, " periapt "),
		strings.Contains(h, " medallion "), strings.Contains(h, " pendant "), strings.Contains(h, " talisman "):
		return "neck"
	case strings.Contains(h, " shield "):
		return "shield"
	case strings.Contains(h, " armor "), strings.Contains(h, " mail "), strings.Contains(h, " plate "),
		strings.Contains(h, " breastplate "):
		return "armor"
	}
	return ""
}

func kindFromCategory(idx, armorCat string) string {
	switch strings.ToLower(idx) {
	case "weapon":
		return "weapon"
	case "armor":
		if strings.EqualFold(armorCat, "shield") {
			return "armor"
		}
		return "armor"
	case "potion":
		return "potion"
	case "ring":
		return "ring"
	case "wand":
		return "wand"
	case "staff":
		return "staff"
	case "rod":
		return "rod"
	case "scroll":
		return "scroll"
	case "ammunition":
		return "ammo"
	case "wondrous-items", "wondrous-item":
		return "wondrous"
	case "tools", "tool":
		return "tool"
	case "adventuring-gear":
		return "gear"
	default:
		if idx == "" {
			return "gear"
		}
		return strings.ToLower(idx)
	}
}

func kindFromTypeID(id int) string {
	switch id {
	case 1:
		return "wondrous"
	case 2:
		return "potion"
	case 3:
		return "ring"
	case 4:
		return "scroll"
	case 5:
		return "wand"
	case 6:
		return "rod"
	case 7:
		return "staff"
	case 8:
		return "armor"
	case 9:
		return "weapon"
	default:
		return "wondrous"
	}
}

func rarityFromQuality(id int) string {
	switch id {
	case 6:
		return "common"
	case 1:
		return "uncommon"
	case 2:
		return "rare"
	case 3:
		return "very rare"
	case 4:
		return "legendary"
	case 7:
		return "artifact"
	case 5:
		return "varies"
	default:
		return ""
	}
}

func armorClassString(cat string, base int, dexBonus bool, dexMax int) string {
	if cat == "shield" {
		if base == 0 {
			return "+2"
		}
		return fmt.Sprintf("+%d", base)
	}
	if base == 0 {
		return ""
	}
	if !dexBonus || dexMax == 0 {
		return strconv.Itoa(base)
	}
	if dexMax > 0 {
		return fmt.Sprintf("%d+DEX(max %d)", base, dexMax)
	}
	return fmt.Sprintf("%d+DEX", base)
}

func costGP(c apiCost) float64 {
	q := c.Quantity
	switch strings.ToLower(c.Unit) {
	case "cp":
		return q / 100
	case "sp":
		return q / 10
	case "ep":
		return q / 2
	case "pp":
		return q * 10
	default:
		return q
	}
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

func descAttunement(desc []string) bool {
	if len(desc) == 0 {
		return false
	}
	return strings.Contains(strings.ToLower(desc[0]), "requires attunement")
}

func descCharges(desc []string) int {
	re := regexp.MustCompile(`(?i)has (\d+) charges`)
	for _, s := range desc {
		if m := re.FindStringSubmatch(s); len(m) == 2 {
			n, _ := strconv.Atoi(m[1])
			return n
		}
	}
	return 0
}

func renderSQL(rows []seedRow, srd, stubs int) string {
	var b strings.Builder
	b.WriteString(`-- Generated catalog items. Do not edit by hand; rerun scripts/genitems.
-- 5e14 IDs from GET https://5e14.dnd.su/piece/items/index-list/ (HTML cards + window.filterItems).
-- 5eapi 2014 equipment + magic-items; URLs stored only when the detail endpoint returned 200.
-- Stubs: 5e14 DMG (source 101) cards with no 5eapi match. No lore article scrape.
-- 5eapi-only IDs start at 1000000 (stable by slug sort).
-- srd_count: `)
	b.WriteString(strconv.Itoa(srd))
	b.WriteString("\n-- stub_count: ")
	b.WriteString(strconv.Itoa(stubs))
	b.WriteString("\n-- item_count: ")
	b.WriteString(strconv.Itoa(len(rows)))
	b.WriteString("\n\nDELETE FROM catalog_items;\n\n")
	b.WriteString(`INSERT INTO catalog_items (
    id, slug, name_en, name_ru, kind, cost_gp, weight_lb, damage_dice, damage_type, armor_class,
    properties, rarity, source_url, source_url_ru, desc_en, armor_category, ac_base, dex_max,
    stealth_disadv, str_min, weapon_category, versatile_dice, range_normal, range_long,
    suggested_slot, requires_attunement, consumable, charges_max, is_stub
) VALUES
`)
	for i, r := range rows {
		if i > 0 {
			b.WriteString(",\n")
		}
		props := r.Properties
		if props == nil {
			props = []string{}
		}
		fmt.Fprintf(&b, "(%d, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %d, %d, %d, %d, %s, %s, %d, %d, %s, %d, %d, %d, %d)",
			r.ID, q(r.Slug), q(r.NameEN), q(r.NameRU), q(r.Kind),
			fmt.Sprintf("%g", r.CostGP), fmt.Sprintf("%g", r.WeightLB),
			q(r.DamageDice), q(r.DamageType), q(r.ArmorClass),
			q(mustJSON(props)), q(r.Rarity), q(r.SourceURL), q(r.SourceURLRU), q(r.DescEN),
			q(r.ArmorCategory), r.ACBase, r.DexMax, btoi(r.StealthDisadv), r.StrMin,
			q(r.WeaponCategory), q(r.VersatileDice), r.RangeNormal, r.RangeLong,
			q(r.SuggestedSlot), btoi(r.RequiresAttunement), btoi(r.Consumable), r.ChargesMax, btoi(r.IsStub),
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
	blockRe := regexp.MustCompile(`(?s)data-search='([^']*)'\s+data-id='(\d+)'[^>]*>.*?href='(/items/[^']+/)'`)
	matches := blockRe.FindAllStringSubmatch(s, -1)
	if len(matches) == 0 {
		return nil, fmt.Errorf("5e14 items: no cards")
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
		c := card14{ID: id, TitleRU: ru, TitleEN: en, Link: m[3]}
		if f, ok := filters[strconv.Itoa(id)]; ok {
			c.Sources = atoiAll(f.Source)
			c.Attunement = hasStr(f.Attunement, "2")
			if len(f.Type) > 0 {
				c.TypeID, _ = strconv.Atoi(f.Type[0])
			}
			if len(f.Quality) > 0 {
				c.QualityID, _ = strconv.Atoi(f.Quality[0])
			}
		}
		out = append(out, c)
	}
	return out, nil
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

func hasStr(ss []string, want string) bool {
	for _, s := range ss {
		if s == want {
			return true
		}
	}
	return false
}

func hasSource(c card14, id int) bool {
	for _, s := range c.Sources {
		if s == id {
			return true
		}
	}
	return false
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

func kebab(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, "'", "")
	s = strings.ReplaceAll(s, "/", " ")
	re := regexp.MustCompile(`[^a-z0-9]+`)
	s = re.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

func titleCase(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}
	parts := strings.Fields(s)
	for i, p := range parts {
		if p == "" {
			continue
		}
		parts[i] = strings.ToUpper(p[:1]) + strings.ToLower(p[1:])
	}
	return strings.Join(parts, " ")
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
