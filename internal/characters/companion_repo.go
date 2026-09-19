package characters

import (
	"encoding/json"
	"strings"
)

func (r *Repository) ListCompanions(characterID int64) ([]Companion, error) {
	rows, err := r.DB.Query(`
		SELECT id, character_id, kind, COALESCE(catalog_monster_id, 0), COALESCE(item_id, 0),
		       name, name_en, name_ru, size, type, armor_class, hp_current, hp_max,
		       str, dex, con, intel, wis, cha, resist_json, source_url, source_url_ru,
		       can_attack, sort_order
		FROM character_companions WHERE character_id = ?
		ORDER BY sort_order, id`, characterID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Companion
	for rows.Next() {
		c, err := scanCompanion(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *Repository) Companion(characterID, id int64) (*Companion, error) {
	c, err := scanCompanion(r.DB.QueryRow(`
		SELECT id, character_id, kind, COALESCE(catalog_monster_id, 0), COALESCE(item_id, 0),
		       name, name_en, name_ru, size, type, armor_class, hp_current, hp_max,
		       str, dex, con, intel, wis, cha, resist_json, source_url, source_url_ru,
		       can_attack, sort_order
		FROM character_companions WHERE id = ? AND character_id = ?`, id, characterID))
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *Repository) InsertCompanion(c *Companion) (int64, error) {
	res, err := r.DB.Exec(`
		INSERT INTO character_companions (
			character_id, kind, catalog_monster_id, item_id, name, name_en, name_ru,
			size, type, armor_class, hp_current, hp_max, str, dex, con, intel, wis, cha,
			resist_json, source_url, source_url_ru, can_attack, sort_order
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		c.CharacterID, c.Kind, nullIfZero(c.CatalogMonsterID), nullIfZero(c.ItemID),
		c.Name, c.NameEN, c.NameRU, c.Size, c.Type, c.AC, c.HPCurrent, c.HPMax,
		c.STR, c.DEX, c.CON, c.INT, c.WIS, c.CHA, nzJSON(c.ResistJSON),
		c.SourceURL, c.SourceURLRU, boolInt(c.CanAttack), c.SortOrder,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (r *Repository) UpdateCompanion(c *Companion) error {
	_, err := r.DB.Exec(`
		UPDATE character_companions SET
			name = ?, name_en = ?, name_ru = ?, size = ?, type = ?,
			armor_class = ?, hp_current = ?, hp_max = ?,
			str = ?, dex = ?, con = ?, intel = ?, wis = ?, cha = ?,
			resist_json = ?, source_url = ?, source_url_ru = ?, can_attack = ?, sort_order = ?,
			catalog_monster_id = ?, item_id = ?
		WHERE id = ? AND character_id = ?`,
		c.Name, c.NameEN, c.NameRU, c.Size, c.Type, c.AC, c.HPCurrent, c.HPMax,
		c.STR, c.DEX, c.CON, c.INT, c.WIS, c.CHA, nzJSON(c.ResistJSON),
		c.SourceURL, c.SourceURLRU, boolInt(c.CanAttack), c.SortOrder,
		nullIfZero(c.CatalogMonsterID), nullIfZero(c.ItemID), c.ID, c.CharacterID,
	)
	return err
}

func (r *Repository) DeleteCompanion(characterID, id int64) error {
	_, err := r.DB.Exec(`DELETE FROM character_companions WHERE id = ? AND character_id = ?`, id, characterID)
	return err
}

func scanCompanion(row interface{ Scan(dest ...any) error }) (Companion, error) {
	var c Companion
	var atk int
	if err := row.Scan(&c.ID, &c.CharacterID, &c.Kind, &c.CatalogMonsterID, &c.ItemID,
		&c.Name, &c.NameEN, &c.NameRU, &c.Size, &c.Type, &c.AC, &c.HPCurrent, &c.HPMax,
		&c.STR, &c.DEX, &c.CON, &c.INT, &c.WIS, &c.CHA, &c.ResistJSON, &c.SourceURL, &c.SourceURLRU,
		&atk, &c.SortOrder); err != nil {
		return Companion{}, err
	}
	c.CanAttack = atk != 0
	if strings.TrimSpace(c.ResistJSON) == "" {
		c.ResistJSON = "{}"
	}
	return c, nil
}

func nzJSON(s string) string {
	if strings.TrimSpace(s) == "" {
		return "{}"
	}
	var raw json.RawMessage
	if json.Unmarshal([]byte(s), &raw) != nil {
		return "{}"
	}
	return s
}
