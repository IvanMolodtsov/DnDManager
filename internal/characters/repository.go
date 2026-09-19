package characters

import (
	"database/sql"
	"strings"
	"time"

	"dndmanager/internal/rules"
)

// Repository is SQLite access for live characters, class levels, features, and drafts.
type Repository struct {
	DB *sql.DB
}

func (r *Repository) FindByID(id int64) (*Character, error) {
	return scanCharacter(r.DB.QueryRow(characterSelect+` WHERE ch.id = ?`, id))
}

func (r *Repository) ListByOwner(ownerID int64) ([]Character, error) {
	rows, err := r.DB.Query(characterSelect+`
		WHERE ch.owner_id = ?
		ORDER BY ch.created_at DESC`, ownerID)
	if err != nil {
		return nil, err
	}
	return scanCharacters(rows)
}

func (r *Repository) ListByCampaign(campaignID int64) ([]Character, error) {
	rows, err := r.DB.Query(characterSelect+`
		WHERE ch.campaign_id = ?
		ORDER BY ch.name`, campaignID)
	if err != nil {
		return nil, err
	}
	return scanCharacters(rows)
}

const characterSelect = `
		SELECT ch.id, ch.name, ch.owner_id, u.username, ch.campaign_id, c.name, ch.level,
		       ch.str, ch.dex, ch.con, ch.intel, ch.wis, ch.cha, ch.created_at,
		       ch.race_id, ch.background_id, ch.hp_max, ch.hp_current, ch.hp_temp,
		       ch.death_success, ch.death_fail, ch.proficiency_bonus, ch.gold, c.souls, c.souls_cap
		FROM characters ch
		JOIN users u ON u.id = ch.owner_id
		JOIN campaigns c ON c.id = ch.campaign_id`

