package rules

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"dndmanager/internal/catalog"
)

const (
	FeatureKindItem     = "item"
	FeatureKindSpell    = "spell"
	FeatureKindProperty = "property"
	FeatureKindStat     = "stat"

	OriginBase    = "base"
	OriginCommon  = "common"
	OriginMagical = "magical"
	OriginUnique  = "unique"

	StatAtkBonus          = "atk_bonus"
	StatDmgBonus          = "dmg_bonus"
	StatExtraDice         = "extra_dice"
	StatCritRange         = "crit_range"
	StatACBonus           = "ac_bonus"
	StatACBase            = "ac_base"
	StatACFloor           = "ac_floor"
	StatSpeedBonus        = "speed_bonus"
	StatSpeedMult         = "speed_mult"
	StatVersatile         = "versatile"
	StatThrown            = "thrown"
	StatProperty          = "property"
	StatPersonality       = "personality"
	StatNote              = "note"
	StatSkillProficiency  = "skill_proficiency"
	StatResistance        = "resistance"
	StatVulnerability     = "vulnerability"
	StatImmunity          = "immunity"
	StatConditionImmunity = "condition_immunity"
	StatGrantCantrip      = "grant_cantrip"
	StatGrantSpell        = "grant_spell"
	StatAbilityBonus      = "ability_bonus"
	StatAbilityPenalty    = "ability_penalty"
)

// FeatureDTO is one stat mapping on a character item (not a wide overlay).
type FeatureDTO struct {
	ID          int64
	ItemID      int64
	SortOrder   int
	CatalogID   int64
	NameEN      string
	NameRU      string
	Stat        string
	Value       string
	Origin      string
	SourceURL   string
	CatalogKind string
	CatalogSlug string
}

func (f FeatureDTO) Name(lang string) string {
	return catalog.Pick(lang, f.NameEN, f.NameRU)
}

func (f FeatureDTO) Manual() bool {
	return f.Origin != OriginBase
}

func (f FeatureDTO) HasMechanics() bool {
	return strings.TrimSpace(f.Stat) != "" && (strings.TrimSpace(f.Value) != "" || f.Stat == StatProperty)
}

func (f FeatureDTO) SentientLine() string {
	if f.Stat != StatPersonality {
		return ""
	}
	return strings.TrimSpace(f.Value)
}

func (f FeatureDTO) IntValue() int {
	n, _ := strconv.Atoi(strings.TrimSpace(f.Value))
	return n
}

func (f FeatureDTO) ValueInputType() string {
	switch f.Stat {
	case StatAtkBonus, StatDmgBonus, StatACBonus, StatACBase, StatACFloor, StatSpeedBonus, StatSpeedMult, StatCritRange:
		return "number"
	default:
		return "text"
	}
}

// Picker is the configure-row control: select, ability (two fields), number, or text.
func (f FeatureDTO) Picker() string {
	switch f.Stat {
	case StatSkillProficiency:
		return "skill"
	case StatResistance, StatVulnerability, StatImmunity:
		return "damage"
	case StatConditionImmunity:
		return "condition"
	case StatGrantCantrip, StatGrantSpell:
		return "spell"
	case StatAbilityBonus, StatAbilityPenalty:
		return "ability"
	case StatAtkBonus, StatDmgBonus, StatACBonus, StatACBase, StatACFloor, StatSpeedBonus, StatSpeedMult, StatCritRange:
		return "number"
	case StatNote, StatPersonality:
		return "multiline"
	default:
		return "text"
	}
}

func (f FeatureDTO) GrantLevel() int {
	switch f.Stat {
	case StatGrantCantrip:
		return 0
	case StatGrantSpell:
		return trailingLevel(f.CatalogSlug)
	default:
		return -1
	}
}

func (f FeatureDTO) SpellID() int64 {
	if f.Stat != StatGrantSpell && f.Stat != StatGrantCantrip {
		return 0
	}
	n, _ := strconv.ParseInt(strings.TrimSpace(f.Value), 10, 64)
	return n
}

