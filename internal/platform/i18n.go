package platform

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Bundle holds en/ru message maps loaded from locales/*.json.
type Bundle struct {
	messages map[string]map[string]string
}

// LoadBundle reads locales/en.json and locales/ru.json.
func LoadBundle(dir string) (*Bundle, error) {
	b := &Bundle{messages: map[string]map[string]string{}}
	for _, lang := range []string{"en", "ru"} {
		raw, err := os.ReadFile(filepath.Join(dir, lang+".json"))
		if err != nil {
			return nil, fmt.Errorf("load locale %s: %w", lang, err)
		}
		var m map[string]string
		if err := json.Unmarshal(raw, &m); err != nil {
			return nil, fmt.Errorf("parse locale %s: %w", lang, err)
		}
		b.messages[lang] = m
	}
	return b, nil
}

// T looks up key in lang, then English, then returns the key itself.
func (b *Bundle) T(lang, key string) string {
	if lang != "en" && lang != "ru" {
		lang = "en"
	}
	if s, ok := b.messages[lang][key]; ok {
		return s
	}
	if s, ok := b.messages["en"][key]; ok {
		return s
	}
	return key
}

func NormalizeLang(lang string) string {
	lang = strings.ToLower(strings.TrimSpace(lang))
	if strings.HasPrefix(lang, "ru") {
		return "ru"
	}
	return "en"
}

func AcceptLanguage(header string) string {
	header = strings.ToLower(header)
	// Prefer the first supported tag. "ru" before "en" only if ru is listed first
	// or has a higher q — keep it simple: ru if the header starts with ru or contains ru- before en.
	parts := strings.Split(header, ",")
	best := "en"
	bestQ := -1.0
	for _, p := range parts {
		p = strings.TrimSpace(p)
		tag, q := p, 1.0
		if i := strings.Index(p, ";"); i >= 0 {
			tag = strings.TrimSpace(p[:i])
			if j := strings.Index(p, "q="); j >= 0 {
				fmt.Sscanf(p[j+2:], "%f", &q)
			}
		}
		lang := NormalizeLang(tag)
		if q > bestQ {
			bestQ = q
			best = lang
		}
	}
	if bestQ < 0 {
		return "en"
	}
	return best
}