func (r *Repository) InsertLive(ch *Character, classes []rules.ClassProgress, featureIDs []int64) (int64, error) {
	tx, err := r.DB.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	res, err := tx.Exec(`
		INSERT INTO characters (
			name, owner_id, campaign_id, level, str, dex, con, intel, wis, cha,
			race_id, background_id, hp_max, hp_current, proficiency_bonus
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		ch.Name, ch.OwnerID, ch.CampaignID, ch.Level, ch.STR, ch.DEX, ch.CON, ch.INT, ch.WIS, ch.CHA,
		nullIfZero(ch.RaceID), nullIfZero(ch.BackgroundID), ch.HPMax, ch.HPCurrent, ch.ProficiencyBonus,
	)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	if err := replaceClasses(tx, id, classes); err != nil {
		return 0, err
	}
	for _, fid := range featureIDs {
		if _, err := tx.Exec(`INSERT OR IGNORE INTO character_features (character_id, feature_id) VALUES (?, ?)`, id, fid); err != nil {
			return 0, err
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return id, nil
}

func (r *Repository) ApplyProgress(id int64, ch *Character, classes []rules.ClassProgress, newFeatureIDs []int64) error {
	tx, err := r.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`
		UPDATE characters SET
			level = ?, str = ?, dex = ?, con = ?, intel = ?, wis = ?, cha = ?,
			hp_max = ?, hp_current = ?, proficiency_bonus = ?
		WHERE id = ?`,
		ch.Level, ch.STR, ch.DEX, ch.CON, ch.INT, ch.WIS, ch.CHA,
		ch.HPMax, ch.HPCurrent, ch.ProficiencyBonus, id,
	); err != nil {
		return err
	}
	if err := replaceClasses(tx, id, classes); err != nil {
		return err
	}
	for _, fid := range newFeatureIDs {
		if _, err := tx.Exec(`INSERT OR IGNORE INTO character_features (character_id, feature_id) VALUES (?, ?)`, id, fid); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func replaceClasses(tx *sql.Tx, characterID int64, classes []rules.ClassProgress) error {
	if _, err := tx.Exec(`DELETE FROM character_class_levels WHERE character_id = ?`, characterID); err != nil {
		return err
	}
	for _, cl := range classes {
		if _, err := tx.Exec(
			`INSERT INTO character_class_levels (character_id, class_id, subclass_id, levels) VALUES (?, ?, ?, ?)`,
			characterID, cl.ClassID, nullIfZero(cl.SubclassID), cl.Levels,
		); err != nil {
			return err
		}
	}
	return nil
}

func (r *Repository) ListClassLevels(characterID int64) ([]struct {
	ClassID    int64
	SubclassID int64
	Levels     int
}, error) {
	rows, err := r.DB.Query(
		`SELECT class_id, subclass_id, levels FROM character_class_levels WHERE character_id = ? ORDER BY class_id`,
		characterID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []struct {
		ClassID    int64
		SubclassID int64
		Levels     int
	}
	for rows.Next() {
		var classID, levels int64
		var subclass sql.NullInt64
		if err := rows.Scan(&classID, &subclass, &levels); err != nil {
			return nil, err
		}
		out = append(out, struct {
			ClassID    int64
			SubclassID int64
			Levels     int
		}{ClassID: classID, SubclassID: subclass.Int64, Levels: int(levels)})
	}
	return out, rows.Err()
}

func (r *Repository) ListFeatureIDs(characterID int64) ([]int64, error) {
	rows, err := r.DB.Query(`SELECT feature_id FROM character_features WHERE character_id = ? ORDER BY feature_id`, characterID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

func (r *Repository) GetDraft(ownerID, campaignID int64) (*Draft, error) {
	d := &Draft{}
	var race, bg, class, sub sql.NullInt64
	var scores string
	err := r.DB.QueryRow(`
		SELECT id, owner_id, campaign_id, step, name, race_id, background_id, class_id, subclass_id, abilities_json
		FROM character_drafts WHERE owner_id = ? AND campaign_id = ?`, ownerID, campaignID,
	).Scan(&d.ID, &d.OwnerID, &d.CampaignID, &d.Step, &d.Name, &race, &bg, &class, &sub, &scores)
	if err != nil {
		return nil, err
	}
	d.RaceID, d.BackgroundID, d.ClassID, d.SubclassID = race.Int64, bg.Int64, class.Int64, sub.Int64
	d.BaseScores, d.HasScores = parseScoresJSON(scores)
	if d.Step < 1 {
		d.Step = 1
	}
	return d, nil
}

func (r *Repository) InsertDraft(d *Draft) (int64, error) {
	res, err := r.DB.Exec(`
		INSERT INTO character_drafts (owner_id, campaign_id, step, name, race_id, background_id, class_id, subclass_id, abilities_json)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		d.OwnerID, d.CampaignID, d.Step, d.Name, nullIfZero(d.RaceID), nullIfZero(d.BackgroundID),
		nullIfZero(d.ClassID), nullIfZero(d.SubclassID), d.ScoresJSON(),
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (r *Repository) UpdateDraft(d *Draft) error {
	_, err := r.DB.Exec(`
		UPDATE character_drafts SET
			step = ?, name = ?, race_id = ?, background_id = ?, class_id = ?, subclass_id = ?,
			abilities_json = ?, updated_at = datetime('now')
		WHERE id = ?`,
		d.Step, d.Name, nullIfZero(d.RaceID), nullIfZero(d.BackgroundID),
		nullIfZero(d.ClassID), nullIfZero(d.SubclassID), d.ScoresJSON(), d.ID,
	)
	return err
}

func (r *Repository) DeleteDraft(id int64) error {
	_, err := r.DB.Exec(`DELETE FROM character_drafts WHERE id = ?`, id)
	return err
}

func scanCharacters(rows *sql.Rows) ([]Character, error) {
	defer rows.Close()
	var out []Character
	for rows.Next() {
		ch, err := scanCharacterRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *ch)
	}
	return out, rows.Err()
}

type scanner interface {
	Scan(dest ...any) error
}

func scanCharacter(row *sql.Row) (*Character, error) {
	return scanCharacterRow(row)
}

func scanCharacterRow(row scanner) (*Character, error) {
	ch := &Character{}
	var created string
	var raceID, bgID sql.NullInt64
	if err := row.Scan(
		&ch.ID, &ch.Name, &ch.OwnerID, &ch.OwnerName, &ch.CampaignID, &ch.Campaign, &ch.Level,
		&ch.STR, &ch.DEX, &ch.CON, &ch.INT, &ch.WIS, &ch.CHA, &created,
		&raceID, &bgID, &ch.HPMax, &ch.HPCurrent, &ch.HPTemp, &ch.DeathSuccess, &ch.DeathFail, &ch.ProficiencyBonus,
		&ch.Gold, &ch.Souls, &ch.SoulsCap,
	); err != nil {
		return nil, err
	}
	ch.RaceID, ch.BackgroundID = raceID.Int64, bgID.Int64
	if ch.SoulsCap < 1 {
		ch.SoulsCap = 5000
	}
	if t, err := time.ParseInLocation("2006-01-02 15:04:05", created, time.UTC); err == nil {
		ch.CreatedAt = t
	}
	return ch, nil
}

func nullIfZero(id int64) any {
	if id == 0 {
		return nil
	}
	return id
}

type characterSpellRow struct {
	SpellID  int64
	Prepared bool
}

func (r *Repository) ListCharacterSpells(characterID int64) ([]characterSpellRow, error) {
	rows, err := r.DB.Query(`
		SELECT spell_id, prepared FROM character_spells WHERE character_id = ? ORDER BY spell_id`, characterID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []characterSpellRow
	for rows.Next() {
		var id, prep int64
		if err := rows.Scan(&id, &prep); err != nil {
			return nil, err
		}
		out = append(out, characterSpellRow{SpellID: id, Prepared: prep != 0})
	}
	return out, rows.Err()
}

func (r *Repository) UpsertCharacterSpell(characterID, spellID int64, prepared bool) error {
	_, err := r.DB.Exec(`
		INSERT INTO character_spells (character_id, spell_id, prepared) VALUES (?, ?, ?)
		ON CONFLICT(character_id, spell_id) DO UPDATE SET prepared = excluded.prepared`,
		characterID, spellID, boolInt(prepared))
	return err
}

func (r *Repository) SetPrepared(characterID, spellID int64, prepared bool) error {
	res, err := r.DB.Exec(`UPDATE character_spells SET prepared = ? WHERE character_id = ? AND spell_id = ?`,
		boolInt(prepared), characterID, spellID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *Repository) DeleteCharacterSpell(characterID, spellID int64) error {
	_, err := r.DB.Exec(`DELETE FROM character_spells WHERE character_id = ? AND spell_id = ?`, characterID, spellID)
	return err
}

func (r *Repository) ListResources(characterID int64) ([]rules.Pool, error) {
	rows, err := r.DB.Query(`
		SELECT kind, slot_level, current, max FROM character_resources
		WHERE character_id = ? ORDER BY kind, slot_level`, characterID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []rules.Pool
	for rows.Next() {
		var p rules.Pool
		if err := rows.Scan(&p.Kind, &p.SlotLevel, &p.Current, &p.Max); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *Repository) ReplaceResources(characterID int64, pools []rules.Pool) error {
	tx, err := r.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM character_resources WHERE character_id = ?`, characterID); err != nil {
		return err
	}
	for _, p := range pools {
		if _, err := tx.Exec(`
			INSERT INTO character_resources (character_id, kind, slot_level, current, max)
			VALUES (?, ?, ?, ?, ?)`, characterID, p.Kind, p.SlotLevel, p.Current, p.Max); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func (r *Repository) ListSkills(characterID int64) ([]rules.SkillMark, error) {
	rows, err := r.DB.Query(`
		SELECT skill_slug, proficient, expertise FROM character_skills
		WHERE character_id = ?`, characterID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []rules.SkillMark
	for rows.Next() {
		var m rules.SkillMark
		var p, e int
		if err := rows.Scan(&m.Slug, &p, &e); err != nil {
			return nil, err
		}
		m.Proficient, m.Expertise = p != 0, e != 0
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *Repository) ListSaves(characterID int64) ([]rules.SaveMark, error) {
	rows, err := r.DB.Query(`
		SELECT ability, proficient FROM character_saves WHERE character_id = ?`, characterID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []rules.SaveMark
	for rows.Next() {
		var m rules.SaveMark
		var p int
		if err := rows.Scan(&m.Ability, &p); err != nil {
			return nil, err
		}
		m.Proficient = p != 0
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *Repository) GrantSkillIfNew(characterID int64, slug string) error {
	_, err := r.DB.Exec(`
		INSERT OR IGNORE INTO character_skills (character_id, skill_slug, proficient, expertise)
		VALUES (?, ?, 1, 0)`, characterID, slug)
	return err
}

func (r *Repository) GrantSaveIfNew(characterID int64, ability string) error {
	_, err := r.DB.Exec(`
		INSERT OR IGNORE INTO character_saves (character_id, ability, proficient)
		VALUES (?, ?, 1)`, characterID, ability)
	return err
}

func (r *Repository) UpsertSkill(characterID int64, slug string, proficient, expertise bool) error {
	if expertise {
		proficient = true
	}
	if !proficient {
		expertise = false
	}
	_, err := r.DB.Exec(`
		INSERT INTO character_skills (character_id, skill_slug, proficient, expertise)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(character_id, skill_slug) DO UPDATE SET
			proficient = excluded.proficient, expertise = excluded.expertise`,
		characterID, slug, boolInt(proficient), boolInt(expertise))
	return err
}

func (r *Repository) UpsertSave(characterID int64, ability string, proficient bool) error {
	_, err := r.DB.Exec(`
		INSERT INTO character_saves (character_id, ability, proficient)
		VALUES (?, ?, ?)
		ON CONFLICT(character_id, ability) DO UPDATE SET proficient = excluded.proficient`,
		characterID, ability, boolInt(proficient))
	return err
}

func (r *Repository) UpdateGold(characterID int64, gold int) error {
	_, err := r.DB.Exec(`UPDATE characters SET gold = ? WHERE id = ?`, gold, characterID)
	return err
}

func (r *Repository) UpdateVitals(characterID int64, hpCurrent, hpTemp, deathSuccess, deathFail int) error {
	_, err := r.DB.Exec(`
		UPDATE characters SET hp_current = ?, hp_temp = ?, death_success = ?, death_fail = ?
		WHERE id = ?`, hpCurrent, hpTemp, deathSuccess, deathFail, characterID)
	return err
}

func (r *Repository) ListEffects(characterID int64) ([]rules.Effect, error) {
	rows, err := r.DB.Query(`
		SELECT id, slug, kind, name_en, name_ru, source_spell_id, source, duration_key, formula_en, formula_ru,
		       ac_bonus, ac_base, ac_floor, speed_bonus, speed_mult, temp_hp, tags,
		       hidden, remove_on_battle_end, duration_turns, damage_formula, damage_type, source_url
		FROM character_effects WHERE character_id = ? ORDER BY id`, characterID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []rules.Effect
	for rows.Next() {
		e, err := scanEffect(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (r *Repository) UpsertEffect(characterID int64, e rules.Effect) error {
	_, err := r.DB.Exec(`
		INSERT INTO character_effects (
			character_id, slug, kind, name_en, name_ru, source_spell_id, source, duration_key, formula_en, formula_ru,
			ac_bonus, ac_base, ac_floor, speed_bonus, speed_mult, temp_hp, tags,
			hidden, remove_on_battle_end, duration_turns, damage_formula, damage_type, source_url
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(character_id, slug) DO UPDATE SET
			kind = excluded.kind, name_en = excluded.name_en, name_ru = excluded.name_ru,
			source_spell_id = excluded.source_spell_id, source = excluded.source,
			duration_key = excluded.duration_key, formula_en = excluded.formula_en, formula_ru = excluded.formula_ru,
			ac_bonus = excluded.ac_bonus, ac_base = excluded.ac_base, ac_floor = excluded.ac_floor,
			speed_bonus = excluded.speed_bonus, speed_mult = excluded.speed_mult,
			temp_hp = excluded.temp_hp, tags = excluded.tags,
			hidden = excluded.hidden, remove_on_battle_end = excluded.remove_on_battle_end,
			duration_turns = excluded.duration_turns, damage_formula = excluded.damage_formula,
			damage_type = excluded.damage_type, source_url = excluded.source_url`,
		characterID, e.Slug, e.Kind, e.NameEN, e.NameRU, nullIfZero(e.SourceSpellID),
		nz(e.Source, "spell"), e.DurationKey, e.FormulaEN, e.FormulaRU,
		e.ACBonus, e.ACBase, e.ACFloor, e.SpeedBonus, e.SpeedMult, e.TempHP, e.Tags,
		boolInt(e.Hidden), boolInt(e.RemoveOnBattleEnd), e.DurationTurns, e.DamageFormula, e.DamageType, e.SourceURL)
	return err
}

type effectScanner interface {
	Scan(dest ...any) error
}

func scanEffect(row effectScanner) (rules.Effect, error) {
	var e rules.Effect
	var spellID sql.NullInt64
	var hidden, removeEnd int
	if err := row.Scan(&e.ID, &e.Slug, &e.Kind, &e.NameEN, &e.NameRU, &spellID,
		&e.Source, &e.DurationKey, &e.FormulaEN, &e.FormulaRU,
		&e.ACBonus, &e.ACBase, &e.ACFloor, &e.SpeedBonus, &e.SpeedMult, &e.TempHP, &e.Tags,
		&hidden, &removeEnd, &e.DurationTurns, &e.DamageFormula, &e.DamageType, &e.SourceURL); err != nil {
		return e, err
	}
	e.SourceSpellID = spellID.Int64
	e.Hidden = hidden != 0
	e.RemoveOnBattleEnd = removeEnd != 0
	return e, nil
}

func nz(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}

func (r *Repository) DeleteEffect(characterID, effectID int64) error {
	_, err := r.DB.Exec(`DELETE FROM character_effects WHERE id = ? AND character_id = ?`, effectID, characterID)
	return err
}

const characterItemSelect = `
		SELECT id, catalog_item_id, quantity, equipped_slot, attuned, charges_current, charges_max,
		       custom_name, notes, atk_bonus, dmg_bonus, extra_damage, ac_bonus, ac_base, ac_floor,
		       speed_bonus, speed_mult, two_handed
		FROM character_items`

func (r *Repository) ListCharacterItems(characterID int64) ([]InventoryItem, error) {
	rows, err := r.DB.Query(characterItemSelect+` WHERE character_id = ? ORDER BY id`, characterID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []InventoryItem
	ids := make([]int64, 0)
	byID := map[int64]int{}
	for rows.Next() {
		it, err := scanInventoryItem(rows)
		if err != nil {
			return nil, err
		}
		byID[it.ID] = len(out)
		ids = append(ids, it.ID)
		out = append(out, it)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	feats, err := r.listItemFeatures(ids)
	if err != nil {
		return nil, err
	}
	for itemID, list := range feats {
		i, ok := byID[itemID]
		if !ok {
			continue
		}
		out[i].Features = list
		out[i].SyncOverlayFromFeatures()
	}
	return out, nil
}

func (r *Repository) GetCharacterItem(characterID, id int64) (InventoryItem, error) {
	return scanInventoryItem(r.DB.QueryRow(characterItemSelect+` WHERE character_id = ? AND id = ?`, characterID, id))
}

func scanInventoryItem(row scanner) (InventoryItem, error) {
	var it InventoryItem
	var attune, twoH int
	err := row.Scan(&it.ID, &it.CatalogID, &it.Quantity, &it.EquippedSlot, &attune, &it.ChargesCurrent, &it.ChargesMax,
		&it.CustomName, &it.Notes, &it.AtkBonus, &it.DmgBonus, &it.ExtraDamage, &it.ACBonus, &it.ACBase, &it.ACFloor,
		&it.SpeedBonus, &it.SpeedMult, &twoH)
	if err != nil {
		return InventoryItem{}, err
	}
	it.Attuned = attune != 0
	it.TwoHanded = twoH != 0
	return it, nil
}

func (r *Repository) InsertCharacterItem(characterID int64, it InventoryItem) (int64, error) {
	tx, err := r.DB.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	res, err := tx.Exec(`
		INSERT INTO character_items (
			character_id, catalog_item_id, quantity, equipped_slot, attuned, charges_current, charges_max,
			custom_name, notes, atk_bonus, dmg_bonus, extra_damage, ac_bonus, ac_base, ac_floor,
			speed_bonus, speed_mult, two_handed
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		characterID, it.CatalogID, it.Quantity, it.EquippedSlot, boolInt(it.Attuned), it.ChargesCurrent, it.ChargesMax,
		it.CustomName, it.Notes, it.AtkBonus, it.DmgBonus, it.ExtraDamage, it.ACBonus, it.ACBase, it.ACFloor,
		it.SpeedBonus, it.SpeedMult, boolInt(it.TwoHanded),
	)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	if err := insertItemFeatures(tx, id, it.Features); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return id, nil
}

func (r *Repository) UpdateCharacterItem(characterID int64, it InventoryItem) error {
	_, err := r.DB.Exec(`
		UPDATE character_items SET
			quantity = ?, equipped_slot = ?, attuned = ?, charges_current = ?, charges_max = ?,
			custom_name = ?, notes = ?, atk_bonus = ?, dmg_bonus = ?, extra_damage = ?,
			ac_bonus = ?, ac_base = ?, ac_floor = ?, speed_bonus = ?, speed_mult = ?, two_handed = ?
		WHERE id = ? AND character_id = ?`,
		it.Quantity, it.EquippedSlot, boolInt(it.Attuned), it.ChargesCurrent, it.ChargesMax,
		it.CustomName, it.Notes, it.AtkBonus, it.DmgBonus, it.ExtraDamage,
		it.ACBonus, it.ACBase, it.ACFloor, it.SpeedBonus, it.SpeedMult, boolInt(it.TwoHanded),
		it.ID, characterID,
	)
	return err
}

func (r *Repository) DeleteCharacterItem(characterID, id int64) error {
	_, err := r.DB.Exec(`DELETE FROM character_items WHERE id = ? AND character_id = ?`, id, characterID)
	return err
}

const itemFeatureSelect = `
		SELECT id, character_item_id, sort_order, name_en, name_ru, stat, value, origin, source_url,
		       catalog_id, catalog_kind, catalog_slug
		FROM character_item_features`

func (r *Repository) listItemFeatures(itemIDs []int64) (map[int64][]rules.FeatureDTO, error) {
	out := map[int64][]rules.FeatureDTO{}
	if len(itemIDs) == 0 {
		return out, nil
	}
	args := make([]any, len(itemIDs))
	ph := make([]string, len(itemIDs))
	for i, id := range itemIDs {
		args[i] = id
		ph[i] = "?"
	}
	q := itemFeatureSelect + ` WHERE character_item_id IN (` + strings.Join(ph, ",") + `) ORDER BY sort_order, id`
	rows, err := r.DB.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		f, err := scanItemFeature(rows)
		if err != nil {
			return nil, err
		}
		out[f.ItemID] = append(out[f.ItemID], f)
	}
	return out, rows.Err()
}

func insertItemFeatures(tx *sql.Tx, itemID int64, feats []rules.FeatureDTO) error {
	for i, f := range feats {
		if _, err := tx.Exec(`
			INSERT INTO character_item_features (
				character_item_id, sort_order, name_en, name_ru, stat, value, origin, source_url,
				catalog_id, catalog_kind, catalog_slug
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			itemID, i, f.NameEN, f.NameRU, nz(f.Stat, rules.StatNote), f.Value, nz(f.Origin, rules.OriginCommon), f.SourceURL,
			f.CatalogID, f.CatalogKind, f.CatalogSlug,
		); err != nil {
			return err
		}
	}
	return nil
}

func scanItemFeature(row scanner) (rules.FeatureDTO, error) {
	var f rules.FeatureDTO
	err := row.Scan(&f.ID, &f.ItemID, &f.SortOrder, &f.NameEN, &f.NameRU, &f.Stat, &f.Value, &f.Origin, &f.SourceURL,
		&f.CatalogID, &f.CatalogKind, &f.CatalogSlug)
	return f, err
}

func (r *Repository) DeleteTempHPEffects(characterID int64) error {
	_, err := r.DB.Exec(`
		DELETE FROM character_effects
		WHERE character_id = ? AND temp_hp > 0 AND (tags = 'temp_hp' OR tags LIKE 'temp_hp,%' OR tags LIKE '%,temp_hp' OR slug = ?)`,
		characterID, rules.OtherTempSlug)
	return err
}
