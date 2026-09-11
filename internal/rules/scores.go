package rules

import "strings"

// AbilityScores is STR/DEX/CON/INT/WIS/CHA.
type AbilityScores struct {
	STR int `json:"str"`
	DEX int `json:"dex"`
	CON int `json:"con"`
	INT int `json:"int"`
	WIS int `json:"wis"`
	CHA int `json:"cha"`
}

// StandardArray is the PHB 5e assignment {15, 14, 13, 12, 10, 8}.
var StandardArray = []int{15, 14, 13, 12, 10, 8}

var AbilityKeys = []string{"str", "dex", "con", "int", "wis", "cha"}

func (a AbilityScores) Get(key string) int {
	switch strings.ToLower(key) {
	case "str":
		return a.STR
	case "dex":
		return a.DEX
	case "con":
		return a.CON
	case "int":
		return a.INT
	case "wis":
		return a.WIS
	case "cha":
		return a.CHA
	default:
		return 0
	}
}

func (a *AbilityScores) Set(key string, v int) {
	switch strings.ToLower(key) {
	case "str":
		a.STR = v
	case "dex":
		a.DEX = v
	case "con":
		a.CON = v
	case "int":
		a.INT = v
	case "wis":
		a.WIS = v
	case "cha":
		a.CHA = v
	}
}

func (a *AbilityScores) AddKey(key string, n int) {
	a.Set(key, a.Get(key)+n)
}

func (a AbilityScores) Add(b AbilityScores) AbilityScores {
	return AbilityScores{
		STR: a.STR + b.STR,
		DEX: a.DEX + b.DEX,
		CON: a.CON + b.CON,
		INT: a.INT + b.INT,
		WIS: a.WIS + b.WIS,
		CHA: a.CHA + b.CHA,
	}
}

func (a AbilityScores) Sum() int {
	return a.STR + a.DEX + a.CON + a.INT + a.WIS + a.CHA
}

func FromMap(m map[string]int) AbilityScores {
	var a AbilityScores
	for k, v := range m {
		a.AddKey(k, v)
	}
	return a
}

func (a AbilityScores) Map() map[string]int {
	return map[string]int{
		"str": a.STR, "dex": a.DEX, "con": a.CON,
		"int": a.INT, "wis": a.WIS, "cha": a.CHA,
	}
}

// Modifier is (score-10)/2, rounding down.
func Modifier(score int) int {
	m := score - 10
	if m >= 0 {
		return m / 2
	}
	return (m - 1) / 2
}

// ProficiencyBonus is 2 + floor((level-1)/4), clamped to levels 1–20.
func ProficiencyBonus(totalLevel int) int {
	if totalLevel < 1 {
		totalLevel = 1
	}
	if totalLevel > 20 {
		totalLevel = 20
	}
	return 2 + (totalLevel-1)/4
}

// AverageHitPoints is the PHB average: 1 + hitDie/2 (e.g. d10 → 6).
func AverageHitPoints(hitDie int) int {
	return 1 + hitDie/2
}

func ValidAbilityKey(key string) bool {
	switch strings.ToLower(key) {
	case "str", "dex", "con", "int", "wis", "cha":
		return true
	default:
		return false
	}
}
