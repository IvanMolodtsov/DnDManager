package rules

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	SourceSpell   = "spell"
	SourcePotion  = "potion"
	SourceAbility = "ability"
	SourceOther   = "other"
)

// SpellEffectSpec is the combat overlay applied when a prepared spell is cast.
type SpellEffectSpec struct {
	Kind        string
	ACBonus     int
	ACBase      int
	ACFloor     int
	SpeedMult   int
	Tags        string
	SetsTemp    bool
	DurationKey string
}

var spellEffects = map[string]SpellEffectSpec{
	"armor-of-agathys": {Kind: "buff", SetsTemp: true, Tags: "temp_hp", DurationKey: "character.status.dur.1hour"},
	"mage-armor":       {Kind: "buff", ACBase: 13, DurationKey: "character.status.dur.8hours"},
	"shield":           {Kind: "buff", ACBonus: 5, DurationKey: "character.status.dur.1round"},
	"shield-of-faith":  {Kind: "buff", ACBonus: 2, DurationKey: "character.status.dur.conc_10min"},
	"barkskin":         {Kind: "buff", ACFloor: 16, DurationKey: "character.status.dur.conc_1hour"},
	"haste":            {Kind: "buff", ACBonus: 2, SpeedMult: 2, DurationKey: "character.status.dur.conc_1min"},
	"sanctuary":        {Kind: "condition", Tags: "sanctuary", DurationKey: "character.status.dur.1minute"},
}

// SpellCombatEffect returns the sheet status for a cast, if the spell changes combat stats.
func SpellCombatEffect(slug, nameEN, nameRU string, spellID int64, tempFromFormula int) (Effect, bool) {
	spec, ok := spellEffects[slug]
	if !ok {
		return Effect{}, false
	}
	e := Effect{
		Slug: slug, Kind: spec.Kind, NameEN: nameEN, NameRU: nameRU, SourceSpellID: spellID,
		Source: SourceSpell, DurationKey: spec.DurationKey,
		ACBonus: spec.ACBonus, ACBase: spec.ACBase, ACFloor: spec.ACFloor,
		SpeedMult: spec.SpeedMult, Tags: spec.Tags,
	}
	if spec.SetsTemp {
		e.TempHP = tempFromFormula
	}
	e.FormulaEN, e.FormulaRU = EffectFormula(e)
	return e, true
}

func EffectFormula(e Effect) (en, ru string) {
	var enP, ruP []string
	if e.TempHP > 0 {
		enP = append(enP, fmt.Sprintf("+%d temp HP", e.TempHP))
		ruP = append(ruP, fmt.Sprintf("+%d врем. хитов", e.TempHP))
	}
	if e.ACBonus > 0 {
		enP = append(enP, fmt.Sprintf("+%d AC", e.ACBonus))
		ruP = append(ruP, fmt.Sprintf("+%d КД", e.ACBonus))
	}
	if e.ACBase > 0 {
		enP = append(enP, fmt.Sprintf("AC %d+DEX", e.ACBase))
		ruP = append(ruP, fmt.Sprintf("КД %d+ЛОВ", e.ACBase))
	}
	if e.ACFloor > 0 {
		enP = append(enP, fmt.Sprintf("AC min %d", e.ACFloor))
		ruP = append(ruP, fmt.Sprintf("КД не ниже %d", e.ACFloor))
	}
	if e.SpeedBonus != 0 {
		enP = append(enP, fmt.Sprintf("%+d ft speed", e.SpeedBonus))
		ruP = append(ruP, fmt.Sprintf("%+d фт. скорости", e.SpeedBonus))
	}
	if e.SpeedMult > 1 {
		enP = append(enP, fmt.Sprintf("speed ×%d", e.SpeedMult))
		ruP = append(ruP, fmt.Sprintf("скорость ×%d", e.SpeedMult))
	}
	if e.HasTag("sanctuary") && len(enP) == 0 {
		enP = append(enP, "attackers must Wis save")
		ruP = append(ruP, "атакующие — спас. Мудрости")
	}
	if strings.TrimSpace(e.DamageFormula) != "" {
		line := strings.TrimSpace(e.DamageFormula)
		if e.DamageType != "" {
			line += " " + e.DamageType
		}
		enP = append(enP, line)
		ruP = append(ruP, line)
	}
	return joinFormula(enP), joinFormula(ruP)
}

func joinFormula(parts []string) string {
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += " · "
		}
		out += p
	}
	return out
}

func PotionCombatEffect(slug, nameEN, nameRU string) (Effect, bool) {
	switch slug {
	case "potion-of-speed":
		e := Effect{
			Slug: slug, Kind: "buff", NameEN: nameEN, NameRU: nameRU,
			Source: SourcePotion, DurationKey: "character.status.dur.1minute",
			ACBonus: 2, SpeedMult: 2,
		}
		e.FormulaEN, e.FormulaRU = EffectFormula(e)
		return e, true
	default:
		return Effect{}, false
	}
}

func HealingPotionFormula(slug string) string {
	switch slug {
	case "potion-of-healing", "potion-of-healing-common":
		return "2d4+2"
	case "potion-of-healing-greater", "potion-of-greater-healing":
		return "4d4+4"
	case "potion-of-healing-superior", "potion-of-superior-healing":
		return "8d4+8"
	case "potion-of-healing-supreme", "potion-of-supreme-healing":
		return "10d4+20"
	default:
		return ""
	}
}

func ParseFlatAmount(formula string) int {
	n, err := strconv.Atoi(formula)
	if err != nil || n < 0 {
		return 0
	}
	return n
}
