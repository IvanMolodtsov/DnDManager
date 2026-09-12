// Generates migrations/004_spells_and_resources.sql from 5e14 index + 5eapi 2014.
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
	dnd14ListURL = "https://5e14.dnd.su/piece/spells/index-list/"
	apiListURL   = "https://www.dnd5eapi.co/api/2014/spells"
	apiSpellURL  = "https://www.dnd5eapi.co/api/2014/spells/"
	phbSource    = 102
)

type listFile struct {
	Cards []card `json:"cards"`
}

type card struct {
	Title     string `json:"title"`
	TitleEN   string `json:"title_en"`
	Link      string `json:"link"`
	Level     string `json:"level"`
	School    string `json:"school"`
	Sources   []int  `json:"filter_source"`
	Ritual    []any  `json:"filter_ritual"`
	Conc      []any  `json:"filter_concentration"`
	CastTime  []any  `json:"filter_casttime"`
	ItemSfx   string `json:"item_suffix"`
}

type apiList struct {
	Results []struct {
		Index string `json:"index"`
		Name  string `json:"name"`
		URL   string `json:"url"`
	} `json:"results"`
}

type apiSpell struct {
	Index        string `json:"index"`
	Name         string `json:"name"`
	Level        int    `json:"level"`
	Ritual       bool   `json:"ritual"`
	Concentration bool  `json:"concentration"`
	CastingTime  string `json:"casting_time"`
	Range        string `json:"range"`
	Duration     string `json:"duration"`
	Components   []string `json:"components"`
	School       struct {
		Name string `json:"name"`
	} `json:"school"`
	Classes []struct {
		Index string `json:"index"`
	} `json:"classes"`
	Damage *struct {
		DamageType *struct {
			Index string `json:"index"`
			Name  string `json:"name"`
		} `json:"damage_type"`
		AtSlot      map[string]string `json:"damage_at_slot_level"`
		AtCharacter map[string]string `json:"damage_at_character_level"`
	} `json:"damage"`
	HealAtSlot map[string]string `json:"heal_at_slot_level"`
	HigherLevel []string `json:"higher_level"`
}

type seedRow struct {
	Slug, NameEN, NameRU, School, CastingTime, Range, Duration, Components string
	Classes                                                                []string
	Level                                                                  int
	Ritual, Concentration, Upcast                                          bool
	SourceURL, SourceURLRU                                                 string
	DamageFormula, DamageType, HealFormula, ScaleKind                      string
	DamageAtSlot, DamageAtCharacter                                        string
}

func main() {
	root, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	if filepath.Base(root) == "genspells" {
		root = filepath.Join(root, "..", "..")
	}
	outPath := filepath.Join(root, "migrations", "004_spells_and_resources.sql")

	client := &http.Client{Timeout: 30 * time.Second}

	fmt.Println("fetching 5e14 index…")
	raw14, err := fetch(client, dnd14ListURL)
	if err != nil {
		panic(err)
	}
	cards, err := parse14(raw14)
	if err != nil {
		panic(err)
	}
	fmt.Printf("5e14 cards: %d\n", len(cards))

	byNorm := map[string]card{}
	var phb []card
	for _, c := range cards {
		n := norm(c.TitleEN)
		if n != "" {
			if _, ok := byNorm[n]; !ok {
				byNorm[n] = c
			}
		}
		if hasSource(c, phbSource) {
			phb = append(phb, c)
		}
	}

	fmt.Println("fetching 5eapi 2014 list…")
	listBody, err := fetch(client, apiListURL)
	if err != nil {
		panic(err)
	}
	var list apiList
	if err := json.Unmarshal(listBody, &list); err != nil {
		panic(err)
	}

	type job struct {
		idx  string
		card *card
	}
	var jobs []job
	seen := map[string]bool{}
	for _, r := range list.Results {
		c, ok := byNorm[norm(r.Index)]
		if !ok {
			c, ok = byNorm[norm(r.Name)]
		}
		var cp *card
		if ok {
			cc := c
			cp = &cc
		}
		jobs = append(jobs, job{idx: r.Index, card: cp})
		seen[norm(r.Index)] = true
		seen[norm(r.Name)] = true
	}
	for _, c := range phb {
		n := norm(c.TitleEN)
		if n == "" || seen[n] {
			continue
		}
		seen[n] = true
		cc := c
		jobs = append(jobs, job{card: &cc})
	}

	fmt.Printf("detail fetches: %d\n", len(jobs))
	details := make([]*apiSpell, len(jobs))
	var wg sync.WaitGroup
	sem := make(chan struct{}, 12)
	for i, j := range jobs {
		if j.idx == "" {
			continue
		}
		wg.Add(1)
		go func(i int, idx string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			body, err := fetch(client, apiSpellURL+idx)
			if err != nil {
				fmt.Printf("skip api %s: %v\n", idx, err)
				return
			}
			var sp apiSpell
			if err := json.Unmarshal(body, &sp); err != nil {
				fmt.Printf("parse api %s: %v\n", idx, err)
				return
			}
			details[i] = &sp
		}(i, j.idx)
	}
	wg.Wait()

	var rows []seedRow
	for i, j := range jobs {
		row := buildRow(j.idx, j.card, details[i])
		if row.Slug == "" {
			continue
		}
		rows = append(rows, row)
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Level != rows[j].Level {
			return rows[i].Level < rows[j].Level
		}
		return rows[i].NameEN < rows[j].NameEN
	})
	fmt.Printf("seed rows: %d\n", len(rows))

	if err := os.WriteFile(outPath, []byte(renderSQL(rows)), 0o644); err != nil {
		panic(err)
	}
	fmt.Println("wrote", outPath)
}

