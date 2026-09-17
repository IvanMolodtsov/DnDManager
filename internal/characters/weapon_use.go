package characters

import (
	"net/http"
	"strings"

	"dndmanager/internal/platform"
	"dndmanager/internal/rules"
)

type weaponUseView struct {
	platform.BaseView
	Character  *Character
	Item       InventoryItem
	Attack     rules.WeaponAttack
	Thrown     bool
	ReadOnly   bool
	HitRoll    *rules.RollResult
	DamageRoll *rules.RollResult
	ExtraRolls []rules.RollResult
}

func attackThrown(r *http.Request) bool {
	v := strings.ToLower(platform.FormTrim(r, "thrown"))
	return v == "1" || v == "true" || v == "on"
}

func (c *Controller) loadEquippedWeapon(w http.ResponseWriter, r *http.Request, ch *Character) (InventoryItem, bool) {
	invID, err := platform.PathID(r, "invID")
	if err != nil {
		http.NotFound(w, r)
		return InventoryItem{}, false
	}
	it, ok := ch.InventoryByID(invID)
	if !ok || !it.CanAttack() {
		http.NotFound(w, r)
		return InventoryItem{}, false
	}
	return it, true
}

func (c *Controller) weaponAttackModal(w http.ResponseWriter, r *http.Request) {
	ch, ok := c.loadLive(w, r)
	if !ok {
		return
	}
	it, ok := c.loadEquippedWeapon(w, r, ch)
	if !ok {
		return
	}
	readonly, _ := c.Svc.CanView(ch, platform.UserFrom(r.Context()).ID)
	c.renderWeaponUse(w, r, ch, it, readonly, attackThrown(r), nil, nil, nil)
}

func (c *Controller) rollWeaponAttack(w http.ResponseWriter, r *http.Request) {
	ch, ok := c.ownerSheet(w, r)
	if !ok {
		return
	}
	it, ok := c.loadEquippedWeapon(w, r, ch)
	if !ok {
		return
	}
	thrown := attackThrown(r)
	v := c.weaponUseData(r, ch, it, false, thrown)
	if v.Attack.HitFormula == "" {
		v.Error = "error.weapon.formula"
		c.Render.Render(w, "characters/weapon_use.html", v.Lang, http.StatusUnprocessableEntity, v)
		return
	}
	hit, err := rules.RollFormula(v.Attack.HitFormula)
	if err != nil {
		v.Error = ErrorKey(err)
		c.Render.Render(w, "characters/weapon_use.html", v.Lang, http.StatusUnprocessableEntity, v)
		return
	}
	v.HitRoll = &hit
	if v.Attack.DamageFormula != "" {
		dmg, err := rules.RollFormula(v.Attack.DamageFormula)
		if err != nil {
			v.Error = ErrorKey(err)
			c.Render.Render(w, "characters/weapon_use.html", v.Lang, http.StatusUnprocessableEntity, v)
			return
		}
		v.DamageRoll = &dmg
	}
	for _, extra := range v.Attack.Extra {
		formula := rules.ExtraRollFormula(extra)
		if formula == "" {
			continue
		}
		roll, err := rules.RollFormula(formula)
		if err != nil {
			v.Error = ErrorKey(err)
			c.Render.Render(w, "characters/weapon_use.html", v.Lang, http.StatusUnprocessableEntity, v)
			return
		}
		v.ExtraRolls = append(v.ExtraRolls, roll)
	}
	c.Render.Render(w, "characters/weapon_use.html", v.Lang, http.StatusOK, v)
}

func (c *Controller) renderWeaponUse(w http.ResponseWriter, r *http.Request, ch *Character, it InventoryItem, readonly bool, thrown bool, hit, dmg *rules.RollResult, extras []rules.RollResult) {
	v := c.weaponUseData(r, ch, it, readonly, thrown)
	v.HitRoll, v.DamageRoll, v.ExtraRolls = hit, dmg, extras
	isOwner := ch.OwnerID == platform.UserFrom(r.Context()).ID
	v.ReadOnly = readonly || !isOwner
	c.Render.Render(w, "characters/weapon_use.html", v.Lang, http.StatusOK, v)
}

func (c *Controller) weaponUseData(r *http.Request, ch *Character, it InventoryItem, readonly bool, thrown bool) weaponUseView {
	atk, _ := ch.WeaponAttack(it, thrown)
	return weaponUseView{
		BaseView:  c.base(r, ch.Name),
		Character: ch,
		Item:      it,
		Attack:    atk,
		Thrown:    atk.Throwing,
		ReadOnly:  readonly,
	}
}