func (f FeatureDTO) AbilityKey() string {
	key, _ := ParseAbilityValue(f.Value)
	return key
}

func (f FeatureDTO) AbilityDelta() int {
	_, n := ParseAbilityValue(f.Value)
	return n
}

func trailingLevel(slug string) int {
	slug = strings.ToLower(strings.TrimSpace(slug))
	i := strings.LastIndex(slug, "-")
	if i < 0 || i+1 >= len(slug) {
		return 1
	}
	n, err := strconv.Atoi(slug[i+1:])
	if err != nil {
		return 1
	}
	return n
}

// ParseAbilityValue reads "str:2", "dex+2", or "cha:-1".
func ParseAbilityValue(raw string) (string, int) {
	raw = strings.ToLower(strings.TrimSpace(raw))
	if raw == "" {
		return "", 0
	}
	if i := strings.Index(raw, ":"); i > 0 {
		key := strings.TrimSpace(raw[:i])
		if ValidAbilityKey(key) {
			n, _ := strconv.Atoi(strings.TrimSpace(raw[i+1:]))
			return key, n
		}
	}
	for idx, r := range raw {
		if r == '+' || r == '-' {
			key := strings.TrimSpace(raw[:idx])
			if ValidAbilityKey(key) {
				n, _ := strconv.Atoi(raw[idx:])
				return key, n
			}
			break
		}
	}
	if ValidAbilityKey(raw) {
		return raw, 0
	}
	return "", 0
}

func (f FeatureDTO) Multiline() bool {
	return f.Stat == StatNote || f.Stat == StatPersonality
}

func (f FeatureDTO) PropertySlug() string {
	switch f.Stat {
	case StatVersatile, StatThrown:
		return f.Stat
	case StatProperty:
		return strings.ToLower(strings.TrimSpace(f.Value))
	default:
		return strings.ToLower(f.CatalogSlug)
	}
}

func HasProperty(feats []FeatureDTO, slug string) bool {
	slug = strings.ToLower(slug)
	for _, f := range feats {
		if f.PropertySlug() == slug {
			return true
		}
	}
	return false
}

// WeaponDTO is the typed payload for a weapon instance. Base hit lives here, not as a feature.
type WeaponDTO struct {
	Name          string
	BaseSlug      string
	BaseHit       string
	DamageType    string
	Simple        bool
	Martial       bool
	Melee         bool
	Ranged        bool
	Finesse       bool
	Thrown        bool
	VersatileDice string
	Features      []FeatureDTO
}

func WeaponFromItem(item catalog.Item, name string, feats []FeatureDTO) *WeaponDTO {
	ranged := item.IsRangedWeapon()
	return &WeaponDTO{
		Name: name, BaseSlug: item.Slug, BaseHit: item.DamageDice, DamageType: item.DamageType,
		Simple:  strings.EqualFold(item.WeaponCategory, "simple"),
		Martial: strings.EqualFold(item.WeaponCategory, "martial"),
		Melee:   !ranged, Ranged: ranged,
		Finesse:       item.HasProperty("finesse") || HasProperty(feats, "finesse"),
		Thrown:        item.HasProperty("thrown") || HasProperty(feats, "thrown"),
		VersatileDice: firstNonEmpty(item.VersatileDice, versatileDice(feats)),
		Features:      feats,
	}
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if s := strings.TrimSpace(v); s != "" {
			return s
		}
	}
	return ""
}

// DerivedItemStats is the sheet formula built from base hit plus one-stat features.
type DerivedItemStats struct {
	AtkBonus    int
	DmgBonus    int
	DamageDice  string
	DamageType  string
	Extra       []string
	CritRange   int
	ACBonus     int
	ACBase      int
	ACFloor     int
	SpeedBonus  int
	SpeedMult   int
	DamageLine  string
	Line        string
	RollFormula string
}