func buildRow(apiIndex string, c *card, api *apiSpell) seedRow {
	r := seedRow{
		DamageAtSlot:      "{}",
		DamageAtCharacter: "{}",
	}
	if api != nil {
		r.Slug = api.Index
		r.NameEN = api.Name
		r.Level = api.Level
		r.School = strings.ToLower(api.School.Name)
		r.Ritual = api.Ritual
		r.Concentration = api.Concentration
		r.CastingTime = api.CastingTime
		r.Range = api.Range
		r.Duration = api.Duration
		r.Components = strings.Join(api.Components, ",")
		for _, cl := range api.Classes {
			r.Classes = append(r.Classes, cl.Index)
		}
		r.SourceURL = "https://www.dnd5eapi.co/api/2014/spells/" + api.Index
		if api.Damage != nil {
			if api.Damage.DamageType != nil {
				r.DamageType = strings.ToLower(api.Damage.DamageType.Index)
			}
			if len(api.Damage.AtSlot) > 0 {
				r.ScaleKind = "slot"
				r.DamageAtSlot = mustJSON(api.Damage.AtSlot)
				r.DamageFormula = pickMap(api.Damage.AtSlot, strconv.Itoa(api.Level), true)
				if len(api.Damage.AtSlot) > 1 || len(api.HigherLevel) > 0 {
					r.Upcast = true
				}
			}
			if len(api.Damage.AtCharacter) > 0 {
				r.ScaleKind = "character"
				r.DamageAtCharacter = mustJSON(api.Damage.AtCharacter)
				r.DamageFormula = pickMap(api.Damage.AtCharacter, "1", true)
			}
		}
		if len(api.HealAtSlot) > 0 {
			r.HealFormula = pickMap(api.HealAtSlot, strconv.Itoa(api.Level), true)
			if r.ScaleKind == "" {
				r.ScaleKind = "slot"
				r.DamageAtSlot = mustJSON(api.HealAtSlot)
			}
			if len(api.HealAtSlot) > 1 || len(api.HigherLevel) > 0 {
				r.Upcast = true
			}
		}
		if api.Level > 0 && len(api.HigherLevel) > 0 {
			r.Upcast = true
		}
		if api.Level > 0 {
			r.Upcast = true
		}
	} else if c != nil {
		r.Slug = slugFromLink(c.Link)
		if r.Slug == "" {
			r.Slug = kebab(c.TitleEN)
		}
		r.NameEN = titleCase(c.TitleEN)
		r.Level, _ = strconv.Atoi(c.Level)
		r.School = schoolEN(c.School)
		r.CastingTime = firstString(c.CastTime)
		r.Components = strings.TrimSpace(c.ItemSfx)
		r.Ritual = ritualYes(c.Ritual)
		r.Concentration = concYes(c.Conc)
		if r.Level > 0 {
			r.Upcast = true
		}
	} else {
		return seedRow{}
	}
	if c != nil {
		r.NameRU = c.Title
		r.SourceURLRU = "https://5e14.dnd.su" + c.Link
		if r.NameEN == "" {
			r.NameEN = titleCase(c.TitleEN)
		}
	}
	if r.NameRU == "" {
		r.NameRU = r.NameEN
	}
	if r.Slug == "" {
		return seedRow{}
	}
	return r
}

