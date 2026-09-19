package rules

import (
	"strconv"
	"strings"
)

const (
	DeadSlug = "dead"

	BodyHead = "head"
	BodyBody = "body"
	BodyArms = "arms"
	BodyLegs = "legs"
	BodyTail = "tail"

	StatSkillBonus             = "skill_bonus"
	StatSkillPenalty           = "skill_penalty"
	StatSize                   = "size"
	StatConditionVulnerability = "condition_vulnerability"
	StatGrantWeapon            = "grant_weapon"
	StatBlockSlot              = "block_slot"
	StatHands                  = "hands"
	StatAddPart                = "add_part"
	StatRemovePart             = "remove_part"

	HandsTwoHandAndShield = "two_hand_and_shield"
	HandsExtraOffhand     = "extra_offhand"

	MutationCRCap = 30
	ReviveCost    = 5000
)

// BodyParts is the exclusive mutation slot order.
var BodyParts = []string{BodyHead, BodyBody, BodyArms, BodyLegs, BodyTail}

// CreatureSizes is Tiny…Gargantuan in d6 order.
var CreatureSizes = []string{"tiny", "small", "medium", "large", "huge", "gargantuan"}

// PHBMonsterTypes is the d15 face list (1–14). Face 15 is DM chooses.
var PHBMonsterTypes = []string{
	"aberration", "beast", "celestial", "construct", "dragon", "elemental",
	"fey", "fiend", "giant", "humanoid", "monstrosity", "ooze", "plant", "undead",
}

func DeadEffect() Effect {
	return Effect{
		Slug: DeadSlug, Kind: "condition", Source: SourceOther,
		NameEN: "Dead", NameRU: "Мёртв",
	}
}

func HasDead(effects []Effect) bool {
	for _, e := range effects {
		if e.Slug == DeadSlug {
			return true
		}
	}
	return false
}

func ValidBodyPart(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	for _, p := range BodyParts {
		if p == s {
			return true
		}
	}
	return false
}

func ValidCreatureSize(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	for _, p := range CreatureSizes {
		if p == s {
			return true
		}
	}
	return false
}

func ValidPHBType(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	for _, p := range PHBMonsterTypes {
		if p == s {
			return true
		}
	}
	return false
}

// MapSizeDie maps a d6 face to a creature size.
func MapSizeDie(face int) string {
	if face < 1 || face > 6 {
		return ""
	}
	return CreatureSizes[face-1]
}

// MapTypeDie maps a d15 face. choose is true for face 15.
func MapTypeDie(face int) (slug string, choose bool) {
	if face == 15 {
		return "", true
	}
	if face < 1 || face > 14 {
		return "", false
	}
	return PHBMonsterTypes[face-1], false
}

// MapBodyDie maps a d6 face. choose is true for face 6 (player decides).
func MapBodyDie(face int) (part string, choose bool) {
	if face == 6 {
		return "", true
	}
	if face < 1 || face > 5 {
		return "", false
	}
	return BodyParts[face-1], false
}

// CRLabelFromInt is the catalog cr_label for a rolled integer CR.
func CRLabelFromInt(n int) string {
	if n <= 0 {
		return "0"
	}
	return strconv.Itoa(n)
}

func ClampMutationCR(n, maxLevel int) int {
	if n < 0 {
		n = 0
	}
	if maxLevel < 0 {
		maxLevel = 0
	}
	if n > maxLevel {
		n = maxLevel
	}
	return n
}

// ParseSkillValue reads "athletics:2" or "stealth:-1".
func ParseSkillValue(raw string) (string, int) {
	raw = strings.ToLower(strings.TrimSpace(raw))
	if raw == "" {
		return "", 0
	}
	slug, n := raw, 0
	if i := strings.Index(raw, ":"); i > 0 {
		slug = strings.TrimSpace(raw[:i])
		n, _ = strconv.Atoi(strings.TrimSpace(raw[i+1:]))
	}
	if _, ok := SkillBySlug(slug); !ok {
		return "", 0
	}
	return slug, n
}

// ParseWeaponGrant reads "Bite|1d8|piercing".
func ParseWeaponGrant(raw string) (name, dice, dmgType string) {
	parts := strings.Split(raw, "|")
	if len(parts) < 3 {
		return "", "", ""
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]), strings.ToLower(strings.TrimSpace(parts[2]))
}

