package campaigns

import (
	"errors"
	"net/http"
	"strconv"

	"dndmanager/internal/platform"
)

func (c *Controller) soulsAdjustModal(w http.ResponseWriter, r *http.Request) {
	c.walletModal(w, r, "souls", 0)
}

func (c *Controller) soulsCapModal(w http.ResponseWriter, r *http.Request) {
	c.walletModal(w, r, "cap", 0)
}

func (c *Controller) goldAdjustModal(w http.ResponseWriter, r *http.Request) {
	cid, err := platform.PathID(r, "cid")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	c.walletModal(w, r, "gold", cid)
}

func (c *Controller) walletModal(w http.ResponseWriter, r *http.Request, pool string, characterID int64) {
	v, ok := c.loadShow(w, r)
	if !ok {
		return
	}
	if !v.IsDM {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if pool == "gold" {
		row, found := findCharacter(v.Characters, characterID)
		if !found {
			http.NotFound(w, r)
			return
		}
		v.GoldCharacter = &row
	}
	v.AdjustPool = pool
	v.AdjustSign = r.URL.Query().Get("sign")
	if v.AdjustSign != "-" {
		v.AdjustSign = "+"
	}
	c.Render.Render(w, "campaigns/wallet_adjust.html", v.Lang, http.StatusOK, v)
}

func (c *Controller) applySouls(w http.ResponseWriter, r *http.Request) {
	v, ok := c.loadShow(w, r)
	if !ok {
		return
	}
	u := platform.UserFrom(r.Context())
	pool := "souls"
	var err error
	if platform.FormTrim(r, "pool") == "cap" || r.URL.Query().Get("pool") == "cap" {
		pool = "cap"
		err = c.Svc.SetSoulsCap(v.Campaign.ID, u.ID, platform.FormInt(r, "amount"))
	} else {
		err = c.Svc.AdjustSouls(v.Campaign.ID, u.ID, walletDelta(r))
	}
	if err != nil {
		c.walletErr(w, r, v, pool, 0, err)
		return
	}
	if c.Characters != nil {
		c.Characters.BroadcastWallet(v.Campaign.ID)
	}
	c.renderTreasury(w, r)
}

func (c *Controller) applyGold(w http.ResponseWriter, r *http.Request) {
	v, ok := c.loadShow(w, r)
	if !ok {
		return
	}
	if !v.IsDM || c.Characters == nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	cid, err := platform.PathID(r, "cid")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	u := platform.UserFrom(r.Context())
	if err := c.Characters.AdjustCampaignGold(v.Campaign.ID, cid, u.ID, walletDelta(r)); err != nil {
		c.walletErr(w, r, v, "gold", cid, err)
		return
	}
	c.renderTreasury(w, r)
}

func (c *Controller) applyRevive(w http.ResponseWriter, r *http.Request) {
	v, ok := c.loadShow(w, r)
	if !ok {
		return
	}
	if !v.IsDM || c.Characters == nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	cid, err := platform.PathID(r, "cid")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	u := platform.UserFrom(r.Context())
	if err := c.Characters.Revive(v.Campaign.ID, cid, u.ID); err != nil {
		if errors.Is(err, ErrForbidden) || errors.Is(err, ErrNotMember) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		v.Error = ErrorKey(err)
		w.Header().Set("HX-Retarget", "#campaign-characters")
		w.Header().Set("HX-Reswap", "outerHTML")
		c.Render.Render(w, "campaigns/characters.html", v.Lang, http.StatusUnprocessableEntity, v)
		return
	}
	platform.Redirect(w, r, "/campaigns/"+strconv.FormatInt(v.Campaign.ID, 10))
}

func (c *Controller) renderTreasury(w http.ResponseWriter, r *http.Request) {
	v, ok := c.loadShow(w, r)
	if !ok {
		return
	}
	w.Header().Set("HX-Trigger", "close-wallet-modal")
	c.Render.Render(w, "campaigns/treasury.html", v.Lang, http.StatusOK, v)
}

func (c *Controller) walletErr(w http.ResponseWriter, r *http.Request, v showView, pool string, characterID int64, err error) {
	if errors.Is(err, ErrForbidden) || errors.Is(err, ErrNotMember) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if errors.Is(err, ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if platform.FormTrim(r, "pool") == "cap" || pool == "cap" {
		v.AdjustPool = "cap"
	} else {
		v.AdjustPool = pool
	}
	if pool == "gold" {
		if row, found := findCharacter(v.Characters, characterID); found {
			v.GoldCharacter = &row
		}
	}
	v.Error = ErrorKey(err)
	v.AdjustSign = platform.FormTrim(r, "sign")
	if v.AdjustSign != "-" {
		v.AdjustSign = "+"
	}
	v.AdjustAmount = platform.FormInt(r, "amount")
	w.Header().Set("HX-Retarget", "#wallet-body")
	w.Header().Set("HX-Reswap", "innerHTML")
	c.Render.Render(w, "campaigns/wallet_adjust.html", v.Lang, http.StatusUnprocessableEntity, v)
}

func findCharacter(chars []CharacterSummary, id int64) (CharacterSummary, bool) {
	for _, ch := range chars {
		if ch.ID == id {
			return ch, true
		}
	}
	return CharacterSummary{}, false
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
