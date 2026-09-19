package catalog

// ConditionsTableURL is the PHB 2014 Appendix A conditions article on 5e14.
// Individual condition cards are not a separate 5e14 index; all PHB rows
// point here (same pattern as the arms table). Do not invent /condition/{slug}/.
const ConditionsTableURL = "https://5e14.dnd.su/articles/mechanics/27-conditions/"

const conditionsAPIBase = "https://www.dnd5eapi.co/api/2014/conditions/"

// PHBCondition is one PHB 2014 Appendix A condition, or a common combat overlay.
type PHBCondition struct {
	Slug          string
	NameEN        string
	NameRU        string
	PHB           bool
	DamageFormula string
	DamageType    string
}

// PHBConditions is the 2014 Appendix A list plus Burning (DoT overlay, not a condition).
// Poisoned is disadvantage, not damage, unless the DM sets a formula.
func PHBConditions() []PHBCondition {
	return []PHBCondition{
		{Slug: "blinded", NameEN: "Blinded", NameRU: "Ослеплённый", PHB: true},
		{Slug: "charmed", NameEN: "Charmed", NameRU: "Очарованный", PHB: true},
		{Slug: "deafened", NameEN: "Deafened", NameRU: "Оглохший", PHB: true},
		{Slug: "exhaustion", NameEN: "Exhaustion", NameRU: "Истощённый", PHB: true},
		{Slug: "frightened", NameEN: "Frightened", NameRU: "Испуганный", PHB: true},
		{Slug: "grappled", NameEN: "Grappled", NameRU: "Схваченный", PHB: true},
		{Slug: "incapacitated", NameEN: "Incapacitated", NameRU: "Недееспособный", PHB: true},
		{Slug: "invisible", NameEN: "Invisible", NameRU: "Невидимый", PHB: true},
		{Slug: "paralyzed", NameEN: "Paralyzed", NameRU: "Парализованный", PHB: true},
		{Slug: "petrified", NameEN: "Petrified", NameRU: "Окаменевший", PHB: true},
		{Slug: "poisoned", NameEN: "Poisoned", NameRU: "Отравленный", PHB: true},
		{Slug: "prone", NameEN: "Prone", NameRU: "Сбитый с ног", PHB: true},
		{Slug: "restrained", NameEN: "Restrained", NameRU: "Опутанный", PHB: true},
		{Slug: "stunned", NameEN: "Stunned", NameRU: "Ошеломлённый", PHB: true},
		{Slug: "unconscious", NameEN: "Unconscious", NameRU: "Бессознательный", PHB: true},
		{Slug: "burning", NameEN: "Burning", NameRU: "Горение", DamageFormula: "1d6", DamageType: "fire"},
	}
}

func PHBConditionBySlug(slug string) (PHBCondition, bool) {
	for _, c := range PHBConditions() {
		if c.Slug == slug {
			return c, true
		}
	}
	return PHBCondition{}, false
}

func conditionAPIURL(slug string) string {
	if slug == "" || slug == "burning" {
		return ""
	}
	return conditionsAPIBase + slug
}
