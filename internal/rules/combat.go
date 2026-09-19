package rules

// Effect is a dismissible buff, debuff, or condition on a live sheet.
type Effect struct {
	ID                int64
	Slug              string
	Kind              string
	NameEN            string
	NameRU            string
	SourceSpellID     int64
	Source            string
	DurationKey       string
	FormulaEN         string
	FormulaRU         string
	ACBonus           int
	ACBase            int
	ACFloor           int
	SpeedBonus        int
	SpeedMult         int
	TempHP            int
	Tags              string
	Hidden            bool
	RemoveOnBattleEnd bool
	DurationTurns     int
	DamageFormula     string
	DamageType        string
	SourceURL         string
}

func (e Effect) Name(lang string) string {
	if lang == "ru" && e.NameRU != "" {
		return e.NameRU
	}
	if e.NameEN != "" {
		return e.NameEN
	}
	return e.NameRU
}

func (e Effect) Formula(lang string) string {
	if lang == "ru" && e.FormulaRU != "" {
		return e.FormulaRU
	}
	if e.FormulaEN != "" {
		return e.FormulaEN
	}
	return e.FormulaRU
}

func (e Effect) HasTag(tag string) bool {
	for _, p := range splitCSV(e.Tags) {
		if p == tag {
			return true
		}
	}
	return false
}

func splitCSV(s string) []string {
	var out []string
	cur := ""
	for _, r := range s {
		if r == ',' {
			if cur != "" {
				out = append(out, cur)
				cur = ""
			}
			continue
		}
		if r == ' ' && cur == "" {
			continue
		}
		cur += string(r)
	}
	if cur != "" {
		out = append(out, cur)
	}
	return out
}

// CombatInput is the snapshot used to derive AC, speed, and related notes.
type CombatInput struct {
	Scores       AbilityScores
	Classes      []ClassLevel
	RaceSpeed    int
	HPCurrent    int
	HPMax        int
	TempHP       int
	DeathSuccess int
	DeathFail    int
	Effects      []Effect
	Gear         []GearPiece
}

// CombatStats is what the sheet vitals block shows.
type CombatStats struct {
	HPCurrent    int
	HPMax        int
	TempHP       int
	AC           int
	ACNote       string
	Speed        int
	SpeedNote    string
	DeathSuccess int
	DeathFail    int
	SuccessPips  []bool
	FailPips     []bool
	Effects      []Effect
	Down         bool
}

// DeriveCombat computes AC and speed from unarmored features, worn gear, and effects.
func DeriveCombat(in CombatInput) CombatStats {
	dex := Modifier(in.Scores.DEX)
	armored := WearingArmor(in.Gear)
	shielded := WearingShield(in.Gear)
	ac := 10 + dex
	acNote := "character.combat.ac.dex"
	if !armored {
		if n := classLevels(in.Classes, "monk"); n >= 1 && !shielded {
			ud := 10 + dex + Modifier(in.Scores.WIS)
			if ud > ac {
				ac, acNote = ud, "character.combat.ac.monk"
			}
		}
		if n := classLevels(in.Classes, "barbarian"); n >= 1 {
			ud := 10 + dex + Modifier(in.Scores.CON)
			if ud > ac {
				ac, acNote = ud, "character.combat.ac.barb"
			}
		}
	}
	if armor, ok := wornArmor(in.Gear); ok {
		base := armor.ACBase
		if base <= 0 {
			base = 10
		}
		ac = base + armorDexBonus(dex, armor.DexMax, armor.Category())
		acNote = "character.combat.ac.armor"
	}
	if sh, ok := wornShield(in.Gear); ok {
		bonus := sh.ACBase
		if bonus == 0 {
			bonus = 2
		}
		ac += bonus
		if acNote == "character.combat.ac.dex" || acNote == "character.combat.ac.barb" || acNote == "character.combat.ac.monk" {
			if armored {
				acNote = "character.combat.ac.armor"
			} else {
				acNote = "character.combat.ac.shield"
			}
		}
	}
	for _, e := range in.Effects {
		if e.Hidden {
			continue
		}
		if e.ACBase > 0 && !armored {
			mage := e.ACBase + dex
			if mage > ac {
				ac, acNote = mage, "character.combat.ac.mage"
			}
		}
		ac += e.ACBonus
		if e.ACFloor > ac {
			ac = e.ACFloor
			acNote = "character.combat.ac.floor"
		}
	}
	for _, g := range in.Gear {
		ac += g.ACBonus
		if g.ACFloor > ac {
			ac = g.ACFloor
			acNote = "character.combat.ac.floor"
		}
	}

	speed := in.RaceSpeed
	if speed <= 0 {
		speed = 30
	}
	speedNote := "character.combat.speed.race"
	if !armored && !shielded {
		if bonus := monkUnarmoredMovement(classLevels(in.Classes, "monk")); bonus > 0 {
			speed += bonus
			speedNote = "character.combat.speed.monk"
		}
	}
	mult := 1
	for _, e := range in.Effects {
		if e.Hidden {
			continue
		}
		speed += e.SpeedBonus
		if e.SpeedMult > mult {
			mult = e.SpeedMult
		}
	}
	for _, g := range in.Gear {
		speed += g.SpeedBonus
		if g.SpeedMult > mult {
			mult = g.SpeedMult
		}
	}
	if mult > 1 {
		speed *= mult
		speedNote = "character.combat.speed.haste"
	}

	st := CombatStats{
		HPCurrent: in.HPCurrent, HPMax: in.HPMax, TempHP: SumTempHP(in.Effects),
		AC: ac, ACNote: acNote, Speed: speed, SpeedNote: speedNote,
		DeathSuccess: clampDeath(in.DeathSuccess), DeathFail: clampDeath(in.DeathFail),
		Effects: in.Effects, Down: in.HPCurrent <= 0,
	}
	st.SuccessPips = deathPips(st.DeathSuccess)
	st.FailPips = deathPips(st.DeathFail)
	return st
}