// InheritWeaponSlug maps a few magic stubs onto a mundane weapon for auto features.
// Moonblade is a longsword; d100 runes are still chosen by the player, not encoded here.
func InheritWeaponSlug(slug string) string {
	switch strings.ToLower(strings.TrimSpace(slug)) {
	case "moonblade":
		return "longsword"
	default:
		return ""
	}
}

func inheritItem(item catalog.Item, inherit *catalog.Item) catalog.Item {
	if inherit == nil {
		return item
	}
	src := item
	if src.DamageDice == "" && inherit.DamageDice != "" {
		src.DamageDice = inherit.DamageDice
		src.DamageType = inherit.DamageType
	}
	if len(src.Properties) == 0 {
		src.Properties = append([]string{}, inherit.Properties...)
	}
	if src.WeaponCategory == "" {
		src.WeaponCategory = inherit.WeaponCategory
	}
	if src.VersatileDice == "" {
		src.VersatileDice = inherit.VersatileDice
	}
	if src.RangeNormal == 0 {
		src.RangeNormal = inherit.RangeNormal
		src.RangeLong = inherit.RangeLong
	}
	return src
}

// CatalogItemFeatures builds one-stat features from structured catalog fields.
// Base dice stay on the weapon (WeaponDTO.BaseHit); this does not emit "Base damage".
func CatalogItemFeatures(item catalog.Item, inherit *catalog.Item) []FeatureDTO {
	src := inheritItem(item, inherit)
	if item.Consumable {
		return nil
	}
	url := item.Source5e14()
	if item.IsBase || src.IsBase {
		switch {
		case item.IsArmor() || src.IsArmor():
			url = catalog.ArmorTableURL
		case item.IsJewelry() || src.IsJewelry():
			url = ""
		default:
			url = catalog.ArmsTableURL
		}
	}
	var out []FeatureDTO
	if src.ACBase > 0 {
		nameEN, nameRU := "Armor", "Доспех"
		if item.IsShield() {
			nameEN, nameRU = "Shield", "Щит"
		}
		out = append(out, FeatureDTO{
			Origin: OriginBase, CatalogKind: FeatureKindProperty, CatalogSlug: "armor-class",
			NameEN: nameEN, NameRU: nameRU, Stat: StatACBase, Value: strconv.Itoa(src.ACBase),
			SourceURL: url,
		})
	}
	if cat := strings.ToLower(strings.TrimSpace(src.WeaponCategory)); cat != "" {
		en, ru := propertyName(cat)
		out = append(out, FeatureDTO{
			Origin: OriginBase, CatalogKind: FeatureKindProperty, CatalogSlug: cat,
			NameEN: en, NameRU: ru, Stat: StatProperty, Value: cat, SourceURL: url,
		})
	}
	if cat := strings.ToLower(strings.TrimSpace(src.ArmorCategory)); cat != "" && cat != "shield" {
		en, ru := propertyName(cat)
		out = append(out, FeatureDTO{
			Origin: OriginBase, CatalogKind: FeatureKindProperty, CatalogSlug: cat,
			NameEN: en, NameRU: ru, Stat: StatProperty, Value: cat, SourceURL: url,
		})
	}
	if src.StrMin > 0 {
		out = append(out, FeatureDTO{
			Origin: OriginBase, CatalogKind: FeatureKindProperty, CatalogSlug: "str-min",
			NameEN: "Strength " + strconv.Itoa(src.StrMin), NameRU: "Сила " + strconv.Itoa(src.StrMin),
			Stat: StatProperty, Value: "str-min-" + strconv.Itoa(src.StrMin), SourceURL: url,
		})
	}
	for _, p := range src.Properties {
		p = strings.ToLower(strings.TrimSpace(p))
		if p == "" || p == "shield" {
			continue
		}
		en, ru := propertyName(p)
		f := FeatureDTO{
			Origin: OriginBase, CatalogKind: FeatureKindProperty, CatalogSlug: p,
			NameEN: en, NameRU: ru, Stat: StatProperty, Value: p, SourceURL: url,
		}
		if p == "versatile" {
			f.Stat = StatVersatile
			f.Value = src.VersatileDice
		}
		if p == "thrown" {
			f.Stat = StatThrown
			if src.RangeLong > 0 {
				f.Value = fmt.Sprintf("%d/%d", src.RangeNormal, src.RangeLong)
			}
		}
		out = append(out, f)
	}
	if src.StealthDisadv {
		out = append(out, FeatureDTO{
			Origin: OriginBase, CatalogKind: FeatureKindProperty, CatalogSlug: "stealth-disadvantage",
			NameEN: "Stealth disadvantage", NameRU: "Помеха Скрытности",
			Stat: StatProperty, Value: "stealth-disadvantage", SourceURL: url,
		})
	}
	return out
}

