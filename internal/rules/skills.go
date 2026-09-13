package rules

import "fmt"

// Skill is one of the 18 PHB 5e (2014) skills.
type Skill struct {
	Slug      string
	Ability   string
	NameEN    string
	NameRU    string
	SourceURL string
}

// Skills is the PHB skill list in handbook order.
var Skills = []Skill{
	{Slug: "acrobatics", Ability: "dex", NameEN: "Acrobatics", NameRU: "Акробатика", SourceURL: "https://www.dnd5eapi.co/api/2014/skills/acrobatics"},
	{Slug: "animal-handling", Ability: "wis", NameEN: "Animal Handling", NameRU: "Уход за животными", SourceURL: "https://www.dnd5eapi.co/api/2014/skills/animal-handling"},
	{Slug: "arcana", Ability: "int", NameEN: "Arcana", NameRU: "Магия", SourceURL: "https://www.dnd5eapi.co/api/2014/skills/arcana"},
	{Slug: "athletics", Ability: "str", NameEN: "Athletics", NameRU: "Атлетика", SourceURL: "https://www.dnd5eapi.co/api/2014/skills/athletics"},
	{Slug: "deception", Ability: "cha", NameEN: "Deception", NameRU: "Обман", SourceURL: "https://www.dnd5eapi.co/api/2014/skills/deception"},
	{Slug: "history", Ability: "int", NameEN: "History", NameRU: "История", SourceURL: "https://www.dnd5eapi.co/api/2014/skills/history"},
	{Slug: "insight", Ability: "wis", NameEN: "Insight", NameRU: "Проницательность", SourceURL: "https://www.dnd5eapi.co/api/2014/skills/insight"},
	{Slug: "intimidation", Ability: "cha", NameEN: "Intimidation", NameRU: "Запугивание", SourceURL: "https://www.dnd5eapi.co/api/2014/skills/intimidation"},
	{Slug: "investigation", Ability: "int", NameEN: "Investigation", NameRU: "Расследование", SourceURL: "https://www.dnd5eapi.co/api/2014/skills/investigation"},
	{Slug: "medicine", Ability: "wis", NameEN: "Medicine", NameRU: "Медицина", SourceURL: "https://www.dnd5eapi.co/api/2014/skills/medicine"},
	{Slug: "nature", Ability: "int", NameEN: "Nature", NameRU: "Природа", SourceURL: "https://www.dnd5eapi.co/api/2014/skills/nature"},
	{Slug: "perception", Ability: "wis", NameEN: "Perception", NameRU: "Внимательность", SourceURL: "https://www.dnd5eapi.co/api/2014/skills/perception"},
	{Slug: "performance", Ability: "cha", NameEN: "Performance", NameRU: "Выступление", SourceURL: "https://www.dnd5eapi.co/api/2014/skills/performance"},
	{Slug: "persuasion", Ability: "cha", NameEN: "Persuasion", NameRU: "Убеждение", SourceURL: "https://www.dnd5eapi.co/api/2014/skills/persuasion"},
	{Slug: "religion", Ability: "int", NameEN: "Religion", NameRU: "Религия", SourceURL: "https://www.dnd5eapi.co/api/2014/skills/religion"},
	{Slug: "sleight-of-hand", Ability: "dex", NameEN: "Sleight of Hand", NameRU: "Ловкость рук", SourceURL: "https://www.dnd5eapi.co/api/2014/skills/sleight-of-hand"},
	{Slug: "stealth", Ability: "dex", NameEN: "Stealth", NameRU: "Скрытность", SourceURL: "https://www.dnd5eapi.co/api/2014/skills/stealth"},
	{Slug: "survival", Ability: "wis", NameEN: "Survival", NameRU: "Выживание", SourceURL: "https://www.dnd5eapi.co/api/2014/skills/survival"},
}

func (s Skill) Name(lang string) string {
	if lang == "ru" && s.NameRU != "" {
		return s.NameRU
	}
	return s.NameEN
}

