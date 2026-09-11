package characters

import (
	"database/sql"
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
		       ch.race_id, ch.background_id, ch.hp_max, ch.hp_current, ch.proficiency_bonus
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
		&raceID, &bgID, &ch.HPMax, &ch.HPCurrent, &ch.ProficiencyBonus,
	); err != nil {
		return nil, err
	}
	ch.RaceID, ch.BackgroundID = raceID.Int64, bgID.Int64
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
