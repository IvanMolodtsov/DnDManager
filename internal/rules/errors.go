package rules

import "errors"

var (
	ErrUnknownIntent      = errors.New("unknown progression intent")
	ErrRaceRequired       = errors.New("race required")
	ErrBackgroundRequired = errors.New("background required")
	ErrClassRequired      = errors.New("class required")
	ErrSubclassRequired   = errors.New("subclass required")
	ErrSubclassInvalid    = errors.New("subclass does not match class")
	ErrArrayInvalid       = errors.New("ability scores must be the standard array")
	ErrASIRequired        = errors.New("ability score improvement required")
	ErrASIInvalid         = errors.New("ability score improvement must total +2")
	ErrHPRoll             = errors.New("hit point roll must be between 1 and the hit die")
	ErrMaxLevel           = errors.New("character is already level 20")
	ErrCatalog            = errors.New("catalog entry missing")
)