func CommonPlusN(n int) []FeatureDTO {
	if n == 0 {
		return nil
	}
	v := strconv.Itoa(n)
	return []FeatureDTO{
		{NameEN: "Attack bonus", NameRU: "Бонус атаки", Stat: StatAtkBonus, Value: v, Origin: OriginCommon, CatalogKind: FeatureKindStat, CatalogSlug: "atk-bonus"},
		{NameEN: "Damage bonus", NameRU: "Бонус урона", Stat: StatDmgBonus, Value: v, Origin: OriginCommon, CatalogKind: FeatureKindStat, CatalogSlug: "dmg-bonus"},
	}
}

func CommonPlusAC(n int) []FeatureDTO {
	if n == 0 {
		return nil
	}
	return []FeatureDTO{{
		NameEN: "AC bonus", NameRU: "Бонус КД", Stat: StatACBonus, Value: strconv.Itoa(n),
		Origin: OriginCommon, CatalogKind: FeatureKindStat, CatalogSlug: "ac-bonus",
	}}
}

func CommonPlusFor(item catalog.Item, n int) []FeatureDTO {
	if item.IsArmor() || item.IsJewelry() {
		return CommonPlusAC(n)
	}
	return CommonPlusN(n)
}

func FeatureFromStat(sf catalog.StatFeature) FeatureDTO {
	origin := sf.Origin
	if origin == "" {
		origin = OriginUnique
	}
	return FeatureDTO{
		CatalogID: sf.ID, CatalogKind: FeatureKindStat, CatalogSlug: sf.Slug,
		NameEN: sf.NameEN, NameRU: sf.NameRU, Stat: sf.Stat, Value: sf.DefaultValue,
		Origin: origin, SourceURL: sf.SourceURL,
	}
}

// FeaturesFromItem unpacks a searched/pasted catalog item into one-stat rows.
// Vorpal becomes +3 atk, +3 dmg, and a decapitate note. Article HTML is never fetched.
func FeaturesFromItem(item catalog.Item) []FeatureDTO {
	meta := FeatureDTO{
		NameEN: item.NameEN, NameRU: item.NameRU, Origin: OriginMagical,
		SourceURL: item.Source5e14(), CatalogKind: FeatureKindItem, CatalogID: item.ID, CatalogSlug: item.Slug,
	}
	var out []FeatureDTO
	if n := MagicWeaponBonus(item.DescEN); n != 0 {
		a, d := meta, meta
		a.Stat, a.Value = StatAtkBonus, strconv.Itoa(n)
		d.Stat, d.Value = StatDmgBonus, strconv.Itoa(n)
		out = append(out, a, d)
	}
	if note := vorpalNote(item.DescEN); note != "" {
		n := meta
		n.Stat, n.Value = StatNote, note
		out = append(out, n)
	}
	if len(out) == 0 {
		n := meta
		n.Stat = StatNote
		n.Value = strings.TrimSpace(item.Rarity)
		if n.Value == "" {
			n.Value = item.NameEN
		}
		out = append(out, n)
	}
	return out
}

