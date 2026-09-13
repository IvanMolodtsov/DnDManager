package catalog

// JewelryBaseIDStart is past 5eapi-only (1_000_000) and 5e14 numeric ids.
const JewelryBaseIDStart int64 = 2000001

// JewelryBase is a slot-only builder row (there is no PHB jewelry table).
type JewelryBase struct {
	ID     int64
	Slug   string
	NameEN string
	NameRU string
	Slot   string
}

// JewelryBases are the wearable slot templates for the jewelry wizard.
func JewelryBases() []JewelryBase {
	return []JewelryBase{
		{ID: 2000001, Slug: "ring", NameEN: "Ring", NameRU: "Кольцо", Slot: "ring"},
		{ID: 2000002, Slug: "neck", NameEN: "Amulet", NameRU: "Ожерелье", Slot: "neck"},
		{ID: 2000003, Slug: "cloak", NameEN: "Cloak", NameRU: "Плащ", Slot: "cloak"},
		{ID: 2000004, Slug: "head", NameEN: "Headwear", NameRU: "Головной убор", Slot: "head"},
		{ID: 2000005, Slug: "gloves", NameEN: "Gloves", NameRU: "Перчатки", Slot: "gloves"},
		{ID: 2000006, Slug: "boots", NameEN: "Boots", NameRU: "Сапоги", Slot: "boots"},
		{ID: 2000007, Slug: "belt", NameEN: "Belt", NameRU: "Пояс", Slot: "belt"},
	}
}

func JewelryBaseBySlug(slug string) (JewelryBase, bool) {
	for _, j := range JewelryBases() {
		if j.Slug == slug {
			return j, true
		}
	}
	return JewelryBase{}, false
}

func JewelryBaseSlugs() []string {
	rows := JewelryBases()
	out := make([]string, len(rows))
	for i, j := range rows {
		out[i] = j.Slug
	}
	return out
}
