package battles

import (
	"strings"
)

func (r *Repository) ListLoot(battleID int64) ([]LootRow, error) {
	rows, err := r.DB.Query(`
		SELECT id, battle_id, kind, name, name_ru, COALESCE(catalog_item_id, 0), COALESCE(catalog_spell_id, 0),
		       qty, rarity, notes, sort_order
		FROM battle_loot WHERE battle_id = ?
		ORDER BY sort_order, id`, battleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []LootRow
	for rows.Next() {
		row, err := scanLoot(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r *Repository) Loot(id int64) (*LootRow, error) {
	row, err := scanLoot(r.DB.QueryRow(`
		SELECT id, battle_id, kind, name, name_ru, COALESCE(catalog_item_id, 0), COALESCE(catalog_spell_id, 0),
		       qty, rarity, notes, sort_order
		FROM battle_loot WHERE id = ?`, id))
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *Repository) InsertLoot(row *LootRow) (int64, error) {
	res, err := r.DB.Exec(`
		INSERT INTO battle_loot (
			battle_id, kind, name, name_ru, catalog_item_id, catalog_spell_id, qty, rarity, notes, sort_order
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		row.BattleID, row.Kind, row.NameEN, row.NameRU, nullIfZero(row.CatalogItemID), nullIfZero(row.CatalogSpellID),
		row.Qty, row.Rarity, row.Notes, row.SortOrder)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (r *Repository) DeleteLoot(id, battleID int64) error {
	_, err := r.DB.Exec(`DELETE FROM battle_loot WHERE id = ? AND battle_id = ?`, id, battleID)
	return err
}

func (r *Repository) DeleteGeneratedLoot(battleID int64) error {
	_, err := r.DB.Exec(`
		DELETE FROM battle_loot WHERE battle_id = ? AND kind IN (?, ?, ?, ?)`,
		battleID, LootGold, LootMagic, LootScroll, LootTreasure)
	return err
}

func (r *Repository) UpsertSoulsLoot(battleID int64, qty int) error {
	var id int64
	err := r.DB.QueryRow(`SELECT id FROM battle_loot WHERE battle_id = ? AND kind = ?`, battleID, LootSouls).Scan(&id)
	if err == nil {
		_, err = r.DB.Exec(`UPDATE battle_loot SET qty = ?, name = ?, name_ru = ? WHERE id = ?`, qty, "Souls", "Души", id)
		return err
	}
	if !isNoRows(err) {
		return err
	}
	_, err = r.InsertLoot(&LootRow{
		BattleID: battleID, Kind: LootSouls, NameEN: "Souls", NameRU: "Души", Qty: qty, SortOrder: 1,
	})
	return err
}

func scanLoot(row rowScanner) (LootRow, error) {
	var l LootRow
	if err := row.Scan(&l.ID, &l.BattleID, &l.Kind, &l.NameEN, &l.NameRU, &l.CatalogItemID, &l.CatalogSpellID,
		&l.Qty, &l.Rarity, &l.Notes, &l.SortOrder); err != nil {
		return LootRow{}, err
	}
	if strings.TrimSpace(l.NameRU) == "" {
		l.NameRU = l.NameEN
	}
	return l, nil
}
