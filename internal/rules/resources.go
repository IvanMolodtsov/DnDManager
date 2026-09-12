package rules

import (
	"sort"
	"strconv"
)

const (
	KindSlots   = "spell_slots"
	KindPact    = "pact"
	KindKi      = "ki"
	KindSorcery = "sorcery"
	KindChannel = "channel_divinity"
)

// ClassLevel is the class-slug snapshot used to compute resource maxima.
type ClassLevel struct {
	Slug         string
	Levels       int
	SubclassSlug string
}

// Pool is one tracked resource (a slot level, pact row, or point pool).
type Pool struct {
	Kind      string
	SlotLevel int
	Current   int
	Max       int
}

func (p Pool) Remaining() int { return p.Current }

// MaxResources is the 2014 PHB maxima for the given class levels.
// Pact slots come from Warlock class levels only and are never mixed into
// the multiclass spell-slot table.
func MaxResources(classes []ClassLevel) []Pool {
	var out []Pool
	caster := multiclassCasterLevel(classes)
	if caster > 0 {
		slots := fullCasterSlots[caster]
		for lv, n := range slots {
			if n > 0 {
				out = append(out, Pool{Kind: KindSlots, SlotLevel: lv + 1, Current: n, Max: n})
			}
		}
	}
	if pact := pactMagic(classLevels(classes, "warlock")); pact.Max > 0 {
		out = append(out, pact)
	}
	if ki := monkKi(classLevels(classes, "monk")); ki > 0 {
		out = append(out, Pool{Kind: KindKi, Current: ki, Max: ki})
	}
	if sp := sorceryPoints(classLevels(classes, "sorcerer")); sp > 0 {
		out = append(out, Pool{Kind: KindSorcery, Current: sp, Max: sp})
	}
	if cd := channelDivinity(classes); cd > 0 {
		out = append(out, Pool{Kind: KindChannel, Current: cd, Max: cd})
	}
	return out
}

func classLevels(classes []ClassLevel, slug string) int {
	for _, c := range classes {
		if c.Slug == slug {
			return c.Levels
		}
	}
	return 0
}

func multiclassCasterLevel(classes []ClassLevel) int {
	n := 0
	for _, c := range classes {
		switch c.Slug {
		case "bard", "cleric", "druid", "sorcerer", "wizard":
			n += c.Levels
		case "paladin", "ranger":
			n += c.Levels / 2
		case "fighter":
			if c.SubclassSlug == "eldritch-knight" {
				n += c.Levels / 3
			}
		case "rogue":
			if c.SubclassSlug == "arcane-trickster" {
				n += c.Levels / 3
			}
			// warlock: pact magic only — do not add to this table
		}
	}
	if n > 20 {
		n = 20
	}
	return n
}

// fullCasterSlots[characterLevel][slotLevel-1] — PHB 2014 spellcasting table.
var fullCasterSlots = [21][9]int{
	1:  {2},
	2:  {3},
	3:  {4, 2},
	4:  {4, 3},
	5:  {4, 3, 2},
	6:  {4, 3, 3},
	7:  {4, 3, 3, 1},
	8:  {4, 3, 3, 2},
	9:  {4, 3, 3, 3, 1},
	10: {4, 3, 3, 3, 2},
	11: {4, 3, 3, 3, 2, 1},
	12: {4, 3, 3, 3, 2, 1},
	13: {4, 3, 3, 3, 2, 1, 1},
	14: {4, 3, 3, 3, 2, 1, 1},
	15: {4, 3, 3, 3, 2, 1, 1, 1},
	16: {4, 3, 3, 3, 2, 1, 1, 1},
	17: {4, 3, 3, 3, 2, 1, 1, 1, 1},
	18: {4, 3, 3, 3, 3, 1, 1, 1, 1},
	19: {4, 3, 3, 3, 3, 2, 1, 1, 1},
	20: {4, 3, 3, 3, 3, 2, 2, 1, 1},
}

func pactMagic(warlockLevel int) Pool {
	if warlockLevel < 1 {
		return Pool{Kind: KindPact}
	}
	slots, level := 1, 1
	switch {
	case warlockLevel >= 17:
		slots, level = 4, 5
	case warlockLevel >= 11:
		slots, level = 3, 5
	case warlockLevel >= 9:
		slots, level = 2, 5
	case warlockLevel >= 7:
		slots, level = 2, 4
	case warlockLevel >= 5:
		slots, level = 2, 3
	case warlockLevel >= 3:
		slots, level = 2, 2
	case warlockLevel >= 2:
		slots, level = 2, 1
	}
	return Pool{Kind: KindPact, SlotLevel: level, Current: slots, Max: slots}
}