func renderSQL(rows []seedRow) string {
	var b strings.Builder
	b.WriteString(`-- Spells + character resource/spell tables.
-- 5e14 IDs discovered from GET https://5e14.dnd.su/piece/spells/index-list/ (the /spells/ index).
-- 5eapi 2014 URLs stored only when the detail endpoint returned 200.
-- Mechanical stats only; no lore article text.
-- spell_count: `)
	b.WriteString(strconv.Itoa(len(rows)))
	b.WriteString("\n\n")
	b.WriteString(`ALTER TABLE catalog_spells ADD COLUMN damage_formula TEXT NOT NULL DEFAULT '';
ALTER TABLE catalog_spells ADD COLUMN damage_type TEXT NOT NULL DEFAULT '';
ALTER TABLE catalog_spells ADD COLUMN heal_formula TEXT NOT NULL DEFAULT '';
ALTER TABLE catalog_spells ADD COLUMN scale_kind TEXT NOT NULL DEFAULT '';
ALTER TABLE catalog_spells ADD COLUMN upcast INTEGER NOT NULL DEFAULT 0;
ALTER TABLE catalog_spells ADD COLUMN concentration INTEGER NOT NULL DEFAULT 0;
ALTER TABLE catalog_spells ADD COLUMN damage_at_slot TEXT NOT NULL DEFAULT '{}';
ALTER TABLE catalog_spells ADD COLUMN damage_at_character TEXT NOT NULL DEFAULT '{}';

CREATE TABLE IF NOT EXISTS character_spells (
    character_id INTEGER NOT NULL REFERENCES characters(id) ON DELETE CASCADE,
    spell_id INTEGER NOT NULL REFERENCES catalog_spells(id),
    prepared INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (character_id, spell_id)
);

CREATE TABLE IF NOT EXISTS character_resources (
    character_id INTEGER NOT NULL REFERENCES characters(id) ON DELETE CASCADE,
    kind TEXT NOT NULL,
    slot_level INTEGER NOT NULL DEFAULT 0,
    current INTEGER NOT NULL DEFAULT 0,
    max INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (character_id, kind, slot_level)
);

CREATE INDEX IF NOT EXISTS idx_catalog_spells_name ON catalog_spells(name_en);
CREATE INDEX IF NOT EXISTS idx_character_spells_char ON character_spells(character_id);
CREATE INDEX IF NOT EXISTS idx_character_resources_char ON character_resources(character_id);

DELETE FROM catalog_spells;

INSERT INTO catalog_spells (
    id, slug, name_en, name_ru, level, school, ritual, casting_time, spell_range, duration,
    components, classes, source_url, source_url_ru,
    damage_formula, damage_type, heal_formula, scale_kind, upcast, concentration,
    damage_at_slot, damage_at_character
) VALUES
`)
	for i, r := range rows {
		if i > 0 {
			b.WriteString(",\n")
		}
		fmt.Fprintf(&b, "(%d, %s, %s, %s, %d, %s, %d, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %d, %d, %s, %s)",
			i+1,
			q(r.Slug), q(r.NameEN), q(r.NameRU), r.Level, q(r.School),
			btoi(r.Ritual), q(r.CastingTime), q(r.Range), q(r.Duration),
			q(r.Components), q(mustJSON(r.Classes)), q(r.SourceURL), q(r.SourceURLRU),
			q(r.DamageFormula), q(r.DamageType), q(r.HealFormula), q(r.ScaleKind),
			btoi(r.Upcast), btoi(r.Concentration),
			q(r.DamageAtSlot), q(r.DamageAtCharacter),
		)
	}
	b.WriteString(";\n")
	return b.String()
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

func parse14(raw []byte) ([]card, error) {
	s := string(raw)
	i := strings.Index(s, "window.LIST")
	if i < 0 {
		i = strings.Index(s, `{"cards"`)
		if i < 0 {
			return nil, fmt.Errorf("5e14 list: no window.LIST")
		}
	} else {
		eq := strings.Index(s[i:], "=")
		if eq < 0 {
			return nil, fmt.Errorf("5e14 list: no =")
		}
		i = i + eq + 1
	}
	dec := json.NewDecoder(strings.NewReader(strings.TrimSpace(s[i:])))
	var lf listFile
	if err := dec.Decode(&lf); err != nil {
		return nil, err
	}
	return lf.Cards, nil
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

func hasSource(c card, id int) bool {
	for _, s := range c.Sources {
		if s == id {
			return true
		}
	}
	return false
}

func ritualYes(v []any) bool {
	return firstString(v) == "2"
}

func concYes(v []any) bool {
	return firstString(v) == "2"
}

func firstString(v []any) string {
	if len(v) == 0 {
		return ""
	}
	switch t := v[0].(type) {
	case string:
		return t
	case float64:
		return strconv.Itoa(int(t))
	default:
		return fmt.Sprint(t)
	}
}

func schoolEN(ru string) string {
	switch ru {
	case "Воплощение":
		return "evocation"
	case "Вызов":
		return "conjuration"
	case "Иллюзия":
		return "illusion"
	case "Некромантия":
		return "necromancy"
	case "Ограждение":
		return "abjuration"
	case "Очарование":
		return "enchantment"
	case "Преобразование":
		return "transmutation"
	case "Прорицание":
		return "divination"
	default:
		return strings.ToLower(ru)
	}
}

func pickMap(m map[string]string, key string, lowest bool) string {
	if v, ok := m[key]; ok {
		return v
	}
	keys := make([]int, 0, len(m))
	for k := range m {
		n, err := strconv.Atoi(k)
		if err == nil {
			keys = append(keys, n)
		}
	}
	if len(keys) == 0 {
		return ""
	}
	sort.Ints(keys)
	if lowest {
		return m[strconv.Itoa(keys[0])]
	}
	return m[strconv.Itoa(keys[len(keys)-1])]
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
