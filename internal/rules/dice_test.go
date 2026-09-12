package rules

import "testing"

func TestMagicMissileFormulaRoll(t *testing.T) {
	for i := 0; i < 40; i++ {
		r, err := RollFormula("3d4+3")
		if err != nil {
			t.Fatal(err)
		}
		if r.Total < 6 || r.Total > 15 {
			t.Fatalf("3d4+3 rolled %d parts=%v", r.Total, r.Parts)
		}
	}
}

func TestFireballFormulaRoll(t *testing.T) {
	r, err := RollFormula("8d6")
	if err != nil {
		t.Fatal(err)
	}
	if r.Total < 8 || r.Total > 48 {
		t.Fatalf("8d6=%d", r.Total)
	}
}

func TestBadFormula(t *testing.T) {
	if _, err := RollFormula(""); err != ErrBadFormula {
		t.Fatalf("empty: %v", err)
	}
	if _, err := RollFormula("fire"); err != ErrBadFormula {
		t.Fatalf("text: %v", err)
	}
}
