package main

import (
	"fmt"
	"strconv"
)

// PHB 2014 dice for spells 5eapi omits or does not structure (Hex, Witch Bolt, …).
// Mechanical stats only — not lore text from dnd.su articles.
type phbFormula struct {
	formula, dmgType, heal, scale string
	atSlot, atChar                map[string]string
}

func applyPHBFormula(r *seedRow) {
	if r.Slug == "" || r.DamageFormula != "" || r.HealFormula != "" {
		return
	}
	f, ok := phbFormulas[r.Slug]
	if !ok {
		return
	}
	r.DamageFormula = f.formula
	r.DamageType = f.dmgType
	r.HealFormula = f.heal
	if r.ScaleKind == "" {
		r.ScaleKind = f.scale
	}
	if f.atSlot != nil {
		r.DamageAtSlot = mustJSON(f.atSlot)
	}
	if f.atChar != nil {
		r.DamageAtCharacter = mustJSON(f.atChar)
	}
}

func slotDice(baseCount, sides, minLevel int) map[string]string {
	m := make(map[string]string, 10-minLevel)
	for lv := minLevel; lv <= 9; lv++ {
		n := baseCount + (lv - minLevel)
		m[strconv.Itoa(lv)] = fmt.Sprintf("%dd%d", n, sides)
	}
	return m
}

func slotFlat(base, perSlot, minLevel int) map[string]string {
	m := make(map[string]string, 10-minLevel)
	for lv := minLevel; lv <= 9; lv++ {
		m[strconv.Itoa(lv)] = strconv.Itoa(base + perSlot*(lv-minLevel))
	}
	return m
}

func cantrip(sides int) map[string]string {
	d := fmt.Sprintf("1d%d", sides)
	return map[string]string{"1": d, "5": fmt.Sprintf("2d%d", sides), "11": fmt.Sprintf("3d%d", sides), "17": fmt.Sprintf("4d%d", sides)}
}

var phbFormulas = map[string]phbFormula{
	"hex":               {formula: "1d6", dmgType: "necrotic"},
	"hunters-mark":      {formula: "1d6"},
	"witch-bolt":        {formula: "1d12", dmgType: "lightning", scale: "slot", atSlot: slotDice(1, 12, 1)},
	"chromatic-orb":     {formula: "3d8", scale: "slot", atSlot: slotDice(3, 8, 1)},
	"armor-of-agathys":  {formula: "5", dmgType: "cold", scale: "slot", atSlot: slotFlat(5, 5, 1)},
	"arms-of-hadar":     {formula: "2d6", dmgType: "necrotic", scale: "slot", atSlot: slotDice(2, 6, 1)},
	"dissonant-whispers": {formula: "3d6", dmgType: "psychic", scale: "slot", atSlot: slotDice(3, 6, 1)},
	"hail-of-thorns":    {formula: "1d10", dmgType: "piercing", scale: "slot", atSlot: slotDice(1, 10, 1)},
	"ensnaring-strike":  {formula: "1d6", dmgType: "piercing", scale: "slot", atSlot: slotDice(1, 6, 1)},
	"thorn-whip":        {formula: "1d6", dmgType: "piercing", scale: "character", atChar: cantrip(6)},
	"thunderous-smite":  {formula: "2d6", dmgType: "thunder"},
	"wrathful-smite":    {formula: "1d6", dmgType: "psychic"},
	"searing-smite":     {formula: "1d6", dmgType: "fire", scale: "slot", atSlot: slotDice(1, 6, 1)},
	"branding-smite":    {formula: "2d6", dmgType: "radiant", scale: "slot", atSlot: slotDice(2, 6, 2)},
	"banishing-smite":   {formula: "5d10", dmgType: "force"},
	"staggering-smite":  {formula: "4d6", dmgType: "psychic"},
	"blinding-smite":    {formula: "3d8", dmgType: "radiant"},
	"elemental-weapon":  {formula: "1d4", scale: "slot", atSlot: map[string]string{"3": "1d4", "5": "2d4", "7": "3d4"}},
	"hunger-of-hadar":  {formula: "2d6", dmgType: "cold"},
}
