package rules

import (
	"strings"
	"unicode"
)

// VisibleEffects returns statuses the viewer may see. Hidden (curse) rows
// are DM-only and never included when includeHidden is false.
func VisibleEffects(effects []Effect, includeHidden bool) []Effect {
	var out []Effect
	for _, e := range effects {
		if e.Hidden && !includeHidden {
			continue
		}
		out = append(out, e)
	}
	return out
}

// HiddenEffects returns only hidden rows (DM curses).
func HiddenEffects(effects []Effect) []Effect {
	var out []Effect
	for _, e := range effects {
		if e.Hidden {
			out = append(out, e)
		}
	}
	return out
}

// DeriveEffects strips hidden statuses. Player-facing AC/speed/temp/rolls
// and ApplyDamage math never use hidden rows.
func DeriveEffects(effects []Effect) []Effect {
	return VisibleEffects(effects, false)
}

func (e Effect) TicksDamage() bool {
	if strings.TrimSpace(e.DamageFormula) == "" {
		return false
	}
	return ValidDamageType(e.DamageType)
}

func (e Effect) TurnLimited() bool {
	return e.DurationTurns > 0
}

// EndTurnEffects decrements turn-limited durations after that unit's turn.
// Duration 0 means not turn-limited. When a positive duration hits 0, the row is removed.
func EndTurnEffects(effects []Effect) []Effect {
	var out []Effect
	for _, e := range effects {
		if e.DurationTurns > 0 {
			e.DurationTurns--
			if e.DurationTurns <= 0 {
				continue
			}
		}
		out = append(out, e)
	}
	return out
}

// CustomStatusSlug builds a stable slug for a DM-typed name.
func CustomStatusSlug(name string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.TrimSpace(name) {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			b.WriteRune(unicode.ToLower(r))
			dash = false
			continue
		}
		if !dash && b.Len() > 0 {
			b.WriteByte('-')
			dash = true
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		out = "custom"
	}
	return "custom-" + out
}

// SanitizeSourceURL keeps http(s) links only (5e14 paste or catalog).
func SanitizeSourceURL(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	lower := strings.ToLower(s)
	if strings.HasPrefix(lower, "https://") || strings.HasPrefix(lower, "http://") {
		return s
	}
	return ""
}
