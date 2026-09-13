package catalog

// ArtifactsURL is the DMG 2014 artifact properties article. Mechanical rows
// are encoded below (no HTML scrape, no d100 roller).
const ArtifactsURL = "https://5e14.dnd.su/articles/inventory/139-artifacts/"

// StatFeatureSeed is one catalog_stat_features row (generic picker or DMG template).
type StatFeatureSeed struct {
	ID           int64
	Slug         string
	NameEN       string
	NameRU       string
	Stat         string
	DefaultValue string
	Origin       string
	SourceURL    string
	SortOrder    int
}

// PickerStatFeatures are the generic one-control pickers (skill, spell, …).
func PickerStatFeatures() []StatFeatureSeed {
	return []StatFeatureSeed{
		{20, "skill-proficiency", "Skill proficiency", "Владение навыком", "skill_proficiency", "stealth", "magical", "", 300},
		{21, "resistance", "Resistance", "Сопротивление", "resistance", "fire", "magical", "", 310},
		{22, "vulnerability", "Vulnerability", "Уязвимость", "vulnerability", "fire", "unique", "", 320},
		{23, "immunity", "Damage immunity", "Иммунитет к урону", "immunity", "poison", "magical", "", 330},
		{24, "condition-immunity", "Condition immunity", "Иммунитет к состоянию", "condition_immunity", "charmed", "magical", "", 340},
		{25, "grant-cantrip", "Cast a cantrip", "Заговор", "grant_cantrip", "", "magical", "", 350},
		{26, "grant-spell-1", "Cast a 1st-level spell", "Заклинание 1 уровня", "grant_spell", "", "magical", "", 351},
		{27, "grant-spell-2", "Cast a 2nd-level spell", "Заклинание 2 уровня", "grant_spell", "", "magical", "", 352},
		{28, "grant-spell-3", "Cast a 3rd-level spell", "Заклинание 3 уровня", "grant_spell", "", "magical", "", 353},
		{29, "grant-spell-4", "Cast a 4th-level spell", "Заклинание 4 уровня", "grant_spell", "", "magical", "", 354},
		{30, "grant-spell-5", "Cast a 5th-level spell", "Заклинание 5 уровня", "grant_spell", "", "magical", "", 355},
		{31, "grant-spell-6", "Cast a 6th-level spell", "Заклинание 6 уровня", "grant_spell", "", "magical", "", 356},
		{32, "grant-spell-7", "Cast a 7th-level spell", "Заклинание 7 уровня", "grant_spell", "", "magical", "", 357},
		{33, "ability-bonus", "Ability bonus", "Бонус характеристики", "ability_bonus", "str:2", "magical", "", 360},
		{34, "ability-penalty", "Ability penalty", "Штраф характеристики", "ability_penalty", "str:-1", "unique", "", 370},
		{35, "speed-bonus", "Speed bonus", "Бонус скорости", "speed_bonus", "10", "magical", "", 380},
	}
}

// ArtifactStatFeatures are named DMG artifact mechanical (and a few flavor-note) templates.
func ArtifactStatFeatures() []StatFeatureSeed {
	u := ArtifactsURL
	return []StatFeatureSeed{
		{40, "artifact-ac-1", "Artifact: +1 AC", "Артефакт: +1 КД", "ac_bonus", "1", "magical", u, 400},
		{41, "artifact-skill", "Artifact: skill proficiency", "Артефакт: владение навыком", "skill_proficiency", "stealth", "magical", u, 410},
		{42, "artifact-resistance", "Artifact: resistance", "Артефакт: сопротивление", "resistance", "fire", "magical", u, 420},
		{43, "artifact-cantrip", "Artifact: cantrip", "Артефакт: заговор", "grant_cantrip", "", "magical", u, 430},
		{44, "artifact-spell-1", "Artifact: 1st-level spell", "Артефакт: заклинание 1 ур.", "grant_spell", "", "magical", u, 431},
		{45, "artifact-spell-2", "Artifact: 2nd-level spell", "Артефакт: заклинание 2 ур.", "grant_spell", "", "magical", u, 432},
		{46, "artifact-spell-3", "Artifact: 3rd-level spell", "Артефакт: заклинание 3 ур.", "grant_spell", "", "magical", u, 433},
		{47, "artifact-spell-4", "Artifact: 4th-level spell", "Артефакт: заклинание 4 ур.", "grant_spell", "", "magical", u, 434},
		{48, "artifact-spell-5", "Artifact: 5th-level spell", "Артефакт: заклинание 5 ур.", "grant_spell", "", "magical", u, 435},
		{49, "artifact-spell-6", "Artifact: 6th-level spell", "Артефакт: заклинание 6 ур.", "grant_spell", "", "magical", u, 436},
		{50, "artifact-spell-7", "Artifact: 7th-level spell", "Артефакт: заклинание 7 ур.", "grant_spell", "", "magical", u, 437},
		{51, "artifact-charm-immunity", "Artifact: charm immunity", "Артефакт: иммунитет к очарованию", "condition_immunity", "charmed", "magical", u, 440},
		{52, "artifact-fear-immunity", "Artifact: fear immunity", "Артефакт: иммунитет к испугу", "condition_immunity", "frightened", "magical", u, 441},
		{53, "artifact-disease-immunity", "Artifact: disease immunity", "Артефакт: иммунитет к болезням", "immunity", "disease", "magical", u, 442},
		{54, "artifact-ability-plus-2", "Artifact: +2 to an ability (max 24)", "Артефакт: +2 к характеристике (макс. 24)", "ability_bonus", "str:2", "magical", u, 450},
		{55, "artifact-extra-1d6", "Artifact: extra 1d6 damage", "Артефакт: доп. 1d6 урона", "extra_dice", "1d6", "magical", u, 460},
		{56, "artifact-speed-10", "Artifact: +10 speed", "Артефакт: +10 скорости", "speed_bonus", "10", "magical", u, 470},
		{57, "artifact-regen", "Artifact: regeneration", "Артефакт: регенерация", "note", "While attuned, you regain 1d6 hit points at the start of your turn if you have at least 1 hit point.", "unique", u, 480},
		{58, "artifact-note-unattractive", "Artifact: unattractive", "Артефакт: непривлекательность", "note", "While attuned to this artifact, you have disadvantage on Charisma (Persuasion) checks made to socially impress others.", "unique", u, 500},
		{59, "artifact-note-glow", "Artifact: glow", "Артефакт: свечение", "note", "This artifact sheds dim light in a 5-foot radius while you are attuned to it. You cannot extinguish the light.", "unique", u, 510},
		{60, "artifact-note-whisper", "Artifact: whispers", "Артефакт: шёпот", "note", "While attuned, you hear occasional whispers in a language you do not know. The artifact has no mechanical effect beyond this flavor.", "unique", u, 520},
		{61, "artifact-note-hunger", "Artifact: hunger", "Артефакт: голод", "note", "While attuned, you must eat and drink twice as much as normal to avoid exhaustion from hunger or thirst.", "unique", u, 530},
	}
}
