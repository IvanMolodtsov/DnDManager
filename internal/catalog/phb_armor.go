package catalog

// ArmorTableURL is the PHB 2014 armor and shields table. Base-property links
// point here; the table itself is encoded below (no HTML scrape).
const ArmorTableURL = "https://5e14.dnd.su/articles/inventory/95-armor-and-shields/"

// PHBArmorRow is one row of the PHB armor/shields table.
type PHBArmorRow struct {
	Slug           string
	NameEN         string
	NameRU         string
	Category       string // light | medium | heavy | shield
	ACBase         int
	DexMax         int // -1 = unlimited DEX (light); 2 = medium cap; 0 = none
	StealthDisadv  bool
	StrMin         int
	SuggestedSlot  string
}

// PHBArmor is the PHB 2014 armor table (12 suits + shield).
func PHBArmor() []PHBArmorRow {
	return []PHBArmorRow{
		{Slug: "padded-armor", NameEN: "Padded Armor", NameRU: "Стёганый доспех", Category: "light", ACBase: 11, DexMax: -1, StealthDisadv: true, SuggestedSlot: "armor"},
		{Slug: "leather-armor", NameEN: "Leather Armor", NameRU: "Кожаный доспех", Category: "light", ACBase: 11, DexMax: -1, SuggestedSlot: "armor"},
		{Slug: "studded-leather-armor", NameEN: "Studded Leather Armor", NameRU: "Проклёпанный кожаный доспех", Category: "light", ACBase: 12, DexMax: -1, SuggestedSlot: "armor"},
		{Slug: "hide-armor", NameEN: "Hide Armor", NameRU: "Шкурный доспех", Category: "medium", ACBase: 12, DexMax: 2, SuggestedSlot: "armor"},
		{Slug: "chain-shirt", NameEN: "Chain Shirt", NameRU: "Кольчужная рубаха", Category: "medium", ACBase: 13, DexMax: 2, SuggestedSlot: "armor"},
		{Slug: "scale-mail", NameEN: "Scale Mail", NameRU: "Чешуйчатый доспех", Category: "medium", ACBase: 14, DexMax: 2, StealthDisadv: true, SuggestedSlot: "armor"},
		{Slug: "breastplate", NameEN: "Breastplate", NameRU: "Кираса", Category: "medium", ACBase: 14, DexMax: 2, SuggestedSlot: "armor"},
		{Slug: "half-plate-armor", NameEN: "Half Plate Armor", NameRU: "Полулаты", Category: "medium", ACBase: 15, DexMax: 2, StealthDisadv: true, SuggestedSlot: "armor"},
		{Slug: "ring-mail", NameEN: "Ring Mail", NameRU: "Колечный доспех", Category: "heavy", ACBase: 14, DexMax: 0, StealthDisadv: true, SuggestedSlot: "armor"},
		{Slug: "chain-mail", NameEN: "Chain Mail", NameRU: "Кольчуга", Category: "heavy", ACBase: 16, DexMax: 0, StealthDisadv: true, StrMin: 13, SuggestedSlot: "armor"},
		{Slug: "splint-armor", NameEN: "Splint Armor", NameRU: "Наборный доспех", Category: "heavy", ACBase: 17, DexMax: 0, StealthDisadv: true, StrMin: 15, SuggestedSlot: "armor"},
		{Slug: "plate-armor", NameEN: "Plate Armor", NameRU: "Латы", Category: "heavy", ACBase: 18, DexMax: 0, StealthDisadv: true, StrMin: 15, SuggestedSlot: "armor"},
		{Slug: "shield", NameEN: "Shield", NameRU: "Щит", Category: "shield", ACBase: 2, DexMax: 0, SuggestedSlot: "shield"},
	}
}

func PHBArmorBySlug(slug string) (PHBArmorRow, bool) {
	for _, a := range PHBArmor() {
		if a.Slug == slug {
			return a, true
		}
	}
	return PHBArmorRow{}, false
}

func PHBArmorSlugs() []string {
	rows := PHBArmor()
	out := make([]string, len(rows))
	for i, a := range rows {
		out[i] = a.Slug
	}
	return out
}
