package catalog

import (
	"strings"
	"unicode"
)

// PHB 2014 creature types (Monster Manual / PHB). Unusual stub types stay as stored.
var phbMonsterTypeRU = map[string]string{
	"aberration":  "Аберрация",
	"beast":       "Зверь",
	"celestial":   "Небожитель",
	"construct":   "Конструкт",
	"dragon":      "Дракон",
	"elemental":   "Элементаль",
	"fey":         "Фея",
	"fiend":       "Исчадие",
	"giant":       "Великан",
	"humanoid":    "Гуманоид",
	"monstrosity": "Монстр",
	"ooze":        "Слизь",
	"plant":       "Растение",
	"undead":      "Нежить",
}

// FormatMonsterType is title-case English, or the PHB RU name when lang is ru.
func FormatMonsterType(lang, typ string) string {
	typ = strings.TrimSpace(typ)
	if typ == "" {
		return ""
	}
	key := strings.ToLower(typ)
	if lang == "ru" {
		if ru, ok := phbMonsterTypeRU[key]; ok {
			return ru
		}
		return typ
	}
	if _, ok := phbMonsterTypeRU[key]; ok {
		return titleWords(key)
	}
	return titleWords(typ)
}

func titleWords(s string) string {
	parts := strings.Fields(s)
	for i, p := range parts {
		r := []rune(strings.ToLower(p))
		if len(r) > 0 {
			r[0] = unicode.ToTitle(r[0])
		}
		parts[i] = string(r)
	}
	return strings.Join(parts, " ")
}

// MonsterTypeOption is a distinct catalog_monsters.type value for the search filter.
type MonsterTypeOption struct {
	Slug string
}

func (o MonsterTypeOption) Name(lang string) string {
	return FormatMonsterType(lang, o.Slug)
}

func typeOptions(slugs []string) []MonsterTypeOption {
	out := make([]MonsterTypeOption, 0, len(slugs))
	for _, s := range slugs {
		out = append(out, MonsterTypeOption{Slug: s})
	}
	return out
}
