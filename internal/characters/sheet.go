package characters

import (
	"strings"

	"dndmanager/internal/catalog"
	"dndmanager/internal/rules"
)

// PoolView is one ability-score-like box inside a resource group.
type PoolView struct {
	Kind      string
	Label     string
	SlotLevel int
	Current   int
	Max       int
}

// ResourceGroup is one labeled pool family on the sheet (never mixed).
type ResourceGroup struct {
	Kind      string
	TitleKey  string
	Pools     []PoolView
	Spendable bool
}

// SpellRow is a prepared (or learned) spell for the sheet.
type SpellRow struct {
	Spell    catalog.Spell
	Prepared bool
	Stats    string
	CanUse   bool
	SlotHint int
	FromItem string
	ItemKind bool
}

func resourceGroups(pools []rules.Pool) []ResourceGroup {
	order := []struct {
		kind, key string
		spend     bool
	}{
		{rules.KindSlots, "character.resources.slots", false},
		{rules.KindPact, "character.resources.pact", false},
		{rules.KindKi, "character.resources.ki", true},
		{rules.KindSorcery, "character.resources.sorcery", true},
		{rules.KindChannel, "character.resources.channel", true},
	}
	var out []ResourceGroup
	for _, o := range order {
		var rows []PoolView
		for _, p := range pools {
			if p.Kind != o.kind || p.Max <= 0 {
				continue
			}
			label := ""
			if o.kind == rules.KindSlots || o.kind == rules.KindPact {
				label = slotOrdinal(p.SlotLevel)
			}
			rows = append(rows, PoolView{
				Kind: p.Kind, Label: label, SlotLevel: p.SlotLevel, Current: p.Current, Max: p.Max,
			})
		}
		if len(rows) == 0 {
			continue
		}
		out = append(out, ResourceGroup{Kind: o.kind, TitleKey: o.key, Pools: rows, Spendable: o.spend})
	}
	return out
}

func displaySlot(ch *Character, sp catalog.Spell) int {
	if sp.Level == 0 {
		return 0
	}
	if p, ok := rules.PactPool(ch.Resources); ok && !rules.HasKind(ch.Resources, rules.KindSlots) {
		return p.SlotLevel
	}
	return sp.Level
}

func canCast(ch *Character, sp catalog.Spell) bool {
	if sp.Level == 0 {
		return true
	}
	if p, ok := rules.PactPool(ch.Resources); ok && p.Current > 0 && p.SlotLevel >= sp.Level {
		return true
	}
	for lv := sp.Level; lv <= 9; lv++ {
		if rules.SlotRemaining(ch.Resources, lv) > 0 {
			return true
		}
	}
	return false
}

func spellRows(ch *Character, preparedOnly bool, lang string) []SpellRow {
	var out []SpellRow
	if preparedOnly {
		for _, g := range ch.GrantedSpells {
			slot := g.Spell.Level
			if slot < 1 {
				slot = displaySlot(ch, g.Spell)
			}
			out = append(out, SpellRow{
				Spell: g.Spell, Prepared: true,
				Stats: g.Spell.StatsLine(slot, ch.Level),
				CanUse: true, SlotHint: slot,
				FromItem: g.ItemName(lang), ItemKind: true,
			})
		}
	}
	for _, ls := range ch.Spells {
		if preparedOnly && !ls.Prepared {
			continue
		}
		slot := displaySlot(ch, ls.Spell)
		out = append(out, SpellRow{
			Spell:    ls.Spell,
			Prepared: ls.Prepared,
			Stats:    ls.Spell.StatsLine(slot, ch.Level),
			CanUse:   ls.Prepared && canCast(ch, ls.Spell),
			SlotHint: slot,
		})
	}
	return out
}

// SkillRow is one derived skill on the sheet.
type SkillRow struct {
	Slug       string
	Name       string
	Ability    string
	Bonus      int
	Proficient bool
	Expertise  bool
	FromItem   bool
	Formula    string
	SourceURL  string
}

// SaveRow is one derived saving throw on the sheet.
type SaveRow struct {
	Ability    string
	Score      int
	Bonus      int
	Proficient bool
	Formula    string
}

