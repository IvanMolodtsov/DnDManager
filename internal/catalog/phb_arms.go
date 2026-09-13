package catalog

// ArmsTableURL is the PHB 2014 weapons table. Base-property links point here;
// the table itself is encoded below (no HTML scrape).
const ArmsTableURL = "https://5e14.dnd.su/articles/inventory/96-arms/"

// PHBArm is one row of the PHB arms table (simple/martial × melee/ranged).
type PHBArm struct {
	Slug          string
	NameEN        string
	NameRU        string
	Dice          string
	DamageType    string
	Category      string // simple | martial
	Melee         bool
	Ranged        bool
	Properties    []string
	VersatileDice string
	RangeNormal   int
	RangeLong     int
}

// PHBArms is the PHB 2014 weapon table (club through net), 37 rows.
func PHBArms() []PHBArm {
	return []PHBArm{
		{Slug: "club", NameEN: "Club", NameRU: "Дубинка", Dice: "1d4", DamageType: "bludgeoning", Category: "simple", Melee: true, Properties: []string{"light"}},
		{Slug: "dagger", NameEN: "Dagger", NameRU: "Кинжал", Dice: "1d4", DamageType: "piercing", Category: "simple", Melee: true, Properties: []string{"finesse", "light", "thrown"}, RangeNormal: 20, RangeLong: 60},
		{Slug: "greatclub", NameEN: "Greatclub", NameRU: "Палица", Dice: "1d8", DamageType: "bludgeoning", Category: "simple", Melee: true, Properties: []string{"two-handed"}},
		{Slug: "handaxe", NameEN: "Handaxe", NameRU: "Ручной топор", Dice: "1d6", DamageType: "slashing", Category: "simple", Melee: true, Properties: []string{"light", "thrown"}, RangeNormal: 20, RangeLong: 60},
		{Slug: "javelin", NameEN: "Javelin", NameRU: "Метательное копьё", Dice: "1d6", DamageType: "piercing", Category: "simple", Melee: true, Properties: []string{"thrown"}, RangeNormal: 30, RangeLong: 120},
		{Slug: "light-hammer", NameEN: "Light hammer", NameRU: "Лёгкий молот", Dice: "1d4", DamageType: "bludgeoning", Category: "simple", Melee: true, Properties: []string{"light", "thrown"}, RangeNormal: 20, RangeLong: 60},
		{Slug: "mace", NameEN: "Mace", NameRU: "Булава", Dice: "1d6", DamageType: "bludgeoning", Category: "simple", Melee: true},
		{Slug: "quarterstaff", NameEN: "Quarterstaff", NameRU: "Боевой посох", Dice: "1d6", DamageType: "bludgeoning", Category: "simple", Melee: true, Properties: []string{"versatile"}, VersatileDice: "1d8"},
		{Slug: "sickle", NameEN: "Sickle", NameRU: "Серп", Dice: "1d4", DamageType: "slashing", Category: "simple", Melee: true, Properties: []string{"light"}},
		{Slug: "spear", NameEN: "Spear", NameRU: "Копьё", Dice: "1d6", DamageType: "piercing", Category: "simple", Melee: true, Properties: []string{"thrown", "versatile"}, VersatileDice: "1d8", RangeNormal: 20, RangeLong: 60},
		{Slug: "crossbow-light", NameEN: "Crossbow, light", NameRU: "Лёгкий арбалет", Dice: "1d8", DamageType: "piercing", Category: "simple", Ranged: true, Properties: []string{"ammunition", "loading", "two-handed"}, RangeNormal: 80, RangeLong: 320},
		{Slug: "dart", NameEN: "Dart", NameRU: "Дротик", Dice: "1d4", DamageType: "piercing", Category: "simple", Ranged: true, Properties: []string{"finesse", "thrown"}, RangeNormal: 20, RangeLong: 60},
		{Slug: "shortbow", NameEN: "Shortbow", NameRU: "Короткий лук", Dice: "1d6", DamageType: "piercing", Category: "simple", Ranged: true, Properties: []string{"ammunition", "two-handed"}, RangeNormal: 80, RangeLong: 320},
		{Slug: "sling", NameEN: "Sling", NameRU: "Праща", Dice: "1d4", DamageType: "bludgeoning", Category: "simple", Ranged: true, Properties: []string{"ammunition"}, RangeNormal: 30, RangeLong: 120},
		{Slug: "battleaxe", NameEN: "Battleaxe", NameRU: "Боевой топор", Dice: "1d8", DamageType: "slashing", Category: "martial", Melee: true, Properties: []string{"versatile"}, VersatileDice: "1d10"},
		{Slug: "flail", NameEN: "Flail", NameRU: "Цеп", Dice: "1d8", DamageType: "bludgeoning", Category: "martial", Melee: true},
		{Slug: "glaive", NameEN: "Glaive", NameRU: "Глефа", Dice: "1d10", DamageType: "slashing", Category: "martial", Melee: true, Properties: []string{"heavy", "reach", "two-handed"}},
		{Slug: "greataxe", NameEN: "Greataxe", NameRU: "Секира", Dice: "1d12", DamageType: "slashing", Category: "martial", Melee: true, Properties: []string{"heavy", "two-handed"}},
		{Slug: "greatsword", NameEN: "Greatsword", NameRU: "Двуручный меч", Dice: "2d6", DamageType: "slashing", Category: "martial", Melee: true, Properties: []string{"heavy", "two-handed"}},
		{Slug: "halberd", NameEN: "Halberd", NameRU: "Алебарда", Dice: "1d10", DamageType: "slashing", Category: "martial", Melee: true, Properties: []string{"heavy", "reach", "two-handed"}},
		{Slug: "lance", NameEN: "Lance", NameRU: "Длинное копьё", Dice: "1d12", DamageType: "piercing", Category: "martial", Melee: true, Properties: []string{"reach", "special"}},
		{Slug: "longsword", NameEN: "Longsword", NameRU: "Длинный меч", Dice: "1d8", DamageType: "slashing", Category: "martial", Melee: true, Properties: []string{"versatile"}, VersatileDice: "1d10"},
		{Slug: "maul", NameEN: "Maul", NameRU: "Молот", Dice: "2d6", DamageType: "bludgeoning", Category: "martial", Melee: true, Properties: []string{"heavy", "two-handed"}},
		{Slug: "morningstar", NameEN: "Morningstar", NameRU: "Моргенштерн", Dice: "1d8", DamageType: "piercing", Category: "martial", Melee: true},
		{Slug: "pike", NameEN: "Pike", NameRU: "Пика", Dice: "1d10", DamageType: "piercing", Category: "martial", Melee: true, Properties: []string{"heavy", "reach", "two-handed"}},
		{Slug: "rapier", NameEN: "Rapier", NameRU: "Рапира", Dice: "1d8", DamageType: "piercing", Category: "martial", Melee: true, Properties: []string{"finesse"}},
		{Slug: "scimitar", NameEN: "Scimitar", NameRU: "Скимитар", Dice: "1d6", DamageType: "slashing", Category: "martial", Melee: true, Properties: []string{"finesse", "light"}},
		{Slug: "shortsword", NameEN: "Shortsword", NameRU: "Короткий меч", Dice: "1d6", DamageType: "piercing", Category: "martial", Melee: true, Properties: []string{"finesse", "light"}},
		{Slug: "trident", NameEN: "Trident", NameRU: "Трезубец", Dice: "1d6", DamageType: "piercing", Category: "martial", Melee: true, Properties: []string{"thrown", "versatile"}, VersatileDice: "1d8", RangeNormal: 20, RangeLong: 60},
		{Slug: "war-pick", NameEN: "War pick", NameRU: "Боевая кирка", Dice: "1d8", DamageType: "piercing", Category: "martial", Melee: true},
		{Slug: "warhammer", NameEN: "Warhammer", NameRU: "Боевой молот", Dice: "1d8", DamageType: "bludgeoning", Category: "martial", Melee: true, Properties: []string{"versatile"}, VersatileDice: "1d10"},
		{Slug: "whip", NameEN: "Whip", NameRU: "Кнут", Dice: "1d4", DamageType: "slashing", Category: "martial", Melee: true, Properties: []string{"finesse", "reach"}},
		{Slug: "blowgun", NameEN: "Blowgun", NameRU: "Духовая трубка", Dice: "1", DamageType: "piercing", Category: "martial", Ranged: true, Properties: []string{"ammunition", "loading"}, RangeNormal: 25, RangeLong: 100},
		{Slug: "crossbow-hand", NameEN: "Crossbow, hand", NameRU: "Ручной арбалет", Dice: "1d6", DamageType: "piercing", Category: "martial", Ranged: true, Properties: []string{"ammunition", "light", "loading"}, RangeNormal: 30, RangeLong: 120},
		{Slug: "crossbow-heavy", NameEN: "Crossbow, heavy", NameRU: "Тяжёлый арбалет", Dice: "1d10", DamageType: "piercing", Category: "martial", Ranged: true, Properties: []string{"ammunition", "heavy", "loading", "two-handed"}, RangeNormal: 100, RangeLong: 400},
		{Slug: "longbow", NameEN: "Longbow", NameRU: "Длинный лук", Dice: "1d8", DamageType: "piercing", Category: "martial", Ranged: true, Properties: []string{"ammunition", "heavy", "two-handed"}, RangeNormal: 150, RangeLong: 600},
		{Slug: "net", NameEN: "Net", NameRU: "Сеть", Dice: "", DamageType: "", Category: "martial", Ranged: true, Properties: []string{"thrown", "special"}, RangeNormal: 5, RangeLong: 15},
	}
}

func PHBArmBySlug(slug string) (PHBArm, bool) {
	for _, a := range PHBArms() {
		if a.Slug == slug {
			return a, true
		}
	}
	return PHBArm{}, false
}

func PHBArmSlugs() []string {
	arms := PHBArms()
	out := make([]string, len(arms))
	for i, a := range arms {
		out[i] = a.Slug
	}
	return out
}
