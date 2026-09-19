package rules

import "testing"

func TestApplyDamageNoTemp(t *testing.T) {
	res := ApplyDamage(15, 15, nil, ItemGrants{}, 10, "slashing")
	if res.Applied != 10 || res.NewHP != 5 || res.AbsorbedTemp != 0 {
		t.Fatalf("plain 10 vs 15 HP: %+v", res)
	}
}

func TestApplyDamageResistance(t *testing.T) {
	g := ItemGrants{Resistances: []string{"slashing"}}
	res := ApplyDamage(15, 15, nil, g, 10, "slashing")
	if res.Applied != 5 || res.NewHP != 10 || !res.Resistant {
		t.Fatalf("resist 10: %+v", res)
	}
	odd := ApplyDamage(15, 15, nil, g, 11, "slashing")
	if odd.Applied != 5 {
		t.Fatalf("resist 11 should round down to 5, got %d", odd.Applied)
	}
}

func TestApplyDamageImmunity(t *testing.T) {
	g := ItemGrants{Immunities: []string{"fire"}}
	res := ApplyDamage(15, 15, nil, g, 10, "fire")
	if res.Applied != 0 || res.NewHP != 15 || !res.Immune {
		t.Fatalf("immune: %+v", res)
	}
}

func TestApplyDamageVulnerability(t *testing.T) {
	g := ItemGrants{Vulnerabilities: []string{"cold"}}
	res := ApplyDamage(30, 30, nil, g, 10, "cold")
	if res.Applied != 20 || res.NewHP != 10 || !res.Vulnerable {
		t.Fatalf("vuln: %+v", res)
	}
}

func TestApplyDamageResistAndVulnCancel(t *testing.T) {
	g := ItemGrants{Resistances: []string{"fire"}, Vulnerabilities: []string{"fire"}}
	res := ApplyDamage(20, 20, nil, g, 10, "fire")
	if res.Applied != 10 || res.NewHP != 10 || res.NoteKey() != "" {
		t.Fatalf("resist+vuln should net ×1: %+v key=%q", res, res.NoteKey())
	}
}

func TestApplyDamageTempAbsorbsFirst(t *testing.T) {
	effects := []Effect{OtherTempEffect(5)}
	res := ApplyDamage(8, 20, effects, ItemGrants{}, 10, "slashing")
	if res.Applied != 10 || res.AbsorbedTemp != 5 || res.NewHP != 3 {
		t.Fatalf("8 HP + 5 temp vs 10: %+v", res)
	}
	if SumTempHP(res.NewEffects) != 0 {
		t.Fatalf("temp leftover %+v", res.NewEffects)
	}
}

func TestApplyDamageImmuneBeatsVuln(t *testing.T) {
	g := ItemGrants{Immunities: []string{"poison"}, Vulnerabilities: []string{"poison"}}
	res := ApplyDamage(12, 12, nil, g, 8, "poison")
	if res.Applied != 0 || res.NewHP != 12 || !res.Immune {
		t.Fatalf("immune wins: %+v", res)
	}
}