func ParseHandsValue(raw string) HandsRules {
	raw = strings.ToLower(strings.TrimSpace(raw))
	var h HandsRules
	if raw == HandsTwoHandAndShield {
		h.TwoHandAndShield = true
		return h
	}
	if strings.HasPrefix(raw, HandsExtraOffhand) {
		n := 1
		if i := strings.LastIndex(raw, ":"); i > 0 {
			if v, err := strconv.Atoi(strings.TrimSpace(raw[i+1:])); err == nil && v > 0 {
				n = v
			}
		}
		h.ExtraOffhand = n
	}
	return h
}

type HandsRules struct {
	TwoHandAndShield bool
	ExtraOffhand     int
}

func (h HandsRules) Merge(o HandsRules) HandsRules {
	h.TwoHandAndShield = h.TwoHandAndShield || o.TwoHandAndShield
	h.ExtraOffhand += o.ExtraOffhand
	return h
}

type MutationAttack struct {
	NameEN     string
	NameRU     string
	Dice       string
	DamageType string
}

func (a MutationAttack) Name(lang string) string {
	if lang == "ru" && a.NameRU != "" {
		return a.NameRU
	}
	if a.NameEN != "" {
		return a.NameEN
	}
	return a.NameRU
}

type EquipRules struct {
	Hands   HandsRules
	Blocked []string
}

func SlotsTakenOpts(g GearPiece, h HandsRules) []string {
	if g.Shield() {
		if h.TwoHandAndShield {
			return []string{SlotShield}
		}
		return []string{SlotShield, SlotOffHand}
	}
	if g.TwoHanded {
		if h.TwoHandAndShield || h.ExtraOffhand > 0 {
			return []string{SlotMainHand}
		}
		return []string{SlotMainHand, SlotOffHand}
	}
	if g.Slot == "" {
		return nil
	}
	return []string{g.Slot}
}

func OccupiedMapOpts(gear []GearPiece, h HandsRules) map[string]GearPiece {
	occ := map[string]GearPiece{}
	for _, g := range gear {
		if g.Slot == "" {
			continue
		}
		for _, s := range SlotsTakenOpts(g, h) {
			if s == SlotOffHand && h.ExtraOffhand > 0 && !g.Shield() && !g.TwoHanded {
				continue
			}
			occ[s] = g
		}
	}
	return occ
}

func offHandCount(gear []GearPiece) int {
	n := 0
	for _, g := range gear {
		if g.Slot == SlotOffHand && !g.Shield() && !g.TwoHanded {
			n++
		}
	}
	return n
}

func CheckEquipOpts(current []GearPiece, adding GearPiece, opts EquipRules) error {
	occ := OccupiedMapOpts(current, opts.Hands)
	slot, ok := NormalizeEquipSlot(adding.Slot, adding.Slot, occ)
	if !ok {
		return ErrSlotRequired
	}
	adding.Slot = slot
	for _, b := range opts.Blocked {
		b = strings.ToLower(strings.TrimSpace(b))
		if b == "" {
			continue
		}
		for _, s := range SlotsTakenOpts(adding, opts.Hands) {
			if s == b {
				return &SlotOccupiedError{Slot: s, OtherName: b}
			}
		}
	}
	if adding.RequiresAttune && !adding.Attuned {
		if AttunementCount(current) >= MaxAttunement {
			return ErrAttunementFull
		}
	}
	if adding.Attuned && AttunementCount(current) >= MaxAttunement {
		return ErrAttunementFull
	}
	if adding.Slot == SlotOffHand && !adding.Shield() && !adding.TwoHanded && opts.Hands.ExtraOffhand > 0 {
		if offHandCount(current) >= 1+opts.Hands.ExtraOffhand {
			return &SlotOccupiedError{Slot: SlotOffHand, OtherName: "off-hand"}
		}
		return nil
	}
	for _, s := range SlotsTakenOpts(adding, opts.Hands) {
		if other, busy := occ[s]; busy {
			return &SlotOccupiedError{Slot: s, OtherName: other.Name}
		}
	}
	return nil
}

// MonsterHits is a search callback for PickMutationMonster.
type MonsterHits func(crLabel, size, typ string) []struct {
	ID   int64
	Name string
}

// BumpCRUntilHit increments integer CR until hits is non-empty or CR exceeds MutationCRCap.
func BumpCRUntilHit(start int, size, typ string, search func(crLabel, size, typ string) int) (label string, cr int, ok bool) {
	if start < 0 {
		start = 0
	}
	for cr = start; cr <= MutationCRCap; cr++ {
		label = CRLabelFromInt(cr)
		if search(label, size, typ) > 0 {
			return label, cr, true
		}
	}
	return "", start, false
}
