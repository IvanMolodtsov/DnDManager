package rules

import "strings"

const MaxAttunement = 3

const (
	SlotArmor    = "armor"
	SlotShield   = "shield"
	SlotMainHand = "main_hand"
	SlotOffHand  = "off_hand"
	SlotHead     = "head"
	SlotCloak    = "cloak"
	SlotNeck     = "neck"
	SlotRing1    = "ring_1"
	SlotRing2    = "ring_2"
	SlotGloves   = "gloves"
	SlotBoots    = "boots"
	SlotBelt     = "belt"
	SlotRing     = "ring"
)

// WearSlots is the exclusive wearable order shown on the sheet.
var WearSlots = []string{
	SlotArmor, SlotShield, SlotMainHand, SlotOffHand, SlotHead, SlotCloak, SlotNeck,
	SlotRing1, SlotRing2, SlotGloves, SlotBoots, SlotBelt,
}

// GearPiece is one equipped inventory row for AC/speed derivation.
type GearPiece struct {
	Slot           string
	Name           string
	ArmorCategory  string
	ACBase         int
	DexMax         int
	ACBonus        int
	ACFloor        int
	SpeedBonus     int
	SpeedMult      int
	TwoHanded      bool
	IsShield       bool
	RequiresAttune bool
	Attuned        bool
}

func (g GearPiece) Category() string {
	return strings.ToLower(strings.TrimSpace(g.ArmorCategory))
}

func (g GearPiece) Shield() bool {
	return g.IsShield || g.Slot == SlotShield || g.Category() == "shield"
}

func (g GearPiece) Armor() bool {
	if g.Shield() {
		return false
	}
	if g.Slot == SlotArmor {
		return true
	}
	switch g.Category() {
	case "light", "medium", "heavy":
		return true
	}
	return false
}

// SlotsTaken is every exclusive slot this piece occupies while equipped.
func SlotsTaken(g GearPiece) []string {
	if g.Shield() {
		return []string{SlotShield, SlotOffHand}
	}
	if g.TwoHanded || (g.Slot == SlotMainHand && g.TwoHanded) {
		return []string{SlotMainHand, SlotOffHand}
	}
	if g.Slot == "" {
		return nil
	}
	return []string{g.Slot}
}

func OccupiedMap(gear []GearPiece) map[string]GearPiece {
	occ := map[string]GearPiece{}
	for _, g := range gear {
		if g.Slot == "" {
			continue
		}
		for _, s := range SlotsTaken(g) {
			occ[s] = g
		}
	}
	return occ
}

func AttunementCount(gear []GearPiece) int {
	n := 0
	for _, g := range gear {
		if g.Attuned {
			n++
		}
	}
	return n
}

func PickRingSlot(occ map[string]GearPiece) (string, bool) {
	if _, taken := occ[SlotRing1]; !taken {
		return SlotRing1, true
	}
	if _, taken := occ[SlotRing2]; !taken {
		return SlotRing2, true
	}
	return "", false
}

func NormalizeEquipSlot(requested, suggested string, occ map[string]GearPiece) (string, bool) {
	slot := strings.TrimSpace(requested)
	if slot == "" {
		slot = strings.TrimSpace(suggested)
	}
	if slot == SlotRing || slot == "rings" {
		if picked, ok := PickRingSlot(occ); ok {
			return picked, true
		}
		return SlotRing1, true
	}
	return slot, slot != ""
}

// SlotOccupiedError names the other item blocking an exclusive slot.
type SlotOccupiedError struct {
	Slot      string
	OtherName string
}

func (e *SlotOccupiedError) Error() string {
	if e.OtherName != "" {
		return "slot occupied by " + e.OtherName
	}
	return ErrSlotOccupied.Error()
}

func (e *SlotOccupiedError) Unwrap() error { return ErrSlotOccupied }

func CheckEquip(current []GearPiece, adding GearPiece) error {
	return CheckEquipOpts(current, adding, EquipRules{})
}

func WearingArmor(gear []GearPiece) bool {
	for _, g := range gear {
		if g.Armor() {
			return true
		}
	}
	return false
}

func WearingShield(gear []GearPiece) bool {
	for _, g := range gear {
		if g.Shield() {
			return true
		}
	}
	return false
}

func armorDexBonus(dexMod, dexMax int, category string) int {
	switch strings.ToLower(category) {
	case "heavy", "shield":
		return 0
	case "medium":
		cap := 2
		if dexMax > 0 {
			cap = dexMax
		}
		if dexMod > cap {
			return cap
		}
		return dexMod
	default: // light or unknown with dex
		if dexMax == 0 && category != "light" && category != "" {
			return 0
		}
		if dexMax > 0 && dexMod > dexMax {
			return dexMax
		}
		return dexMod
	}
}

func wornArmor(gear []GearPiece) (GearPiece, bool) {
	for _, g := range gear {
		if g.Armor() {
			return g, true
		}
	}
	return GearPiece{}, false
}

func wornShield(gear []GearPiece) (GearPiece, bool) {
	for _, g := range gear {
		if g.Shield() {
			return g, true
		}
	}
	return GearPiece{}, false
}