func skillRows(ch *Character, lang string) []SkillRow {
	grants := ch.EquippedGrants()
	out := make([]SkillRow, 0, len(rules.Skills))
	for _, sk := range rules.Skills {
		m := ch.EffectiveSkillMark(sk.Slug)
		b := rules.SkillBonus(ch.Scores(), ch.Level, m)
		fromItem := grants.HasSkill(sk.Slug)
		out = append(out, SkillRow{
			Slug: sk.Slug, Name: catalog.Pick(lang, sk.NameEN, sk.NameRU), Ability: sk.Ability, Bonus: b,
			Proficient: m.Proficient, Expertise: m.Expertise, FromItem: fromItem,
			Formula: rules.CheckFormula(b), SourceURL: sk.SourceURL,
		})
	}
	return out
}

func saveRows(ch *Character) []SaveRow {
	marks := map[string]rules.SaveMark{}
	for _, m := range ch.SaveMarks {
		marks[strings.ToLower(m.Ability)] = m
	}
	out := make([]SaveRow, 0, len(rules.AbilityKeys))
	for _, ab := range rules.AbilityKeys {
		m := marks[ab]
		m.Ability = ab
		b := rules.SaveBonus(ch.Scores(), ch.Level, m)
		out = append(out, SaveRow{
			Ability: ab, Score: ch.Scores().Get(ab), Bonus: b, Proficient: m.Proficient, Formula: rules.CheckFormula(b),
		})
	}
	return out
}

func slotOrdinal(n int) string {
	switch n {
	case 1:
		return "1st"
	case 2:
		return "2nd"
	case 3:
		return "3rd"
	default:
		if n <= 0 {
			return ""
		}
		return itoa(n) + "th"
	}
}

type ItemRow struct {
	InventoryItem
	SlotKey string
	Stats   string
}

func equippedRows(ch *Character) []ItemRow {
	bySlot := map[string]InventoryItem{}
	for _, it := range ch.Inventory {
		if it.EquippedSlot != "" {
			bySlot[it.EquippedSlot] = it
		}
	}
	var out []ItemRow
	for _, slot := range rules.WearSlots {
		it, ok := bySlot[slot]
		if !ok {
			continue
		}
		out = append(out, ItemRow{InventoryItem: it, SlotKey: slot, Stats: itemRowStats(it)})
	}
	return out
}

func packRows(ch *Character) []ItemRow {
	var out []ItemRow
	for _, it := range ch.Inventory {
		if it.Equipped() || it.Item.Consumable {
			continue
		}
		out = append(out, ItemRow{InventoryItem: it, Stats: itemRowStats(it)})
	}
	return out
}

func consumableRows(ch *Character) []ItemRow {
	var out []ItemRow
	for _, it := range ch.Inventory {
		if it.Equipped() || !it.Item.Consumable {
			continue
		}
		out = append(out, ItemRow{InventoryItem: it, Stats: itemRowStats(it)})
	}
	return out
}

func itemRowStats(it InventoryItem) string {
	d := it.DerivedStats()
	line := d.Line
	if it.Armor != nil {
		parts := []string{}
		if it.Armor.Category != "" {
			parts = append(parts, it.Armor.Category)
		}
		if it.Armor.ACBase > 0 {
			parts = append(parts, "AC "+itoa(it.Armor.ACBase))
		}
		if d.ACBonus != 0 {
			parts = append(parts, formatSignedLocal(d.ACBonus)+" AC")
		}
		if it.Armor.StealthDisadv {
			parts = append(parts, "stealth disadv.")
		}
		line = strings.Join(parts, " · ")
	}
	if it.Jewelry != nil && line == "" {
		line = it.Jewelry.Slot
		if d.ACBonus != 0 {
			line += " · " + formatSignedLocal(d.ACBonus) + " AC"
		}
	}
	if line == "" {
		line = it.Item.StatsLine()
	}
	for _, f := range it.Features {
		if s := f.SentientLine(); s != "" {
			if line != "" {
				line += " · "
			}
			line += s
		}
	}
	return line
}

func formatSignedLocal(n int) string {
	if n >= 0 {
		return "+" + itoa(n)
	}
	return itoa(n)
}
