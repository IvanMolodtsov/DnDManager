package catalog

import (
	"database/sql"
	"encoding/json"
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

func (r *Repository) ListItems() ([]Item, error) {
	rows, err := r.DB.Query(`
		SELECT id, slug, name_en, name_ru, kind, cost_gp, weight_lb, damage_dice, damage_type,
		       armor_class, properties, rarity, source_url, source_url_ru
		FROM catalog_items ORDER BY id`)
	if err != nil {
		return nil, err
	}
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

func (r *Repository) Item(id int64) (*Item, error) {
	return scanItem(r.DB.QueryRow(`
		SELECT id, slug, name_en, name_ru, kind, cost_gp, weight_lb, damage_dice, damage_type,
		       armor_class, properties, rarity, source_url, source_url_ru
		FROM catalog_items WHERE id = ?`, id))
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

func (r *Repository) SearchSpells(q string, limit int) ([]Spell, error) {
	if limit <= 0 || limit > 40 {
		limit = 20
	}
	q = "%" + q + "%"
	rows, err := r.DB.Query(spellSelect+`
		WHERE name_en LIKE ? OR name_ru LIKE ? OR slug LIKE ?
		ORDER BY level, name_en LIMIT ?`, q, q, q, limit)
	if err != nil {
		return nil, err
	}
	return scanSpells(rows)
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
	if err := row.Scan(&it.ID, &it.Slug, &it.NameEN, &it.NameRU, &it.Kind, &it.CostGP, &it.WeightLB,
		&it.DamageDice, &it.DamageType, &it.ArmorClass, &propsJSON, &it.Rarity, &it.SourceURL, &it.SourceURLRU); err != nil {
		return nil, err
	}
	it.Properties = parseStringSlice(propsJSON)
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