func FeatureFromSpell(sp catalog.Spell) FeatureDTO {
	url := sp.SourceURLRU
	if url == "" {
		url = sp.SourceURL
	}
	note := sp.StatsLine(sp.Level, 1)
	if note == "" {
		note = strings.TrimSpace(sp.School)
	}
	return FeatureDTO{
		Origin: OriginMagical, CatalogKind: FeatureKindSpell,
		CatalogID: sp.ID, CatalogSlug: sp.Slug,
		NameEN: sp.NameEN, NameRU: sp.NameRU, SourceURL: url,
		Stat: StatNote, Value: note,
	}
}

func FeaturesFromURL(ref DND14Ref) []FeatureDTO {
	name := titleFromSlug(ref.Slug)
	return []FeatureDTO{{
		NameEN: name, NameRU: name, Stat: StatNote, Value: ref.URL,
		Origin: OriginUnique, SourceURL: ref.URL, CatalogSlug: ref.Slug, CatalogKind: ref.Type,
	}}
}

func FeaturesFromUnknownURL(raw string) []FeatureDTO {
	raw = strings.TrimSpace(raw)
	name := raw
	if name == "" {
		name = "Feature"
	}
	return []FeatureDTO{{
		NameEN: name, NameRU: name, Stat: StatNote, Value: raw,
		Origin: OriginUnique, SourceURL: raw,
	}}
}

func vorpalNote(desc string) string {
	low := strings.ToLower(desc)
	if !strings.Contains(low, "cut off one of the creature's heads") && !strings.Contains(low, "you lop off") {
		return ""
	}
	note := "Decapitate on a 20"
	if strings.Contains(low, "ignores resistance to slashing") {
		note += "; ignores slashing resistance"
	}
	return note
}

// MagicWeaponBonus reads "+N bonus to attack and damage" from stored SRD text.
func MagicWeaponBonus(desc string) int {
	m := magicBonusRe.FindStringSubmatch(strings.ToLower(desc))
	if m == nil {
		return 0
	}
	n, _ := strconv.Atoi(m[1])
	return n
}

var magicBonusRe = regexp.MustCompile(`(?:you (?:gain|have) a |\+)\s*\+?(\d+)\s+bonus to attack and damage`)

// DeriveItemStats sums atk/dmg bonuses, keeps base dice/type, extra dice as extras, min crit range.
// Personality is display-only.
func DeriveItemStats(baseHit, damageType string, feats []FeatureDTO) DerivedItemStats {
	d := DerivedItemStats{DamageDice: baseHit, DamageType: damageType}
	for _, f := range feats {
		switch f.Stat {
		case StatAtkBonus:
			d.AtkBonus += f.IntValue()
		case StatDmgBonus:
			d.DmgBonus += f.IntValue()
		case StatACBonus:
			d.ACBonus += f.IntValue()
		case StatACBase:
			if n := f.IntValue(); n > 0 {
				d.ACBase = n
			}
		case StatACFloor:
			if n := f.IntValue(); n > d.ACFloor {
				d.ACFloor = n
			}
		case StatSpeedBonus:
			d.SpeedBonus += f.IntValue()
		case StatSpeedMult:
			if n := f.IntValue(); n > d.SpeedMult {
				d.SpeedMult = n
			}
		case StatExtraDice:
			if x := strings.TrimSpace(f.Value); x != "" {
				d.Extra = append(d.Extra, x)
			}
		case StatCritRange:
			if cr := f.IntValue(); cr > 0 && (d.CritRange == 0 || cr < d.CritRange) {
				d.CritRange = cr
			}
		}
	}
	d.RollFormula = d.DamageDice
	if d.DamageDice != "" && d.DmgBonus != 0 {
		d.RollFormula = d.DamageDice + formatSigned(d.DmgBonus)
	}
	d.DamageLine = d.RollFormula
	if d.DamageLine != "" && d.DamageType != "" {
		d.DamageLine += " " + d.DamageType
	}
	var parts []string
	if d.AtkBonus != 0 {
		parts = append(parts, formatSigned(d.AtkBonus))
	}
	if d.DamageLine != "" {
		parts = append(parts, d.DamageLine)
	}
	if d.ACBase > 0 {
		parts = append(parts, "AC "+strconv.Itoa(d.ACBase))
	}
	if d.ACBonus != 0 {
		parts = append(parts, formatSigned(d.ACBonus)+" AC")
	}
	for _, x := range d.Extra {
		if x != "" {
			parts = append(parts, x)
		}
	}
	if d.CritRange > 0 && d.CritRange < 20 {
		parts = append(parts, fmt.Sprintf("crit %d–20", d.CritRange))
	}
	d.Line = strings.Join(parts, " · ")
	return d
}