func monkUnarmoredMovement(monkLevel int) int {
	switch {
	case monkLevel >= 18:
		return 30
	case monkLevel >= 14:
		return 25
	case monkLevel >= 10:
		return 20
	case monkLevel >= 6:
		return 15
	case monkLevel >= 2:
		return 10
	default:
		return 0
	}
}

func clampDeath(n int) int {
	if n < 0 {
		return 0
	}
	if n > 3 {
		return 3
	}
	return n
}

func deathPips(n int) []bool {
	return []bool{n >= 1, n >= 2, n >= 3}
}

// ToggleDeathPip sets successes or failures to pip (1–3), or pip-1 if already at least that many.
func ToggleDeathPip(current, pip int) int {
	pip = clampDeath(pip)
	if pip == 0 {
		return 0
	}
	if current >= pip {
		return pip - 1
	}
	return pip
}

func ClampHP(cur, max int) int {
	if max < 0 {
		max = 0
	}
	if cur < 0 {
		return 0
	}
	if cur > max {
		return max
	}
	return cur
}

func ClampTempHP(n int) int {
	if n < 0 {
		return 0
	}
	return n
}

// SumTempHP is the house rule: temporary hit points from different statuses stack.
func SumTempHP(effects []Effect) int {
	n := 0
	for _, e := range effects {
		if e.Hidden {
			continue
		}
		if e.TempHP > 0 {
			n += e.TempHP
		}
	}
	return n
}

const OtherTempSlug = "other-temp"

func OtherTempEffect(amount int) Effect {
	e := Effect{
		Slug: OtherTempSlug, Kind: "buff", Source: SourceOther,
		NameEN: "Temporary hit points", NameRU: "Временные хиты",
		TempHP: amount, DurationKey: "character.status.dur.until_temp", Tags: "temp_hp",
	}
	e.FormulaEN, e.FormulaRU = EffectFormula(e)
	return e
}

// ReduceStackedTemp spends temp HP from statuses (other pool first, then newest).
func ReduceStackedTemp(effects []Effect, amount int) []Effect {
	if amount <= 0 {
		return effects
	}
	order := make([]int, 0, len(effects))
	for i, e := range effects {
		if e.Slug == OtherTempSlug && e.TempHP > 0 {
			order = append(order, i)
		}
	}
	for i := len(effects) - 1; i >= 0 && amount > 0; i-- {
		if effects[i].TempHP > 0 && effects[i].Slug != OtherTempSlug {
			order = append(order, i)
		}
	}
	for _, i := range order {
		if amount <= 0 {
			break
		}
		take := effects[i].TempHP
		if take > amount {
			take = amount
		}
		effects[i].TempHP -= take
		amount -= take
		effects[i].FormulaEN, effects[i].FormulaRU = EffectFormula(effects[i])
	}
	var out []Effect
	for _, e := range effects {
		if e.TempHP <= 0 && (e.Slug == OtherTempSlug || (e.HasTag("temp_hp") && e.ACBonus == 0 && e.ACBase == 0 && e.ACFloor == 0 && e.SpeedMult <= 1 && !e.HasTag("sanctuary"))) {
			continue
		}
		if e.TempHP < 0 {
			e.TempHP = 0
		}
		out = append(out, e)
	}
	return out
}
