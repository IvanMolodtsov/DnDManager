package rules

import "strings"

// DamageResult is the RAW 2014 outcome of typed hit-point damage.
type DamageResult struct {
	Incoming     int
	Applied      int
	AbsorbedTemp int
	HPDamage     int
	NewHP        int
	NewEffects   []Effect
	Immune       bool
	Resistant    bool
	Vulnerable   bool
	DamageType   string
}

// NoteKey is an i18n key for resistance / immunity / vulnerability, or empty
// when the amount is unchanged (including resist+vuln cancel).
func (r DamageResult) NoteKey() string {
	if r.Immune {
		return "character.hp.mod.immune"
	}
	if r.Resistant && r.Vulnerable {
		return ""
	}
	if r.Resistant {
		return "character.hp.mod.resist"
	}
	if r.Vulnerable {
		return "character.hp.mod.vuln"
	}
	return ""
}

// DamageTypeName returns the localized PHB label, or the slug if unknown.
func DamageTypeName(slug, lang string) string {
	slug = strings.ToLower(strings.TrimSpace(slug))
	for _, o := range DamageTypes() {
		if o.Slug == slug {
			return o.Name(lang)
		}
	}
	return slug
}

// ValidDamageType reports whether slug is in the PHB (plus disease) list.
func ValidDamageType(slug string) bool {
	slug = strings.ToLower(strings.TrimSpace(slug))
	for _, o := range DamageTypes() {
		if o.Slug == slug {
			return true
		}
	}
	return false
}

func hasDamageType(list []string, typ string) bool {
	typ = strings.ToLower(strings.TrimSpace(typ))
	if typ == "" {
		return false
	}
	for _, s := range list {
		if strings.ToLower(strings.TrimSpace(s)) == typ {
			return true
		}
	}
	return false
}

// ScaleDamage applies immunity (0), then resistance (half, round down) and
// vulnerability (double). Resist + vuln for the same type cancel (net ×1).
func ScaleDamage(amount int, dmgType string, g ItemGrants) (applied int, immune, resist, vuln bool) {
	if amount < 0 {
		amount = 0
	}
	typ := strings.ToLower(strings.TrimSpace(dmgType))
	immune = hasDamageType(g.Immunities, typ)
	if immune {
		return 0, true, false, false
	}
	resist = hasDamageType(g.Resistances, typ)
	vuln = hasDamageType(g.Vulnerabilities, typ)
	applied = amount
	if resist && vuln {
		return applied, false, true, true
	}
	if resist {
		applied = amount / 2
	}
	if vuln {
		applied = amount * 2
	}
	return applied, false, resist, vuln
}

// ApplyDamage scales by equipped grants, spends stacked temp HP first (RAW),
// then current HP. Statuses do not add resistance here.
func ApplyDamage(currentHP, maxHP int, effects []Effect, grants ItemGrants, amount int, damageType string) DamageResult {
	list := append([]Effect(nil), effects...)
	applied, immune, resist, vuln := ScaleDamage(amount, damageType, grants)
	temp := SumTempHP(list)
	absorbed := applied
	if absorbed > temp {
		absorbed = temp
	}
	if absorbed > 0 {
		list = ReduceStackedTemp(list, absorbed)
	}
	hpHit := applied - absorbed
	if hpHit < 0 {
		hpHit = 0
	}
	return DamageResult{
		Incoming:     amount,
		Applied:      applied,
		AbsorbedTemp: absorbed,
		HPDamage:     hpHit,
		NewHP:        ClampHP(currentHP-hpHit, maxHP),
		NewEffects:   list,
		Immune:       immune,
		Resistant:    resist,
		Vulnerable:   vuln,
		DamageType:   strings.ToLower(strings.TrimSpace(damageType)),
	}
}