// SkillBySlug returns a PHB skill or false.
func SkillBySlug(slug string) (Skill, bool) {
	for _, s := range Skills {
		if s.Slug == slug {
			return s, true
		}
	}
	return Skill{}, false
}

// SkillMark is persisted proficiency / expertise for one skill.
type SkillMark struct {
	Slug       string
	Proficient bool
	Expertise  bool
}

// SaveMark is persisted proficiency for one ability save.
type SaveMark struct {
	Ability    string
	Proficient bool
}

// SkillBonus is ability modifier + PB (×2 if expertise). Item bonuses are TODO.
func SkillBonus(scores AbilityScores, totalLevel int, mark SkillMark) int {
	sk, ok := SkillBySlug(mark.Slug)
	if !ok {
		return 0
	}
	bonus := Modifier(scores.Get(sk.Ability))
	if !mark.Proficient && !mark.Expertise {
		return bonus
	}
	pb := ProficiencyBonus(totalLevel)
	if mark.Expertise {
		return bonus + pb*2
	}
	return bonus + pb
}

// SaveBonus is ability modifier + PB if proficient. Item bonuses are TODO.
func SaveBonus(scores AbilityScores, totalLevel int, mark SaveMark) int {
	bonus := Modifier(scores.Get(mark.Ability))
	if mark.Proficient {
		bonus += ProficiencyBonus(totalLevel)
	}
	return bonus
}

// CheckFormula is "1d20+N" or "1d20-N" for IRL and server rolls.
func CheckFormula(bonus int) string {
	if bonus >= 0 {
		return fmt.Sprintf("1d20+%d", bonus)
	}
	return fmt.Sprintf("1d20%d", bonus)
}

// BackgroundSkillSlugs are PHB 2014 background skill grants.
var BackgroundSkillSlugs = map[string][]string{
	"acolyte":       {"insight", "religion"},
	"charlatan":     {"deception", "sleight-of-hand"},
	"criminal":      {"deception", "stealth"},
	"entertainer":   {"acrobatics", "performance"},
	"folk-hero":     {"animal-handling", "survival"},
	"guild-artisan": {"insight", "persuasion"},
	"hermit":        {"medicine", "religion"},
	"noble":         {"history", "persuasion"},
	"outlander":     {"athletics", "survival"},
	"sage":          {"arcana", "history"},
	"sailor":        {"athletics", "perception"},
	"soldier":       {"athletics", "intimidation"},
	"urchin":        {"sleight-of-hand", "stealth"},
}

// RaceSkillSlugs are automatic PHB skill grants (choice-based races omitted).
var RaceSkillSlugs = map[string][]string{
	"high-elf": {"perception"},
	"wood-elf": {"perception"},
	"half-orc": {"intimidation"},
}

// ClassSaveAbilities are PHB class saving throw proficiencies.
var ClassSaveAbilities = map[string][]string{
	"barbarian": {"str", "con"},
	"bard":      {"dex", "cha"},
	"cleric":    {"wis", "cha"},
	"druid":     {"int", "wis"},
	"fighter":   {"str", "con"},
	"monk":      {"str", "dex"},
	"paladin":   {"wis", "cha"},
	"ranger":    {"str", "dex"},
	"rogue":     {"dex", "int"},
	"sorcerer":  {"con", "cha"},
	"warlock":   {"wis", "cha"},
	"wizard":    {"int", "wis"},
}

// GrantedSkillSlugs unions background and race automatic skills.
func GrantedSkillSlugs(raceSlug, backgroundSlug string) []string {
	seen := map[string]bool{}
	var out []string
	add := func(list []string) {
		for _, s := range list {
			if !seen[s] {
				seen[s] = true
				out = append(out, s)
			}
		}
	}
	add(RaceSkillSlugs[raceSlug])
	add(BackgroundSkillSlugs[backgroundSlug])
	return out
}

// GrantedSaveAbilities unions saving throws from every class the character has.
func GrantedSaveAbilities(classSlugs []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, slug := range classSlugs {
		for _, a := range ClassSaveAbilities[slug] {
			if !seen[a] {
				seen[a] = true
				out = append(out, a)
			}
		}
	}
	return out
}
