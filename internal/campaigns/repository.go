package campaigns

import (
	"database/sql"
	"time"
)

// Repository is SQLite access for campaigns and membership.
type Repository struct {
	DB *sql.DB
}

func (r *Repository) Create(name, invite string, createdBy int64) (int64, error) {
	tx, err := r.DB.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	res, err := tx.Exec(
		`INSERT INTO campaigns (name, invite_code, created_by) VALUES (?, ?, ?)`,
		name, invite, createdBy,
	)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	if _, err := tx.Exec(
		`INSERT INTO campaign_members (campaign_id, user_id, role) VALUES (?, ?, ?)`,
		id, createdBy, MemberDM,
	); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return id, nil
}

func (r *Repository) FindByID(id int64) (*Campaign, error) {
	return scanCampaign(r.DB.QueryRow(
		`SELECT id, name, invite_code, created_by, created_at, souls, souls_cap FROM campaigns WHERE id = ?`, id,
	))
}

func (r *Repository) FindByInvite(code string) (*Campaign, error) {
	return scanCampaign(r.DB.QueryRow(
		`SELECT id, name, invite_code, created_by, created_at, souls, souls_cap FROM campaigns WHERE invite_code = ?`, code,
	))
}

func (r *Repository) InviteExists(code string) (bool, error) {
	var n int
	err := r.DB.QueryRow(`SELECT COUNT(*) FROM campaigns WHERE invite_code = ?`, code).Scan(&n)
	return n > 0, err
}

func (r *Repository) ListForUser(userID int64) ([]CampaignListItem, error) {
	rows, err := r.DB.Query(`
		SELECT c.id, c.name, c.invite_code, c.created_by, c.created_at, c.souls, c.souls_cap, m.role
		FROM campaigns c
		JOIN campaign_members m ON m.campaign_id = c.id
		WHERE m.user_id = ?
		ORDER BY c.created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []CampaignListItem
	for rows.Next() {
		var item CampaignListItem
		var created string
		if err := rows.Scan(&item.ID, &item.Name, &item.InviteCode, &item.CreatedBy, &created, &item.Souls, &item.SoulsCap, &item.MemberRole); err != nil {
			return nil, err
		}
		item.CreatedAt = parseTime(created)
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r *Repository) Membership(campaignID, userID int64) (*Membership, error) {
	m := &Membership{}
	err := r.DB.QueryRow(
		`SELECT campaign_id, user_id, role FROM campaign_members WHERE campaign_id = ? AND user_id = ?`,
		campaignID, userID,
	).Scan(&m.CampaignID, &m.UserID, &m.Role)
	if err != nil {
		return nil, err
	}
	return m, nil
}

func (r *Repository) AddMember(campaignID, userID int64, role string) error {
	_, err := r.DB.Exec(
		`INSERT INTO campaign_members (campaign_id, user_id, role) VALUES (?, ?, ?)`,
		campaignID, userID, role,
	)
	return err
}

func (r *Repository) Members(campaignID int64) ([]Membership, error) {
	rows, err := r.DB.Query(`
		SELECT m.campaign_id, m.user_id, m.role, u.username
		FROM campaign_members m
		JOIN users u ON u.id = m.user_id
		WHERE m.campaign_id = ?
		ORDER BY CASE m.role WHEN 'DungeonMaster' THEN 0 ELSE 1 END, u.username`, campaignID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Membership
	for rows.Next() {
		var m Membership
		if err := rows.Scan(&m.CampaignID, &m.UserID, &m.Role, &m.Username); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *Repository) UpdateSouls(id int64, souls, cap int) error {
	_, err := r.DB.Exec(`UPDATE campaigns SET souls = ?, souls_cap = ? WHERE id = ?`, souls, cap, id)
	return err
}

func scanCampaign(row *sql.Row) (*Campaign, error) {
	c := &Campaign{}
	var created string
	if err := row.Scan(&c.ID, &c.Name, &c.InviteCode, &c.CreatedBy, &created, &c.Souls, &c.SoulsCap); err != nil {
		return nil, err
	}
	c.CreatedAt = parseTime(created)
	if c.SoulsCap < 1 {
		c.SoulsCap = DefaultSoulsCap
	}
	return c, nil
}

func parseTime(s string) time.Time {
	t, err := time.ParseInLocation("2006-01-02 15:04:05", s, time.UTC)
	if err != nil {
		return time.Time{}
	}
	return t
}
