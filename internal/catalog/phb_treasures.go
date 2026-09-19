package catalog

// PHBTreasure is one encoded DMG-style gem or art object (no HTML scrape).
type PHBTreasure struct {
	Kind   string // gem | art
	GP     int
	NameEN string
	NameRU string
}

func (t PHBTreasure) Name(lang string) string {
	return Pick(lang, t.NameEN, t.NameRU)
}

func (t PHBTreasure) Display(lang string) string {
	return t.Name(lang)
}

// PHBTreasures is a compact PHB/DMG gem and art list used when catalog has no treasure rows.
func PHBTreasures() []PHBTreasure {
	return []PHBTreasure{
		{Kind: "gem", GP: 10, NameEN: "Azurite (10 gp)", NameRU: "Лазурит (10 зм)"},
		{Kind: "gem", GP: 10, NameEN: "Banded agate (10 gp)", NameRU: "Полосчатый агат (10 зм)"},
		{Kind: "gem", GP: 10, NameEN: "Lapis lazuli (10 gp)", NameRU: "Лазурит-ляпис (10 зм)"},
		{Kind: "gem", GP: 10, NameEN: "Malachite (10 gp)", NameRU: "Малахит (10 зм)"},
		{Kind: "gem", GP: 10, NameEN: "Obsidian (10 gp)", NameRU: "Обсидиан (10 зм)"},
		{Kind: "gem", GP: 50, NameEN: "Bloodstone (50 gp)", NameRU: "Гелиотроп (50 зм)"},
		{Kind: "gem", GP: 50, NameEN: "Carnelian (50 gp)", NameRU: "Сердолик (50 зм)"},
		{Kind: "gem", GP: 50, NameEN: "Jasper (50 gp)", NameRU: "Яшма (50 зм)"},
		{Kind: "gem", GP: 50, NameEN: "Moonstone (50 gp)", NameRU: "Лунный камень (50 зм)"},
		{Kind: "gem", GP: 50, NameEN: "Zircon (50 gp)", NameRU: "Циркон (50 зм)"},
		{Kind: "gem", GP: 100, NameEN: "Amber (100 gp)", NameRU: "Янтарь (100 зм)"},
		{Kind: "gem", GP: 100, NameEN: "Amethyst (100 gp)", NameRU: "Аметист (100 зм)"},
		{Kind: "gem", GP: 100, NameEN: "Garnet (100 gp)", NameRU: "Гранат (100 зм)"},
		{Kind: "gem", GP: 100, NameEN: "Jade (100 gp)", NameRU: "Нефрит (100 зм)"},
		{Kind: "gem", GP: 100, NameEN: "Pearl (100 gp)", NameRU: "Жемчужина (100 зм)"},
		{Kind: "gem", GP: 500, NameEN: "Alexandrite (500 gp)", NameRU: "Александрит (500 зм)"},
		{Kind: "gem", GP: 500, NameEN: "Aquamarine (500 gp)", NameRU: "Аквамарин (500 зм)"},
		{Kind: "gem", GP: 500, NameEN: "Topaz (500 gp)", NameRU: "Топаз (500 зм)"},
		{Kind: "gem", GP: 1000, NameEN: "Black pearl (1,000 gp)", NameRU: "Чёрная жемчужина (1 000 зм)"},
		{Kind: "gem", GP: 1000, NameEN: "Emerald (1,000 gp)", NameRU: "Изумруд (1 000 зм)"},
		{Kind: "gem", GP: 5000, NameEN: "Diamond (5,000 gp)", NameRU: "Бриллиант (5 000 зм)"},
		{Kind: "gem", GP: 5000, NameEN: "Ruby (5,000 gp)", NameRU: "Рубин (5 000 зм)"},
		{Kind: "art", GP: 25, NameEN: "Silver ewer (25 gp)", NameRU: "Серебряный кувшин (25 зм)"},
		{Kind: "art", GP: 25, NameEN: "Carved bone statuette (25 gp)", NameRU: "Костяная статуэтка (25 зм)"},
		{Kind: "art", GP: 25, NameEN: "Gold bracelet (25 gp)", NameRU: "Золотой браслет (25 зм)"},
		{Kind: "art", GP: 25, NameEN: "Cloth-of-gold vestments (25 gp)", NameRU: "Облачение из золотой ткани (25 зм)"},
		{Kind: "art", GP: 250, NameEN: "Gold ring with bloodstones (250 gp)", NameRU: "Золотое кольцо с гелиотропами (250 зм)"},
		{Kind: "art", GP: 250, NameEN: "Carved ivory statuette (250 gp)", NameRU: "Статуэтка из слоновой кости (250 зм)"},
		{Kind: "art", GP: 250, NameEN: "Large gold bracelet (250 gp)", NameRU: "Массивный золотой браслет (250 зм)"},
		{Kind: "art", GP: 250, NameEN: "Silver-plated steel longsword (250 gp)", NameRU: "Длинный меч с серебрением (250 зм)"},
		{Kind: "art", GP: 750, NameEN: "Silver chalice with gold inlay (750 gp)", NameRU: "Серебряная чаша с золотой насечкой (750 зм)"},
		{Kind: "art", GP: 750, NameEN: "Painted gold war mask (750 gp)", NameRU: "Золотая боевая маска (750 зм)"},
		{Kind: "art", GP: 2500, NameEN: "Fine gold chain with a fire opal (2,500 gp)", NameRU: "Золотая цепь с огненным опалом (2 500 зм)"},
		{Kind: "art", GP: 2500, NameEN: "Old masterpiece painting (2,500 gp)", NameRU: "Старинный шедевр живописи (2 500 зм)"},
		{Kind: "art", GP: 7500, NameEN: "Jeweled gold crown (7,500 gp)", NameRU: "Золотая корона с самоцветами (7 500 зм)"},
		{Kind: "art", GP: 7500, NameEN: "Gold and ruby ring (7,500 gp)", NameRU: "Золотое кольцо с рубином (7 500 зм)"},
	}
}
