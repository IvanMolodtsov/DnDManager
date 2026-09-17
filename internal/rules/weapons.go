package rules

import (
	"fmt"
	"strings"

	"dndmanager/internal/catalog"
)

// ClassWeaponProficiency is the PHB 2014 class weapon list (unioned on multiclass).
type ClassWeaponProficiency struct {
	Simple  bool
	Martial bool
	Extra   []string
}

// ClassWeaponProf is encoded PHB 2014 class weapon proficiency (not every simple for wizard/druid).
var ClassWeaponProf = map[string]ClassWeaponProficiency{
	"barbarian": {Simple: true, Martial: true},
	"fighter":   {Simple: true, Martial: true},
	"paladin":   {Simple: true, Martial: true},
	"ranger":    {Simple: true, Martial: true},
	"cleric":    {Simple: true},
	"warlock":   {Simple: true},
	"bard":      {Simple: true, Extra: []string{"crossbow-hand", "longsword", "rapier", "shortsword"}},
	"rogue":     {Simple: true, Extra: []string{"crossbow-hand", "longsword", "rapier", "shortsword"}},
	"monk":      {Simple: true, Extra: []string{"shortsword"}},
	"wizard":    {Extra: []string{"dagger", "dart", "sling", "quarterstaff", "crossbow-light"}},
	"sorcerer":  {Extra: []string{"dagger", "dart", "sling", "quarterstaff", "crossbow-light"}},
	"druid":     {Extra: []string{"club", "dagger", "dart", "javelin", "mace", "quarterstaff", "scimitar", "sickle", "sling", "spear"}},
}

// WeaponProficient is true if any class grants this weapon (base slug, else simple/martial flags).
func WeaponProficient(classSlugs []string, baseSlug string, simple, martial bool) bool {
	slug, simple, martial := resolveWeaponIdentity(baseSlug, simple, martial)
	for _, class := range classSlugs {
		prof, ok := ClassWeaponProf[strings.ToLower(strings.TrimSpace(class))]
		if !ok {
			continue
		}
		if simple && prof.Simple {
			return true
		}
		if martial && prof.Martial {
			return true
		}
		for _, extra := range prof.Extra {
			if extra == slug {
				return true
			}
		}
	}
	return false
}

func resolveWeaponIdentity(baseSlug string, simple, martial bool) (string, bool, bool) {
	slug := strings.ToLower(strings.TrimSpace(baseSlug))
	if inh := InheritWeaponSlug(slug); inh != "" {
		slug = inh
	}
	if arm, ok := catalog.PHBArmBySlug(slug); ok {
		return slug, arm.Category == "simple", arm.Category == "martial"
	}
	return slug, simple, martial
}

// WeaponAttack is the character-relative to-hit and damage view for an equipped weapon.
type WeaponAttack struct {
	AbilityKey    string
	AbilityMod    int
	Finesse       bool
	Proficient    bool
	PB            int
	ItemAtk       int
	ItemDmg       int
	ToHit         int
	HitFormula    string
	DamageDice    string
	DamageFormula string
	DamageType    string
	DamageLine    string
	Extra         []string
	CritRange     int
	HasThrown     bool
	Throwing      bool
	Versatile     bool
	TwoHanded     bool
	Line          string
}

// DeriveWeaponAttack builds IRL formulas: 1d20 + ability + PB(if proficient) + item atk,
// and weapon dice + ability + item dmg (versatile 2H unless throwing).
func DeriveWeaponAttack(scores AbilityScores, totalLevel int, classSlugs []string, w WeaponDTO, twoHanded, thrown bool) WeaponAttack {
	derived := DeriveItemStats(w.BaseHit, w.DamageType, w.Features)
	finesse := w.Finesse || HasProperty(w.Features, "finesse")
	hasThrown := w.Thrown || HasProperty(w.Features, "thrown")
	throwing := thrown && hasThrown
	versDice := versatileDice(w.Features)
	if versDice == "" {
		versDice = strings.TrimSpace(w.VersatileDice)
	}
	abilityKey, abilityMod := attackAbility(scores, w.Melee, finesse)
	prof := WeaponProficient(classSlugs, w.BaseSlug, w.Simple, w.Martial)
	pb := ProficiencyBonus(totalLevel)
	toHit := abilityMod + derived.AtkBonus
	if prof {
		toHit += pb
	}
	dice := derived.DamageDice
	if twoHanded && !throwing && versDice != "" {
		dice = versDice
	}
	dmgBonus := abilityMod + derived.DmgBonus
	dmgFormula := joinDiceBonus(dice, dmgBonus)
	dmgLine := dmgFormula
	if dmgLine != "" && derived.DamageType != "" {
		dmgLine += " " + derived.DamageType
	}
	var extras []string
	for _, x := range derived.Extra {
		if s := strings.TrimSpace(x); s != "" {
			extras = append(extras, s)
		}
	}
	a := WeaponAttack{
		AbilityKey:    abilityKey,
		AbilityMod:    abilityMod,
		Finesse:       finesse,
		Proficient:    prof,
		PB:            pb,
		ItemAtk:       derived.AtkBonus,
		ItemDmg:       derived.DmgBonus,
		ToHit:         toHit,
		HitFormula:    CheckFormula(toHit),
		DamageDice:    dice,
		DamageFormula: dmgFormula,
		DamageType:    derived.DamageType,
		DamageLine:    dmgLine,
		Extra:         extras,
		CritRange:     derived.CritRange,
		HasThrown:     hasThrown,
		Throwing:      throwing,
		Versatile:     versDice != "",
		TwoHanded:     twoHanded,
	}
	parts := []string{formatSigned(toHit) + " to hit"}
	if a.DamageLine != "" {
		parts = append(parts, a.DamageLine)
	}
	parts = append(parts, extras...)
	if a.CritRange > 0 && a.CritRange < 20 {
		parts = append(parts, fmt.Sprintf("crit %d–20", a.CritRange))
	}
	a.Line = strings.Join(parts, " · ")
	return a
}

func attackAbility(scores AbilityScores, melee, finesse bool) (string, int) {
	str, dex := Modifier(scores.STR), Modifier(scores.DEX)
	if finesse {
		if str > dex || (str == dex && melee) {
			return "str", str
		}
		return "dex", dex
	}
	if melee {
		return "str", str
	}
	return "dex", dex
}

func versatileDice(feats []FeatureDTO) string {
	for _, f := range feats {
		if f.Stat == StatVersatile {
			if v := strings.TrimSpace(f.Value); v != "" {
				return v
			}
		}
	}
	return ""
}

func joinDiceBonus(dice string, bonus int) string {
	dice = strings.TrimSpace(dice)
	if dice == "" {
		return ""
	}
	if bonus == 0 {
		return dice
	}
	return dice + formatSigned(bonus)
}

// ExtraRollFormula keeps the leading dice expression of an extra-damage line ("2d6 fire" → "2d6").
func ExtraRollFormula(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if i := strings.IndexAny(raw, " \t"); i > 0 {
		return raw[:i]
	}
	return raw
}
