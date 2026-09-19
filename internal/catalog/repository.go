package catalog

import (
	"database/sql"
	"encoding/json"
	"strconv"
	"strings"
)

// Repository is SQLite access for catalog tables.
type Repository struct {
	DB *sql.DB
}

func (r *Repository) ListRaces() ([]Race, error) {
	rows, err := r.DB.Query(`
		SELECT id, slug, name_en, name_ru, source_url, source_url_ru, ability_bonuses, speed, hp_bonus_per_level
		FROM catalog_races ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Race
	for rows.Next() {
		race, err := scanRace(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *race)
	}
	return out, rows.Err()
}

func (r *Repository) Race(id int64) (*Race, error) {
	return scanRace(r.DB.QueryRow(`
		SELECT id, slug, name_en, name_ru, source_url, source_url_ru, ability_bonuses, speed, hp_bonus_per_level
		FROM catalog_races WHERE id = ?`, id))
}

func (r *Repository) ListBackgrounds() ([]Background, error) {
	rows, err := r.DB.Query(`
		SELECT id, slug, name_en, name_ru, source_url, source_url_ru, ability_bonuses
		FROM catalog_backgrounds ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Background
	for rows.Next() {
		bg, err := scanBackground(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *bg)
	}
	return out, rows.Err()
}

func (r *Repository) Background(id int64) (*Background, error) {
	return scanBackground(r.DB.QueryRow(`
		SELECT id, slug, name_en, name_ru, source_url, source_url_ru, ability_bonuses
		FROM catalog_backgrounds WHERE id = ?`, id))
}

func (r *Repository) ListClasses() ([]Class, error) {
	rows, err := r.DB.Query(`
		SELECT id, slug, name_en, name_ru, hit_die, subclass_level, asi_levels, source_url, source_url_ru, ability_bonuses
		FROM catalog_classes ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Class
	for rows.Next() {
		c, err := scanClass(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *c)
	}
	return out, rows.Err()
}

func (r *Repository) Class(id int64) (*Class, error) {
	return scanClass(r.DB.QueryRow(`
		SELECT id, slug, name_en, name_ru, hit_die, subclass_level, asi_levels, source_url, source_url_ru, ability_bonuses
		FROM catalog_classes WHERE id = ?`, id))
}

func (r *Repository) ListSubclasses(classID int64) ([]Subclass, error) {
	rows, err := r.DB.Query(`
		SELECT id, class_id, slug, name_en, name_ru, source_url, source_url_ru
		FROM catalog_subclasses WHERE class_id = ? ORDER BY id`, classID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Subclass
	for rows.Next() {
		s, err := scanSubclass(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *s)
	}
	return out, rows.Err()
}

func (r *Repository) Subclass(id int64) (*Subclass, error) {
	return scanSubclass(r.DB.QueryRow(`
		SELECT id, class_id, slug, name_en, name_ru, source_url, source_url_ru
		FROM catalog_subclasses WHERE id = ?`, id))
}

func (r *Repository) FeaturesAt(kind string, sourceID int64, level int) ([]Feature, error) {
	rows, err := r.DB.Query(`
		SELECT id, source_kind, source_id, level, slug, name_en, name_ru, source_url, source_url_ru
		FROM catalog_features
		WHERE source_kind = ? AND source_id = ? AND level = ?
		ORDER BY id`, kind, sourceID, level)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Feature
	for rows.Next() {
		f, err := scanFeature(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *f)
	}
	return out, rows.Err()
}

func (r *Repository) Feature(id int64) (*Feature, error) {
	return scanFeature(r.DB.QueryRow(`
		SELECT id, source_kind, source_id, level, slug, name_en, name_ru, source_url, source_url_ru
		FROM catalog_features WHERE id = ?`, id))
}

const itemSelect = `
		SELECT id, slug, name_en, name_ru, kind, cost_gp, weight_lb, damage_dice, damage_type,
		       armor_class, properties, rarity, source_url, source_url_ru, desc_en, armor_category,
		       ac_base, dex_max, stealth_disadv, str_min, weapon_category, versatile_dice,
		       range_normal, range_long, suggested_slot, requires_attunement, consumable,
		       charges_max, is_stub, desc_ru, is_base
		FROM catalog_items`

func (r *Repository) ListItems() ([]Item, error) {
	rows, err := r.DB.Query(itemSelect + ` ORDER BY kind, name_en`)
	if err != nil {
		return nil, err
	}
	return scanItems(rows)
}

func (r *Repository) Item(id int64) (*Item, error) {
	return scanItem(r.DB.QueryRow(itemSelect+` WHERE id = ?`, id))
}

func (r *Repository) ItemBySlug(slug string) (*Item, error) {
	return scanItem(r.DB.QueryRow(itemSelect+` WHERE slug = ?`, slug))
}

func (r *Repository) ListBaseWeapons() ([]Item, error) {
	return r.listBases("weapon")
}

func (r *Repository) ListBaseArmor() ([]Item, error) {
	return r.listBases("armor")
}

func (r *Repository) ListBaseJewelry() ([]Item, error) {
	return r.listBases("jewelry")
}

func (r *Repository) listBases(kind string) ([]Item, error) {
	rows, err := r.DB.Query(itemSelect + ` WHERE is_base = 1 ORDER BY kind, armor_category, weapon_category, name_en`)
	if err != nil {
		return nil, err
	}
	items, err := scanItems(rows)
	if err != nil {
		return nil, err
	}
	var out []Item
	for _, it := range items {
		if baseMatchesKind(it, kind) {
			out = append(out, it)
		}
	}
	return out, nil
}

func baseMatchesKind(it Item, kind string) bool {
	switch kind {
	case "armor":
		return it.IsArmor()
	case "jewelry":
		return it.IsJewelry()
	default:
		return it.IsWeaponBase()
	}
}

func (r *Repository) SearchBaseWeapons(q string, limit int) ([]Item, error) {
	return r.SearchBases("weapon", q, limit)
}

func (r *Repository) SearchBaseArmor(q string, limit int) ([]Item, error) {
	return r.SearchBases("armor", q, limit)
}

func (r *Repository) SearchBaseJewelry(q string, limit int) ([]Item, error) {
	return r.SearchBases("jewelry", q, limit)
}

func (r *Repository) SearchBases(kind, q string, limit int) ([]Item, error) {
	if limit <= 0 || limit > 40 {
		limit = 40
	}
	items, err := r.listBases(kind)
	if err != nil {
		return nil, err
	}
	needle := strings.ToLower(strings.TrimSpace(q))
	if needle == "" {
		if len(items) > limit {
			return items[:limit], nil
		}
		return items, nil
	}
	var out []Item
	for _, it := range items {
		if matchesFold(needle, it.NameEN, it.NameRU, it.Slug) {
			out = append(out, it)
			if len(out) >= limit {
				break
			}
		}
	}
	return out, nil
}

func (r *Repository) SearchConsumables(q string, limit int) ([]Item, error) {
	if limit <= 0 || limit > 40 {
		limit = 20
	}
	needle := strings.ToLower(strings.TrimSpace(q))
	if needle == "" {
		return nil, nil
	}
	rows, err := r.DB.Query(itemSelect + ` WHERE consumable = 1 ORDER BY name_en`)
	if err != nil {
		return nil, err
	}
	items, err := scanItems(rows)
	if err != nil {
		return nil, err
	}
	var out []Item
	for _, it := range items {
		if matchesFold(needle, it.NameEN, it.NameRU, it.Slug) {
			out = append(out, it)
			if len(out) >= limit {
				break
			}
		}
	}
	return out, nil
}

func (r *Repository) ListStatFeatures() ([]StatFeature, error) {
	rows, err := r.DB.Query(`
		SELECT id, slug, name_en, name_ru, stat, default_value, origin, source_url, sort_order
		FROM catalog_stat_features ORDER BY sort_order, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []StatFeature
	for rows.Next() {
		f, err := scanStatFeature(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

func (r *Repository) StatFeature(id int64) (*StatFeature, error) {
	f, err := scanStatFeature(r.DB.QueryRow(`
		SELECT id, slug, name_en, name_ru, stat, default_value, origin, source_url, sort_order
		FROM catalog_stat_features WHERE id = ?`, id))
	if err != nil {
		return nil, err
	}
	return &f, nil
}

func (r *Repository) SearchStatFeatures(q string, limit int) ([]StatFeature, error) {
	if limit <= 0 || limit > 40 {
		limit = 20
	}
	all, err := r.ListStatFeatures()
	if err != nil {
		return nil, err
	}
	needle := strings.ToLower(strings.TrimSpace(q))
	if needle == "" {
		return nil, nil
	}
	var out []StatFeature
	for _, f := range all {
		if matchesFold(needle, f.NameEN, f.NameRU, f.Slug, f.Stat) {
			out = append(out, f)
			if len(out) >= limit {
				break
			}
		}
	}
	return out, nil
}

func scanStatFeature(row scanner) (StatFeature, error) {
	var f StatFeature
	err := row.Scan(&f.ID, &f.Slug, &f.NameEN, &f.NameRU, &f.Stat, &f.DefaultValue, &f.Origin, &f.SourceURL, &f.SortOrder)
	return f, err
}

func (r *Repository) LootMagicByRarity(rarity string) ([]Item, error) {
	want := NormalizeRarity(rarity)
	if want == "" {
		return nil, nil
	}
	items, err := r.ListItems()
	if err != nil {
		return nil, err
	}
	var out []Item
	for _, it := range items {
		if it.LootMagicEligible() && NormalizeRarity(it.Rarity) == want {
			out = append(out, it)
		}
	}
	return out, nil
}

func (r *Repository) SpellsInLevelRange(min, max int) ([]Spell, error) {
	spells, err := r.ListSpells()
	if err != nil {
		return nil, err
	}
	if min < 0 {
		min = 0
	}
	if max < min {
		return nil, nil
	}
	var out []Spell
	for _, sp := range spells {
		if sp.Level >= min && sp.Level <= max {
			out = append(out, sp)
		}
	}
	return out, nil
}

func (r *Repository) SearchItems(q string, limit int) ([]Item, error) {
	if limit <= 0 || limit > 40 {
		limit = 20
	}
	needle := strings.ToLower(strings.TrimSpace(q))
	if needle == "" {
		return nil, nil
	}
	rows, err := r.DB.Query(itemSelect + ` ORDER BY is_stub, name_en`)
	if err != nil {
		return nil, err
	}
	items, err := scanItems(rows)
	if err != nil {
		return nil, err
	}
	var out []Item
	for _, it := range items {
		if matchesFold(needle, it.NameEN, it.NameRU, it.Slug) {
			out = append(out, it)
			if len(out) >= limit {
				break
			}
		}
	}
	return out, nil
}

func scanItems(rows *sql.Rows) ([]Item, error) {
	defer rows.Close()
	var out []Item
	for rows.Next() {
		item, err := scanItem(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *item)
	}
	return out, rows.Err()
}

const spellSelect = `
		SELECT id, slug, name_en, name_ru, level, school, ritual, casting_time, spell_range, duration,
		       components, classes, source_url, source_url_ru,
		       damage_formula, damage_type, heal_formula, scale_kind, upcast, concentration,
		       damage_at_slot, damage_at_character
		FROM catalog_spells`

func (r *Repository) ListSpells() ([]Spell, error) {
	rows, err := r.DB.Query(spellSelect + ` ORDER BY level, name_en`)
	if err != nil {
		return nil, err
	}
	return scanSpells(rows)
}

func (r *Repository) Spell(id int64) (*Spell, error) {
	return scanSpell(r.DB.QueryRow(spellSelect+` WHERE id = ?`, id))
}

func (r *Repository) SpellBySlug(slug string) (*Spell, error) {
	return scanSpell(r.DB.QueryRow(spellSelect+` WHERE slug = ?`, slug))
}

func (r *Repository) SearchSpells(q string, limit int) ([]Spell, error) {
	if limit <= 0 || limit > 40 {
		limit = 20
	}
	needle := strings.ToLower(strings.TrimSpace(q))
	if needle == "" {
		return nil, nil
	}
	rows, err := r.DB.Query(spellSelect + ` ORDER BY level, name_en`)
	if err != nil {
		return nil, err
	}
	spells, err := scanSpells(rows)
	if err != nil {
		return nil, err
	}
	var out []Spell
	for _, sp := range spells {
		if matchesFold(needle, sp.NameEN, sp.NameRU, sp.Slug) {
			out = append(out, sp)
			if len(out) >= limit {
				break
			}
		}
	}
	return out, nil
}

func matchesFold(needle string, fields ...string) bool {
	for _, f := range fields {
		if strings.Contains(strings.ToLower(f), needle) {
			return true
		}
	}
	return false
}

func monsterTypeMatches(stored, want string) bool {
	stored = strings.ToLower(strings.TrimSpace(stored))
	want = strings.ToLower(strings.TrimSpace(want))
	if stored == "" || want == "" {
		return false
	}
	if stored == want {
		return true
	}
	return strings.HasPrefix(stored, want+" ") || strings.HasPrefix(stored, want+"(") || strings.HasPrefix(stored, want+" (")
}

func scanSpells(rows *sql.Rows) ([]Spell, error) {
	defer rows.Close()
	var out []Spell
	for rows.Next() {
		sp, err := scanSpell(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *sp)
	}
	return out, rows.Err()
}

type scanner interface {
	Scan(dest ...any) error
}

func scanRace(row scanner) (*Race, error) {
	race := &Race{}
	var bonusJSON string
	if err := row.Scan(&race.ID, &race.Slug, &race.NameEN, &race.NameRU, &race.SourceURL, &race.SourceURLRU,
		&bonusJSON, &race.Speed, &race.HPBonusPerLevel); err != nil {
		return nil, err
	}
	race.AbilityBonuses = parseBonusMap(bonusJSON)
	return race, nil
}

func scanBackground(row scanner) (*Background, error) {
	bg := &Background{}
	var bonusJSON string
	if err := row.Scan(&bg.ID, &bg.Slug, &bg.NameEN, &bg.NameRU, &bg.SourceURL, &bg.SourceURLRU, &bonusJSON); err != nil {
		return nil, err
	}
	bg.AbilityBonuses = parseBonusMap(bonusJSON)
	return bg, nil
}

func scanClass(row scanner) (*Class, error) {
	c := &Class{}
	var asiJSON, bonusJSON string
	if err := row.Scan(&c.ID, &c.Slug, &c.NameEN, &c.NameRU, &c.HitDie, &c.SubclassLevel, &asiJSON,
		&c.SourceURL, &c.SourceURLRU, &bonusJSON); err != nil {
		return nil, err
	}
	c.ASILevels = parseIntSlice(asiJSON)
	c.AbilityBonuses = parseBonusMap(bonusJSON)
	return c, nil
}

func scanSubclass(row scanner) (*Subclass, error) {
	s := &Subclass{}
	if err := row.Scan(&s.ID, &s.ClassID, &s.Slug, &s.NameEN, &s.NameRU, &s.SourceURL, &s.SourceURLRU); err != nil {
		return nil, err
	}
	return s, nil
}

func scanFeature(row scanner) (*Feature, error) {
	f := &Feature{}
	if err := row.Scan(&f.ID, &f.SourceKind, &f.SourceID, &f.Level, &f.Slug, &f.NameEN, &f.NameRU, &f.SourceURL, &f.SourceURLRU); err != nil {
		return nil, err
	}
	return f, nil
}

func parseBonusMap(s string) map[string]int {
	out := map[string]int{}
	if s == "" {
		return out
	}
	_ = json.Unmarshal([]byte(s), &out)
	if out == nil {
		return map[string]int{}
	}
	return out
}

func parseIntSlice(s string) []int {
	var out []int
	if s == "" {
		return out
	}
	_ = json.Unmarshal([]byte(s), &out)
	return out
}

func scanItem(row scanner) (*Item, error) {
	it := &Item{}
	var propsJSON string
	var stealth, attune, cons, stub, base int
	if err := row.Scan(&it.ID, &it.Slug, &it.NameEN, &it.NameRU, &it.Kind, &it.CostGP, &it.WeightLB,
		&it.DamageDice, &it.DamageType, &it.ArmorClass, &propsJSON, &it.Rarity, &it.SourceURL, &it.SourceURLRU,
		&it.DescEN, &it.ArmorCategory, &it.ACBase, &it.DexMax, &stealth, &it.StrMin, &it.WeaponCategory,
		&it.VersatileDice, &it.RangeNormal, &it.RangeLong, &it.SuggestedSlot, &attune, &cons, &it.ChargesMax, &stub,
		&it.DescRU, &base); err != nil {
		return nil, err
	}
	it.Properties = parseStringSlice(propsJSON)
	it.StealthDisadv = stealth != 0
	it.RequiresAttunement = attune != 0
	it.Consumable = cons != 0
	it.IsStub = stub != 0
	it.IsBase = base != 0
	return it, nil
}

func scanSpell(row scanner) (*Spell, error) {
	sp := &Spell{}
	var ritual, upcast, conc int
	var classesJSON, slotJSON, charJSON string
	if err := row.Scan(&sp.ID, &sp.Slug, &sp.NameEN, &sp.NameRU, &sp.Level, &sp.School, &ritual,
		&sp.CastingTime, &sp.Range, &sp.Duration, &sp.Components, &classesJSON, &sp.SourceURL, &sp.SourceURLRU,
		&sp.DamageFormula, &sp.DamageType, &sp.HealFormula, &sp.ScaleKind, &upcast, &conc,
		&slotJSON, &charJSON); err != nil {
		return nil, err
	}
	sp.Ritual = ritual != 0
	sp.Upcast = upcast != 0
	sp.Concentration = conc != 0
	sp.Classes = parseStringSlice(classesJSON)
	sp.DamageAtSlot = parseStringMap(slotJSON)
	sp.DamageAtCharacter = parseStringMap(charJSON)
	return sp, nil
}

func parseStringMap(s string) map[string]string {
	out := map[string]string{}
	if s == "" {
		return out
	}
	_ = json.Unmarshal([]byte(s), &out)
	if out == nil {
		return map[string]string{}
	}
	return out
}

func parseStringSlice(s string) []string {
	var out []string
	if s == "" {
		return out
	}
	_ = json.Unmarshal([]byte(s), &out)
	if out == nil {
		return []string{}
	}
	return out
}

const monsterSelect = `
		SELECT id, slug, name_en, name_ru, size, type, armor_class, hit_points, hit_dice, speed,
		       dexterity, cr, cr_label, xp, damage_resistances, damage_immunities, damage_vulnerabilities,
		       condition_immunities, actions, desc_en, source_url, source_url_ru, is_stub
		FROM catalog_monsters`

func (r *Repository) ListMonsters() ([]Monster, error) {
	rows, err := r.DB.Query(monsterSelect + ` ORDER BY name_en`)
	if err != nil {
		return nil, err
	}
	return scanMonsters(rows)
}

func (r *Repository) Monster(id int64) (*Monster, error) {
	return scanMonster(r.DB.QueryRow(monsterSelect+` WHERE id = ?`, id))
}

func (r *Repository) MonsterBySlug(slug string) (*Monster, error) {
	return scanMonster(r.DB.QueryRow(monsterSelect+` WHERE slug = ?`, slug))
}

func (r *Repository) SearchMonsters(q, cr, typ string, limit int) ([]Monster, error) {
	return r.SearchMonstersFilter(q, cr, typ, "", limit)
}

func (r *Repository) SearchMonstersFilter(q, cr, typ, size string, limit int) ([]Monster, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 500 {
		limit = 500
	}
	needle := strings.ToLower(strings.TrimSpace(q))
	cr = strings.TrimSpace(cr)
	typ = strings.TrimSpace(typ)
	size = strings.TrimSpace(size)
	if needle == "" && cr == "" && typ == "" && size == "" {
		return nil, nil
	}
	rows, err := r.DB.Query(monsterSelect + ` ORDER BY is_stub, cr, name_en`)
	if err != nil {
		return nil, err
	}
	all, err := scanMonsters(rows)
	if err != nil {
		return nil, err
	}
	var out []Monster
	for _, m := range all {
		if cr != "" && m.CRLabel != cr && fmtCR(m.CR) != cr {
			continue
		}
		if typ != "" && !monsterTypeMatches(m.Type, typ) {
			continue
		}
		if size != "" && !strings.EqualFold(strings.TrimSpace(m.Size), size) {
			continue
		}
		if needle != "" && !matchesFold(needle, m.NameEN, m.NameRU, m.Slug, m.Type) {
			continue
		}
		out = append(out, m)
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

func (r *Repository) ListMonsterCRs() ([]string, error) {
	rows, err := r.DB.Query(`SELECT DISTINCT cr_label, cr FROM catalog_monsters ORDER BY cr`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	seen := map[string]bool{}
	for rows.Next() {
		var label string
		var cr float64
		if err := rows.Scan(&label, &cr); err != nil {
			return nil, err
		}
		if label == "" || seen[label] {
			continue
		}
		seen[label] = true
		out = append(out, label)
	}
	return out, rows.Err()
}

func (r *Repository) ListMonsterTypes() ([]string, error) {
	rows, err := r.DB.Query(`SELECT DISTINCT type FROM catalog_monsters WHERE type != '' ORDER BY type COLLATE NOCASE`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	seen := map[string]bool{}
	for rows.Next() {
		var typ string
		if err := rows.Scan(&typ); err != nil {
			return nil, err
		}
		typ = strings.TrimSpace(typ)
		if typ == "" || seen[strings.ToLower(typ)] {
			continue
		}
		seen[strings.ToLower(typ)] = true
		out = append(out, typ)
	}
	return out, rows.Err()
}

const conditionSelect = `
		SELECT id, slug, name_en, name_ru, source_url, source_url_ru, damage_formula, damage_type, is_phb
		FROM catalog_conditions`

func (r *Repository) ListConditions() ([]Condition, error) {
	rows, err := r.DB.Query(conditionSelect + ` ORDER BY is_phb DESC, name_en`)
	if err != nil {
		return nil, err
	}
	return scanConditions(rows)
}

func (r *Repository) Condition(id int64) (*Condition, error) {
	return scanCondition(r.DB.QueryRow(conditionSelect+` WHERE id = ?`, id))
}

func (r *Repository) ConditionBySlug(slug string) (*Condition, error) {
	return scanCondition(r.DB.QueryRow(conditionSelect+` WHERE slug = ?`, slug))
}

func (r *Repository) SearchConditions(q string, limit int) ([]Condition, error) {
	if limit <= 0 || limit > 40 {
		limit = 20
	}
	all, err := r.ListConditions()
	if err != nil {
		return nil, err
	}
	needle := strings.ToLower(strings.TrimSpace(q))
	if needle == "" {
		if len(all) > limit {
			return all[:limit], nil
		}
		return all, nil
	}
	var out []Condition
	for _, c := range all {
		if matchesFold(needle, c.NameEN, c.NameRU, c.Slug) {
			out = append(out, c)
			if len(out) >= limit {
				break
			}
		}
	}
	return out, nil
}

func scanConditions(rows *sql.Rows) ([]Condition, error) {
	defer rows.Close()
	var out []Condition
	for rows.Next() {
		c, err := scanCondition(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *c)
	}
	return out, rows.Err()
}

func scanCondition(row scanner) (*Condition, error) {
	c := &Condition{}
	var phb int
	if err := row.Scan(&c.ID, &c.Slug, &c.NameEN, &c.NameRU, &c.SourceURL, &c.SourceURLRU,
		&c.DamageFormula, &c.DamageType, &phb); err != nil {
		return nil, err
	}
	c.IsPHB = phb != 0
	return c, nil
}

func fmtCR(v float64) string {
	return strconv.FormatFloat(v, 'g', -1, 64)
}

func scanMonsters(rows *sql.Rows) ([]Monster, error) {
	defer rows.Close()
	var out []Monster
	for rows.Next() {
		m, err := scanMonster(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *m)
	}
	return out, rows.Err()
}

func scanMonster(row scanner) (*Monster, error) {
	m := &Monster{}
	var resist, immune, vuln, cond string
	var stub int
	if err := row.Scan(&m.ID, &m.Slug, &m.NameEN, &m.NameRU, &m.Size, &m.Type, &m.ArmorClass, &m.HitPoints,
		&m.HitDice, &m.Speed, &m.Dexterity, &m.CR, &m.CRLabel, &m.XP, &resist, &immune, &vuln, &cond,
		&m.ActionsJSON, &m.DescEN, &m.SourceURL, &m.SourceURLRU, &stub); err != nil {
		return nil, err
	}
	m.Resistances = parseStringSlice(resist)
	m.Immunities = parseStringSlice(immune)
	m.Vulnerabilities = parseStringSlice(vuln)
	m.ConditionImmunities = parseStringSlice(cond)
	m.IsStub = stub != 0
	return m, nil
}