func formatSigned(n int) string {
	if n >= 0 {
		return "+" + strconv.Itoa(n)
	}
	return strconv.Itoa(n)
}

func propertyName(slug string) (en, ru string) {
	if p, ok := propertyNames[strings.ToLower(slug)]; ok {
		return p[0], p[1]
	}
	en = titleFromSlug(slug)
	return en, en
}

var propertyNames = map[string][2]string{
	"versatile":  {"Versatile", "Универсальное"},
	"two-handed": {"Two-handed", "Двуручное"},
	"finesse":    {"Finesse", "Фехтовальное"},
	"light":      {"Light", "Лёгкое"},
	"medium":     {"Medium", "Среднее"},
	"heavy":      {"Heavy", "Тяжёлое"},
	"reach":      {"Reach", "Досягаемость"},
	"thrown":     {"Thrown", "Метательное"},
	"ammunition": {"Ammunition", "Боеприпасы"},
	"loading":    {"Loading", "Перезарядка"},
	"special":    {"Special", "Особое"},
	"martial":    {"Martial", "Воинское"},
	"simple":     {"Simple", "Простое"},
	"monk":       {"Monk", "Монашеское"},
	"range":      {"Range", "Дальность"},
	"reload":     {"Reload", "Перезарядка"},
	"burst-fire": {"Burst fire", "Очередь"},
}

func titleFromSlug(slug string) string {
	slug = strings.ReplaceAll(strings.TrimSpace(slug), "-", " ")
	slug = strings.ReplaceAll(slug, "_", " ")
	if slug == "" {
		return "Feature"
	}
	parts := strings.Fields(slug)
	for i, p := range parts {
		if p == "" {
			continue
		}
		parts[i] = strings.ToUpper(p[:1]) + p[1:]
	}
	return strings.Join(parts, " ")
}

// DND14Ref is a parsed https://5e14.dnd.su/{type}/{id}-{slug}/ URL.
type DND14Ref struct {
	Type string
	ID   int64
	Slug string
	URL  string
}

var dnd14Re = regexp.MustCompile(`(?i)(?:https?://)?(?:www\.)?5e14\.dnd\.su/(items?|spells?|weapons?)/(\d+)-([^/\s?#]+)/?`)

func ParseDND14URL(raw string) (DND14Ref, bool) {
	raw = strings.TrimSpace(raw)
	m := dnd14Re.FindStringSubmatch(raw)
	if m == nil {
		return DND14Ref{}, false
	}
	id, _ := strconv.ParseInt(m[2], 10, 64)
	kind := strings.ToLower(m[1])
	switch kind {
	case "item":
		kind = "items"
	case "spell":
		kind = "spells"
	case "weapon", "weapons":
		kind = "items"
	}
	slug := strings.Trim(m[3], "/")
	url := "https://5e14.dnd.su/" + kind + "/" + m[2] + "-" + slug + "/"
	return DND14Ref{Type: kind, ID: id, Slug: slug, URL: url}, true
}
