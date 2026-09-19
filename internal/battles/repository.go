package battles

import (
	"database/sql"
	"errors"
	"strings"

	"dndmanager/internal/catalog"
	"dndmanager/internal/rules"
)

// Repository is SQLite access for battles and units.
type Repository struct {
	DB *sql.DB
}

func (r *Repository) FindByCampaign(campaignID int64) (*Battle, error) {
	return scanBattle(r.DB.QueryRow(`
		SELECT id, campaign_id, status, round, active_index, created_by, loot_generated, souls_applied
		FROM battles WHERE campaign_id = ?`, campaignID))
}

func (r *Repository) Find(id int64) (*Battle, error) {
	return scanBattle(r.DB.QueryRow(`
		SELECT id, campaign_id, status, round, active_index, created_by, loot_generated, souls_applied
		FROM battles WHERE id = ?`, id))
}

func (r *Repository) Insert(campaignID, createdBy int64, status string) (int64, error) {
	res, err := r.DB.Exec(`
		INSERT INTO battles (campaign_id, status, round, active_index, created_by)
		VALUES (?, ?, 1, 0, ?)`, campaignID, status, createdBy)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (r *Repository) Update(b *Battle) error {
	_, err := r.DB.Exec(`
		UPDATE battles SET status = ?, round = ?, active_index = ?, loot_generated = ?, souls_applied = ? WHERE id = ?`,
		b.Status, b.Round, b.ActiveIndex, btoi(b.LootGenerated), btoi(b.SoulsApplied), b.ID)
	return err
}

func (r *Repository) Delete(id int64) error {
	_, err := r.DB.Exec(`DELETE FROM battles WHERE id = ?`, id)
	return err
}

func (r *Repository) ListUnits(battleID int64) ([]Unit, error) {
	rows, err := r.DB.Query(`
		SELECT id, battle_id, kind, COALESCE(character_id, 0), COALESCE(companion_id, 0), COALESCE(catalog_monster_id, 0),
		       name, initiative, hp_current, hp_max, temp_hp, ac, str, dex, con, intel, wis, cha,
		       death_success, death_fail, dead, escaped, knocked, sort_order, resist_json, source_url_ru, cr, cr_label
		FROM battle_units WHERE battle_id = ?
		ORDER BY sort_order, id`, battleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Unit
	for rows.Next() {
		u, err := scanUnit(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func (r *Repository) Unit(id int64) (*Unit, error) {
	u, err := scanUnit(r.DB.QueryRow(`
		SELECT id, battle_id, kind, COALESCE(character_id, 0), COALESCE(companion_id, 0), COALESCE(catalog_monster_id, 0),
		       name, initiative, hp_current, hp_max, temp_hp, ac, str, dex, con, intel, wis, cha,
		       death_success, death_fail, dead, escaped, knocked, sort_order, resist_json, source_url_ru, cr, cr_label
		FROM battle_units WHERE id = ?`, id))
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *Repository) InsertUnit(u *Unit) (int64, error) {
	res, err := r.DB.Exec(`
		INSERT INTO battle_units (
			battle_id, kind, character_id, companion_id, catalog_monster_id, name, initiative,
			hp_current, hp_max, temp_hp, ac, str, dex, con, intel, wis, cha,
			death_success, death_fail, dead, escaped, knocked, sort_order, resist_json, source_url_ru, cr, cr_label
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		u.BattleID, u.Kind, nullIfZero(u.CharacterID), nullIfZero(u.CompanionID), nullIfZero(u.CatalogMonsterID), u.Name, u.Initiative,
		u.HPCurrent, u.HPMax, u.TempHP, u.AC, u.STR, u.DEX, u.CON, u.INT, u.WIS, u.CHA,
		u.DeathSuccess, u.DeathFail, btoi(u.Dead), btoi(u.Escaped), btoi(u.Knocked), u.SortOrder, u.ResistJSON, u.SourceURLRU, u.CR, nz(u.CRLabel, "0"),
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (r *Repository) UpdateUnit(u *Unit) error {
	_, err := r.DB.Exec(`
		UPDATE battle_units SET
			name = ?, initiative = ?, hp_current = ?, hp_max = ?, temp_hp = ?, ac = ?,
			str = ?, dex = ?, con = ?, intel = ?, wis = ?, cha = ?,
			death_success = ?, death_fail = ?, dead = ?, escaped = ?, knocked = ?, sort_order = ?,
			resist_json = ?, source_url_ru = ?, cr = ?, cr_label = ?
		WHERE id = ?`,
		u.Name, u.Initiative, u.HPCurrent, u.HPMax, u.TempHP, u.AC,
		u.STR, u.DEX, u.CON, u.INT, u.WIS, u.CHA,
		u.DeathSuccess, u.DeathFail, btoi(u.Dead), btoi(u.Escaped), btoi(u.Knocked), u.SortOrder,
		u.ResistJSON, u.SourceURLRU, u.CR, nz(u.CRLabel, "0"), u.ID,
	)
	return err
}

func (r *Repository) DeleteUnit(id int64) error {
	_, err := r.DB.Exec(`DELETE FROM battle_units WHERE id = ?`, id)
	return err
}

func (r *Repository) DeleteUnitsByMonster(battleID, catalogID int64) error {
	_, err := r.DB.Exec(`
		DELETE FROM battle_units WHERE battle_id = ? AND catalog_monster_id = ? AND kind = ?`,
		battleID, catalogID, KindMonster)
	return err
}

func snapshotFromMonster(m *catalog.Monster) string {
	if m == nil {
		return "{}"
	}
	return EncodeSnapshot(ResistSnapshot{
		Resistances:     m.Resistances,
		Immunities:      m.Immunities,
		Vulnerabilities: m.Vulnerabilities,
		Saves:           m.SaveProficiencies,
	})
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanBattle(row rowScanner) (*Battle, error) {
	b := &Battle{}
	var lootGen, soulsApplied int
	if err := row.Scan(&b.ID, &b.CampaignID, &b.Status, &b.Round, &b.ActiveIndex, &b.CreatedBy, &lootGen, &soulsApplied); err != nil {
		return nil, err
	}
	b.LootGenerated = lootGen != 0
	b.SoulsApplied = soulsApplied != 0
	return b, nil
}

func scanUnit(row rowScanner) (Unit, error) {
	var u Unit
	var dead, escaped, knocked int
	if err := row.Scan(&u.ID, &u.BattleID, &u.Kind, &u.CharacterID, &u.CompanionID, &u.CatalogMonsterID, &u.Name,
		&u.Initiative, &u.HPCurrent, &u.HPMax, &u.TempHP, &u.AC,
		&u.STR, &u.DEX, &u.CON, &u.INT, &u.WIS, &u.CHA, &u.DeathSuccess, &u.DeathFail,
		&dead, &escaped, &knocked, &u.SortOrder, &u.ResistJSON, &u.SourceURLRU, &u.CR, &u.CRLabel); err != nil {
		return Unit{}, err
	}
	u.Dead = dead != 0
	u.Escaped = escaped != 0
	u.Knocked = knocked != 0
	if strings.TrimSpace(u.ResistJSON) == "" {
		u.ResistJSON = "{}"
	}
	if strings.TrimSpace(u.CRLabel) == "" {
		u.CRLabel = "0"
	}
	return u, nil
}

func nullIfZero(n int64) any {
	if n == 0 {
		return nil
	}
	return n
}

func btoi(v bool) int {
	if v {
		return 1
	}
	return 0
}

func isNoRows(err error) bool {
	return errors.Is(err, sql.ErrNoRows)
}

const unitEffectSelect = `
		SELECT unit_id, id, slug, kind, name_en, name_ru, source, duration_key, formula_en, formula_ru,
		       ac_bonus, ac_base, ac_floor, speed_bonus, speed_mult, temp_hp, tags,
		       hidden, remove_on_battle_end, duration_turns, damage_formula, damage_type, source_url
		FROM battle_unit_effects`

func (r *Repository) ListEffectsForBattle(battleID int64) (map[int64][]rules.Effect, error) {
	rows, err := r.DB.Query(unitEffectSelect+`
		WHERE unit_id IN (SELECT id FROM battle_units WHERE battle_id = ?)
		ORDER BY id`, battleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int64][]rules.Effect{}
	for rows.Next() {
		unitID, e, err := scanUnitEffect(rows)
		if err != nil {
			return nil, err
		}
		out[unitID] = append(out[unitID], e)
	}
	return out, rows.Err()
}

func (r *Repository) ListUnitEffects(unitID int64) ([]rules.Effect, error) {
	rows, err := r.DB.Query(unitEffectSelect+` WHERE unit_id = ? ORDER BY id`, unitID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []rules.Effect
	for rows.Next() {
		_, e, err := scanUnitEffect(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (r *Repository) UpsertUnitEffect(unitID int64, e rules.Effect) error {
	_, err := r.DB.Exec(`
		INSERT INTO battle_unit_effects (
			unit_id, slug, kind, name_en, name_ru, source, duration_key, formula_en, formula_ru,
			ac_bonus, ac_base, ac_floor, speed_bonus, speed_mult, temp_hp, tags,
			hidden, remove_on_battle_end, duration_turns, damage_formula, damage_type, source_url
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(unit_id, slug) DO UPDATE SET
			kind = excluded.kind, name_en = excluded.name_en, name_ru = excluded.name_ru,
			source = excluded.source, duration_key = excluded.duration_key,
			formula_en = excluded.formula_en, formula_ru = excluded.formula_ru,
			ac_bonus = excluded.ac_bonus, ac_base = excluded.ac_base, ac_floor = excluded.ac_floor,
			speed_bonus = excluded.speed_bonus, speed_mult = excluded.speed_mult,
			temp_hp = excluded.temp_hp, tags = excluded.tags,
			hidden = excluded.hidden, remove_on_battle_end = excluded.remove_on_battle_end,
			duration_turns = excluded.duration_turns, damage_formula = excluded.damage_formula,
			damage_type = excluded.damage_type, source_url = excluded.source_url`,
		unitID, e.Slug, e.Kind, e.NameEN, e.NameRU, nz(e.Source, "other"), e.DurationKey, e.FormulaEN, e.FormulaRU,
		e.ACBonus, e.ACBase, e.ACFloor, e.SpeedBonus, e.SpeedMult, e.TempHP, e.Tags,
		btoi(e.Hidden), btoi(e.RemoveOnBattleEnd), e.DurationTurns, e.DamageFormula, e.DamageType, e.SourceURL)
	return err
}

func (r *Repository) DeleteUnitEffect(unitID, effectID int64) error {
	_, err := r.DB.Exec(`DELETE FROM battle_unit_effects WHERE id = ? AND unit_id = ?`, effectID, unitID)
	return err
}

func (r *Repository) DeleteEffectsForBattle(battleID int64) error {
	_, err := r.DB.Exec(`
		DELETE FROM battle_unit_effects WHERE unit_id IN (SELECT id FROM battle_units WHERE battle_id = ?)`, battleID)
	return err
}

func scanUnitEffect(row rowScanner) (int64, rules.Effect, error) {
	var unitID int64
	var e rules.Effect
	var hidden, removeEnd int
	if err := row.Scan(&unitID, &e.ID, &e.Slug, &e.Kind, &e.NameEN, &e.NameRU, &e.Source, &e.DurationKey, &e.FormulaEN, &e.FormulaRU,
		&e.ACBonus, &e.ACBase, &e.ACFloor, &e.SpeedBonus, &e.SpeedMult, &e.TempHP, &e.Tags,
		&hidden, &removeEnd, &e.DurationTurns, &e.DamageFormula, &e.DamageType, &e.SourceURL); err != nil {
		return 0, e, err
	}
	e.Hidden = hidden != 0
	e.RemoveOnBattleEnd = removeEnd != 0
	return unitID, e, nil
}

func nz(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}
