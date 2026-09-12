package rules

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"regexp"
	"strconv"
	"strings"
)

// RollResult is one server-side evaluation of a stored formula.
type RollResult struct {
	Formula string
	Total   int
	Parts   []string
}

var formulaTerm = regexp.MustCompile(`(?i)^\s*([+-])?\s*(?:(\d+)d(\d+)|(\d+))`)

// RollFormula evaluates expressions like "3d4+3", "8d6", "1d8+2".
func RollFormula(formula string) (RollResult, error) {
	formula = strings.TrimSpace(formula)
	if formula == "" {
		return RollResult{}, ErrBadFormula
	}
	rest := strings.ReplaceAll(formula, " ", "")
	rest = stripNonDiceNotes(rest)
	if rest == "" {
		return RollResult{}, ErrBadFormula
	}
	total := 0
	var parts []string
	for rest != "" {
		m := formulaTerm.FindStringSubmatch(rest)
		if m == nil {
			return RollResult{}, ErrBadFormula
		}
		sign := 1
		if m[1] == "-" {
			sign = -1
		}
		if m[2] != "" {
			n, _ := strconv.Atoi(m[2])
			sides, _ := strconv.Atoi(m[3])
			if n < 1 || n > 100 || sides < 2 || sides > 1000 {
				return RollResult{}, ErrBadFormula
			}
			sub := 0
			var dice []string
			for i := 0; i < n; i++ {
				v, err := randInt(sides)
				if err != nil {
					return RollResult{}, err
				}
				sub += v
				dice = append(dice, strconv.Itoa(v))
			}
			if sign < 0 {
				total -= sub
				parts = append(parts, fmt.Sprintf("-%dd%d [%s]", n, sides, strings.Join(dice, ",")))
			} else {
				total += sub
				parts = append(parts, fmt.Sprintf("%dd%d [%s]", n, sides, strings.Join(dice, ",")))
			}
		} else {
			v, _ := strconv.Atoi(m[4])
			total += sign * v
			if sign < 0 {
				parts = append(parts, "-"+strconv.Itoa(v))
			} else {
				parts = append(parts, "+"+strconv.Itoa(v))
			}
		}
		rest = rest[len(m[0]):]
	}
	return RollResult{Formula: formula, Total: total, Parts: parts}, nil
}

func stripNonDiceNotes(s string) string {
	s = strings.ToLower(s)
	for _, cut := range []string{"+mod", "+modifier", "+spellcastingmodifier"} {
		s = strings.ReplaceAll(s, cut, "")
	}
	return s
}

func randInt(max int) (int, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if err != nil {
		return 0, err
	}
	return int(n.Int64()) + 1, nil
}
