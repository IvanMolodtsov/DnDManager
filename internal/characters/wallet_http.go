package characters

import (
	"net/http"

	"dndmanager/internal/platform"
)

func (c *Controller) inventoryPartial(w http.ResponseWriter, r *http.Request) {
	ch, ok := c.loadLive(w, r)
	if !ok {
		return
	}
	u := platform.UserFrom(r.Context())
	readonly, _ := c.Svc.CanView(ch, u.ID)
	c.renderInventory(w, r, ch, readonly)
}

func (c *Controller) goldAdjustModal(w http.ResponseWriter, r *http.Request) {
	c.walletModal(w, r, "gold")
}

func (c *Controller) soulsAdjustModal(w http.ResponseWriter, r *http.Request) {
	c.walletModal(w, r, "souls")
}

func (c *Controller) soulsCapModal(w http.ResponseWriter, r *http.Request) {
	c.walletModal(w, r, "cap")
}

func (c *Controller) walletModal(w http.ResponseWriter, r *http.Request, pool string) {
	ch, ok := c.loadLive(w, r)
	if !ok {
		return
	}
	u := platform.UserFrom(r.Context())
	if pool == "gold" {
		if err := c.Svc.RequireCombatEdit(ch, u.ID); err != nil {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
	} else if err := c.Svc.RequireDM(ch, u.ID); err != nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	v := c.sheetView(r, ch, ch.OwnerID != u.ID)
	v.AdjustPool = pool
	v.AdjustSign = r.URL.Query().Get("sign")
	if v.AdjustSign != "-" {
		v.AdjustSign = "+"
	}
	c.Render.Render(w, "characters/wallet_adjust.html", v.Lang, http.StatusOK, v)
}

func (c *Controller) applyGold(w http.ResponseWriter, r *http.Request) {
	ch, ok := c.loadLive(w, r)
	if !ok {
		return
	}
	u := platform.UserFrom(r.Context())
	delta := walletDelta(r)
	if err := c.Svc.AdjustGold(ch, u.ID, delta); err != nil {
		c.walletErr(w, r, ch, "gold", err)
		return
	}
	ch, _ = c.Svc.Get(ch.ID)
	c.Svc.Localize(ch, platform.LangFrom(r.Context()))
	w.Header().Set("HX-Trigger", "close-wallet-modal")
	c.renderInventory(w, r, ch, ch.OwnerID != u.ID)
}

func (c *Controller) applySouls(w http.ResponseWriter, r *http.Request) {
	ch, ok := c.loadLive(w, r)
	if !ok {
		return
	}
	u := platform.UserFrom(r.Context())
	if platform.FormTrim(r, "pool") == "cap" || r.URL.Query().Get("pool") == "cap" {
		c.applySoulsCap(w, r, ch)
		return
	}
	delta := walletDelta(r)
	if err := c.Svc.AdjustSouls(ch, u.ID, delta); err != nil {
		c.walletErr(w, r, ch, "souls", err)
		return
	}
	ch, _ = c.Svc.Get(ch.ID)
	c.Svc.Localize(ch, platform.LangFrom(r.Context()))
	w.Header().Set("HX-Trigger", "close-wallet-modal")
	c.renderInventory(w, r, ch, ch.OwnerID != u.ID)
}

func (c *Controller) applySoulsCap(w http.ResponseWriter, r *http.Request, ch *Character) {
	u := platform.UserFrom(r.Context())
	cap := platform.FormInt(r, "amount")
	if err := c.Svc.SetSoulsCap(ch, u.ID, cap); err != nil {
		c.walletErr(w, r, ch, "cap", err)
		return
	}
	ch, _ = c.Svc.Get(ch.ID)
	c.Svc.Localize(ch, platform.LangFrom(r.Context()))
	w.Header().Set("HX-Trigger", "close-wallet-modal")
	c.renderInventory(w, r, ch, ch.OwnerID != u.ID)
}

func (c *Controller) walletErr(w http.ResponseWriter, r *http.Request, ch *Character, pool string, err error) {
	u := platform.UserFrom(r.Context())
	v := c.sheetView(r, ch, ch.OwnerID != u.ID)
	v.Error = ErrorKey(err)
	v.AdjustPool = pool
	v.AdjustSign = platform.FormTrim(r, "sign")
	if v.AdjustSign != "-" {
		v.AdjustSign = "+"
	}
	v.AdjustAmount = platform.FormInt(r, "amount")
	w.Header().Set("HX-Retarget", "#wallet-body")
	w.Header().Set("HX-Reswap", "innerHTML")
	c.Render.Render(w, "characters/wallet_adjust.html", v.Lang, http.StatusUnprocessableEntity, v)
}

func walletDelta(r *http.Request) int {
	amount := platform.FormInt(r, "amount")
	if amount < 1 {
		return 0
	}
	if platform.FormTrim(r, "sign") == "-" {
		return -amount
	}
	return amount
}
