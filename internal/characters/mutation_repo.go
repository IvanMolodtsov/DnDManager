package characters

import (
	"database/sql"

	"dndmanager/internal/rules"
)

func (r *Repository) ListMutations(characterID int64) ([]Mutation, error) {
	rows, err := r.DB.Query(`
		SELECT id, character_id, body_part, monster_id, name_en, name_ru, size, type, cr, cr_label, source_url, source_url_ru
		FROM character_mutations WHERE character_id = ? ORDER BY id`, characterID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Mutation
	for rows.Next() {
		m, err := scanMutation(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range out {
		feats, err := r.ListMutationFeatures(out[i].ID)
		if err != nil {
			return nil, err
		}
		out[i].Features = feats
	}
	return out, nil
}

func (r *Repository) Mutation(id int64) (Mutation, error) {
	m, err := scanMutation(r.DB.QueryRow(`
		SELECT id, character_id, body_part, monster_id, name_en, name_ru, size, type, cr, cr_label, source_url, source_url_ru
		FROM character_mutations WHERE id = ?`, id))
	if err != nil {
		return m, err
	}
	m.Features, err = r.ListMutationFeatures(m.ID)
	return m, err
}

func (r *Repository) UpsertMutation(m Mutation) (int64, error) {
	res, err := r.DB.Exec(`
		INSERT INTO character_mutations (
			character_id, body_part, monster_id, name_en, name_ru, size, type, cr, cr_label, source_url, source_url_ru
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(character_id, body_part) DO UPDATE SET
			monster_id = excluded.monster_id, name_en = excluded.name_en, name_ru = excluded.name_ru,
			size = excluded.size, type = excluded.type, cr = excluded.cr, cr_label = excluded.cr_label,
			source_url = excluded.source_url, source_url_ru = excluded.source_url_ru`,
		m.CharacterID, m.BodyPart, nullIfZero(m.MonsterID), m.NameEN, m.NameRU, m.Size, m.Type, m.CR, nz(m.CRLabel, "0"),
		m.SourceURL, m.SourceURLRU)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	if id == 0 {
		err = r.DB.QueryRow(`SELECT id FROM character_mutations WHERE character_id = ? AND body_part = ?`, m.CharacterID, m.BodyPart).Scan(&id)
		if err != nil {
			return 0, err
		}
	}
	return id, nil
}

func (r *Repository) DeleteMutation(characterID, mutationID int64) error {
	_, err := r.DB.Exec(`DELETE FROM character_mutations WHERE id = ? AND character_id = ?`, mutationID, characterID)
	return err
}

func (r *Repository) ListMutationFeatures(mutationID int64) ([]rules.FeatureDTO, error) {
	rows, err := r.DB.Query(`
		SELECT id, mutation_id, sort_order, stat, value, extra, notes
		FROM character_mutation_features WHERE mutation_id = ? ORDER BY sort_order, id`, mutationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []rules.FeatureDTO
	for rows.Next() {
		var f rules.FeatureDTO
		var extra, notes string
		if err := rows.Scan(&f.ID, &f.ItemID, &f.SortOrder, &f.Stat, &f.Value, &extra, &notes); err != nil {
			return nil, err
		}
		if notes != "" && f.Value == "" {
			f.Value = notes
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

func (r *Repository) ReplaceMutationFeatures(mutationID int64, feats []rules.FeatureDTO) error {
	if _, err := r.DB.Exec(`DELETE FROM character_mutation_features WHERE mutation_id = ?`, mutationID); err != nil {
		return err
	}
	for i, f := range feats {
		if _, err := r.DB.Exec(`
			INSERT INTO character_mutation_features (mutation_id, sort_order, stat, value, extra, notes)
			VALUES (?, ?, ?, ?, ?, ?)`, mutationID, i, f.Stat, f.Value, "", f.NameEN); err != nil {
			return err
		}
	}
	return nil
}

func (r *Repository) InsertMutationFeature(mutationID int64, f rules.FeatureDTO) error {
	var n int
	_ = r.DB.QueryRow(`SELECT COALESCE(MAX(sort_order), -1) FROM character_mutation_features WHERE mutation_id = ?`, mutationID).Scan(&n)
	_, err := r.DB.Exec(`
		INSERT INTO character_mutation_features (mutation_id, sort_order, stat, value, extra, notes)
		VALUES (?, ?, ?, ?, ?, ?)`, mutationID, n+1, f.Stat, f.Value, "", f.NameEN)
	return err
}

func (r *Repository) DeleteMutationFeature(mutationID, featureID int64) error {
	_, err := r.DB.Exec(`DELETE FROM character_mutation_features WHERE id = ? AND mutation_id = ?`, featureID, mutationID)
	return err
}

func scanMutation(row effectScanner) (Mutation, error) {
	var m Mutation
	var monster sql.NullInt64
	if err := row.Scan(&m.ID, &m.CharacterID, &m.BodyPart, &monster, &m.NameEN, &m.NameRU, &m.Size, &m.Type, &m.CR, &m.CRLabel, &m.SourceURL, &m.SourceURLRU); err != nil {
		return m, err
	}
	m.MonsterID = monster.Int64
	return m, nil
}