func monkKi(monkLevel int) int {
	if monkLevel < 2 {
		return 0
	}
	return monkLevel
}

func sorceryPoints(sorcererLevel int) int {
	if sorcererLevel < 2 {
		return 0
	}
	return sorcererLevel
}

func channelDivinity(classes []ClassLevel) int {
	best := 0
	for _, c := range classes {
		n := 0
		switch c.Slug {
		case "cleric":
			switch {
			case c.Levels >= 18:
				n = 3
			case c.Levels >= 6:
				n = 2
			case c.Levels >= 2:
				n = 1
			}
		case "paladin":
			if c.Levels >= 3 {
				n = 1
			}
		}
		if n > best {
			best = n
		}
	}
	return best
}

// SyncPools aligns persisted current values to newly computed maxima.
// New pools start full. Raised max grants the difference. Lowered max clamps.
// Spent points persist until a long rest (not implemented here).
func SyncPools(existing, maxima []Pool) []Pool {
	have := indexPools(existing)
	var out []Pool
	for _, m := range maxima {
		if m.Max <= 0 {
			continue
		}
		next := m
		if cur, ok := have[poolKey(m)]; ok {
			delta := m.Max - cur.Max
			next.Current = cur.Current
			if delta > 0 {
				next.Current += delta
			}
			if next.Current > next.Max {
				next.Current = next.Max
			}
			if next.Current < 0 {
				next.Current = 0
			}
		} else {
			next.Current = m.Max
		}
		out = append(out, next)
	}
	sortPools(out)
	return out
}

// Consume spends amount from the matching pool. kind+slotLevel identifies
// the chosen bucket (pact vs spell slots are never shared).
func Consume(pools []Pool, kind string, slotLevel, amount int) ([]Pool, error) {
	if amount < 1 {
		amount = 1
	}
	out := append([]Pool{}, pools...)
	for i := range out {
		if out[i].Kind != kind {
			continue
		}
		if kind == KindSlots && out[i].SlotLevel != slotLevel {
			continue
		}
		if out[i].Current < amount {
			return nil, ErrResourceEmpty
		}
		out[i].Current -= amount
		return out, nil
	}
	return nil, ErrResourceEmpty
}

func FindPool(pools []Pool, kind string, slotLevel int) (Pool, bool) {
	for _, p := range pools {
		if p.Kind != kind {
			continue
		}
		if kind == KindSlots && p.SlotLevel != slotLevel {
			continue
		}
		return p, true
	}
	return Pool{}, false
}

func HasKind(pools []Pool, kind string) bool {
	for _, p := range pools {
		if p.Kind == kind && p.Max > 0 {
			return true
		}
	}
	return false
}

func SlotRemaining(pools []Pool, level int) int {
	p, ok := FindPool(pools, KindSlots, level)
	if !ok {
		return 0
	}
	return p.Current
}

func PactPool(pools []Pool) (Pool, bool) {
	return FindPool(pools, KindPact, 0)
}

func PoolsEqual(a, b []Pool) bool {
	if len(a) != len(b) {
		return false
	}
	ia, ib := indexPools(a), indexPools(b)
	if len(ia) != len(ib) {
		return false
	}
	for k, pa := range ia {
		pb, ok := ib[k]
		if !ok || pa.Current != pb.Current || pa.Max != pb.Max || pa.SlotLevel != pb.SlotLevel {
			return false
		}
	}
	return true
}

func indexPools(pools []Pool) map[string]Pool {
	m := make(map[string]Pool, len(pools))
	for _, p := range pools {
		m[poolKey(p)] = p
	}
	return m
}

func poolKey(p Pool) string {
	if p.Kind == KindSlots {
		return p.Kind + ":" + strconv.Itoa(p.SlotLevel)
	}
	return p.Kind
}

func sortPools(p []Pool) {
	rank := map[string]int{KindSlots: 1, KindPact: 2, KindKi: 3, KindSorcery: 4, KindChannel: 5}
	sort.Slice(p, func(i, j int) bool {
		if rank[p[i].Kind] != rank[p[j].Kind] {
			return rank[p[i].Kind] < rank[p[j].Kind]
		}
		return p[i].SlotLevel < p[j].SlotLevel
	})
}
