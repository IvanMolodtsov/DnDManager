package battles

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"dndmanager/internal/campaigns"
	"dndmanager/internal/catalog"
	"dndmanager/internal/characters"
	"dndmanager/internal/platform"
	"dndmanager/internal/rules"
)

// Controller serves /campaigns/{id}/battle and HTMX islands.
type Controller struct {
	Svc        *Service
	Campaigns  *campaigns.Service
	Characters *characters.Service
	Catalog    *catalog.Service
	Render     *platform.Renderer
}

func (c *Controller) Mount(mux *http.ServeMux, auth func(http.Handler) http.Handler) {
	mux.Handle("GET /campaigns/{id}/battle", auth(http.HandlerFunc(c.show)))
	mux.Handle("POST /campaigns/{id}/battle", auth(http.HandlerFunc(c.start)))
	mux.Handle("GET /campaigns/{id}/battle/board", auth(http.HandlerFunc(c.board)))
	mux.Handle("GET /campaigns/{id}/battle/events", auth(http.HandlerFunc(c.events)))
	mux.Handle("GET /campaigns/{id}/battle/monsters/search", auth(http.HandlerFunc(c.searchMonsters)))
	mux.Handle("GET /campaigns/{id}/battle/monsters/{mid}/preview", auth(http.HandlerFunc(c.previewMonster)))
	mux.Handle("POST /campaigns/{id}/battle/monsters", auth(http.HandlerFunc(c.addMonsters)))
	mux.Handle("GET /campaigns/{id}/battle/monsters/{mid}/edit", auth(http.HandlerFunc(c.editMonsterStats)))
	mux.Handle("POST /campaigns/{id}/battle/monsters/{mid}/edit", auth(http.HandlerFunc(c.saveMonsterStats)))
	mux.Handle("POST /campaigns/{id}/battle/monsters/{mid}/remove", auth(http.HandlerFunc(c.removeMonsters)))
	mux.Handle("POST /campaigns/{id}/battle/initiative", auth(http.HandlerFunc(c.beginInitiative)))
	mux.Handle("POST /campaigns/{id}/battle/units/{uid}/initiative", auth(http.HandlerFunc(c.setInitiative)))
	mux.Handle("POST /campaigns/{id}/battle/units/{uid}/roll", auth(http.HandlerFunc(c.rollInitiative)))
	mux.Handle("POST /campaigns/{id}/battle/confirm", auth(http.HandlerFunc(c.confirm)))
	mux.Handle("POST /campaigns/{id}/battle/next", auth(http.HandlerFunc(c.next)))
	mux.Handle("POST /campaigns/{id}/battle/end", auth(http.HandlerFunc(c.end)))
	mux.Handle("POST /campaigns/{id}/battle/dismiss", auth(http.HandlerFunc(c.dismiss)))
	mux.Handle("POST /campaigns/{id}/battle/units/{uid}/escape", auth(http.HandlerFunc(c.escape)))
	mux.Handle("GET /campaigns/{id}/battle/units/{uid}/adjust", auth(http.HandlerFunc(c.hpAdjustModal)))
	mux.Handle("POST /campaigns/{id}/battle/units/{uid}/adjust", auth(http.HandlerFunc(c.applyHPAdjust)))
	mux.Handle("GET /campaigns/{id}/battle/units/{uid}/death-save", auth(http.HandlerFunc(c.deathSaveModal)))
	mux.Handle("POST /campaigns/{id}/battle/units/{uid}/death-save/roll", auth(http.HandlerFunc(c.rollDeathSave)))
	mux.Handle("POST /campaigns/{id}/battle/units/{uid}/death-save", auth(http.HandlerFunc(c.recordDeathSave)))
	mux.Handle("GET /campaigns/{id}/battle/units/{uid}/effects/new", auth(http.HandlerFunc(c.statusAddModal)))
	mux.Handle("GET /campaigns/{id}/battle/units/{uid}/effects/search", auth(http.HandlerFunc(c.searchConditions)))
	mux.Handle("POST /campaigns/{id}/battle/units/{uid}/effects", auth(http.HandlerFunc(c.addUnitStatus)))
	mux.Handle("POST /campaigns/{id}/battle/units/{uid}/effects/{eid}/remove", auth(http.HandlerFunc(c.removeUnitStatus)))
}

func (c *Controller) base(r *http.Request, title string) platform.BaseView {
	return platform.NewBase(r, title)
}

type boardView struct {
	platform.BaseView
	Campaign         *campaigns.Campaign
	Battle           *Battle
	IsDM             bool
	Groups           []MonsterGroup
	Query            string
	CR               string
	Results          []catalog.Monster
	CRs              []string
	Preview          *catalog.Monster
	Stats            Stats
	HPResult         *characters.HPResult
	OOBBattle        bool
	AdjustPool       string
	AdjustSign       string
	AdjustAmount     int
	AdjustDamageType string
	AdjustUnit       *Unit
	DamageTypes      []rules.NamedOption
	DeathUnit        *Unit
	DeathRoll        int
	RollResult       *rules.RollResult
	EditGroup        *MonsterGroup
	AbilityKeys      []string
	StatusUnit       *Unit
	StatusQuery      string
	StatusResults    []catalog.Condition
	StatusPick       *catalog.Condition
	DefaultRemoveEnd bool
	StatusNewURL     string
	StatusSearchURL  string
	StatusPostURL    string
}

func (c *Controller) loadView(w http.ResponseWriter, r *http.Request) (*boardView, bool) {
	id, err := platform.PathID(r, "id")
	if err != nil {
		http.NotFound(w, r)
		return nil, false
	}
	u := platform.UserFrom(r.Context())
	camp, err := c.Campaigns.Get(id)
	if err != nil {
		http.NotFound(w, r)
		return nil, false
	}
	if _, err := c.Campaigns.RequireMember(id, u.ID); err != nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return nil, false
	}
	b, err := c.Svc.GetForCampaign(id, u.ID)
	if err != nil && !errors.Is(err, ErrNotFound) {
		http.Error(w, "error", http.StatusInternalServerError)
		return nil, false
	}
	lang := platform.LangFrom(r.Context())
	v := &boardView{
		BaseView:    c.base(r, "battle.title"),
		Campaign:    camp,
		Battle:      b,
		IsDM:        c.Campaigns.IsDM(id, u.ID),
		DamageTypes: rules.DamageTypes(),
		AbilityKeys: rules.AbilityKeys,
	}
	if b != nil {
		v.Groups = c.Svc.Groups(b, lang)
		v.Stats = b.ComputeStats()
	}
	crs, _ := c.Catalog.ListMonsterCRs()
	v.CRs = crs
	return v, true
}

func (c *Controller) show(w http.ResponseWriter, r *http.Request) {
	v, ok := c.loadView(w, r)
	if !ok {
		return
	}
	c.Render.Render(w, "battles/show.html", v.Lang, http.StatusOK, v)
}

func (c *Controller) board(w http.ResponseWriter, r *http.Request) {
	v, ok := c.loadView(w, r)
	if !ok {
		return
	}
	c.Render.Render(w, "battles/board.html", v.Lang, http.StatusOK, v)
}

func (c *Controller) start(w http.ResponseWriter, r *http.Request) {
	id, err := platform.PathID(r, "id")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	u := platform.UserFrom(r.Context())
	if _, err := c.Svc.Start(id, u.ID); err != nil {
		http.Error(w, ErrorKey(err), statusFor(err))
		return
	}
	platform.Redirect(w, r, "/campaigns/"+strconv.FormatInt(id, 10)+"/battle")
}

func (c *Controller) renderBoard(w http.ResponseWriter, r *http.Request, b *Battle, status int) {
	v, ok := c.loadView(w, r)
	if !ok {
		return
	}
	if b != nil {
		v.Battle = b
		v.Groups = c.Svc.Groups(b, v.Lang)
		v.Stats = b.ComputeStats()
	}
	c.Render.Render(w, "battles/board.html", v.Lang, status, v)
}

func (c *Controller) mutateBoard(w http.ResponseWriter, r *http.Request, fn func(campaignID, userID int64) (*Battle, error)) {
	id, err := platform.PathID(r, "id")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	u := platform.UserFrom(r.Context())
	b, err := fn(id, u.ID)
	if err != nil {
		v, ok := c.loadView(w, r)
		if !ok {
			return
		}
		v.Error = ErrorKey(err)
		c.Render.Render(w, "battles/board.html", v.Lang, statusFor(err), v)
		return
	}
	c.renderBoard(w, r, b, http.StatusOK)
}

func (c *Controller) next(w http.ResponseWriter, r *http.Request) {
	c.mutateBoard(w, r, c.Svc.Next)
}

func (c *Controller) end(w http.ResponseWriter, r *http.Request) {
	c.mutateBoard(w, r, c.Svc.End)
}

func (c *Controller) beginInitiative(w http.ResponseWriter, r *http.Request) {
	c.mutateBoard(w, r, c.Svc.BeginInitiative)
}

func (c *Controller) confirm(w http.ResponseWriter, r *http.Request) {
	c.mutateBoard(w, r, c.Svc.ConfirmInitiative)
}

func (c *Controller) dismiss(w http.ResponseWriter, r *http.Request) {
	id, err := platform.PathID(r, "id")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	u := platform.UserFrom(r.Context())
	if err := c.Svc.Dismiss(id, u.ID); err != nil {
		http.Error(w, ErrorKey(err), statusFor(err))
		return
	}
	platform.Redirect(w, r, "/campaigns/"+strconv.FormatInt(id, 10))
}

func (c *Controller) escape(w http.ResponseWriter, r *http.Request) {
	uid, err := platform.PathID(r, "uid")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	c.mutateBoard(w, r, func(campaignID, userID int64) (*Battle, error) {
		return c.Svc.MarkEscaped(campaignID, userID, uid)
	})
}

func (c *Controller) setInitiative(w http.ResponseWriter, r *http.Request) {
	uid, err := platform.PathID(r, "uid")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	value := platform.FormInt(r, "initiative")
	c.mutateBoard(w, r, func(campaignID, userID int64) (*Battle, error) {
		return c.Svc.SetInitiative(campaignID, userID, uid, value)
	})
}

func (c *Controller) rollInitiative(w http.ResponseWriter, r *http.Request) {
	uid, err := platform.PathID(r, "uid")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	c.mutateBoard(w, r, func(campaignID, userID int64) (*Battle, error) {
		_, b, err := c.Svc.RollInitiative(campaignID, userID, uid)
		return b, err
	})
}

func (c *Controller) addMonsters(w http.ResponseWriter, r *http.Request) {
	mid := platform.FormInt64(r, "monster_id")
	qty := platform.FormInt(r, "qty")
	if qty < 1 {
		qty = 1
	}
	c.renderRosterAfter(w, r, func(campaignID, userID int64) (*Battle, error) {
		lang := platform.LangFrom(r.Context())
		return c.Svc.AddMonsters(campaignID, userID, mid, qty, lang)
	})
}

func (c *Controller) removeMonsters(w http.ResponseWriter, r *http.Request) {
	mid, err := platform.PathID(r, "mid")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	c.renderRosterAfter(w, r, func(campaignID, userID int64) (*Battle, error) {
		return c.Svc.RemoveMonsterType(campaignID, userID, mid)
	})
}

func (c *Controller) groupByCatalog(v *boardView, catalogID int64) *MonsterGroup {
	for i := range v.Groups {
		if v.Groups[i].CatalogID == catalogID {
			return &v.Groups[i]
		}
	}
	return nil
}

func (c *Controller) editMonsterStats(w http.ResponseWriter, r *http.Request) {
	v, ok := c.loadView(w, r)
	if !ok {
		return
	}
	if !v.IsDM {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	mid, err := platform.PathID(r, "mid")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	g := c.groupByCatalog(v, mid)
	if g == nil {
		http.NotFound(w, r)
		return
	}
	v.EditGroup = g
	c.Render.Render(w, "battles/monster_stats.html", v.Lang, http.StatusOK, v)
}

func (c *Controller) saveMonsterStats(w http.ResponseWriter, r *http.Request) {
	mid, err := platform.PathID(r, "mid")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	st := groupStatsFromForm(r)
	id, err := platform.PathID(r, "id")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	u := platform.UserFrom(r.Context())
	b, err := c.Svc.UpdateMonsterGroupStats(id, u.ID, mid, st)
	v, ok := c.loadView(w, r)
	if !ok {
		return
	}
	if err != nil {
		g := c.groupByCatalog(v, mid)
		if g == nil {
			g = &MonsterGroup{CatalogID: mid}
		}
		g.ApplyStats(st)
		v.EditGroup = g
		v.Error = ErrorKey(err)
		w.Header().Set("HX-Retarget", "#monster-stats-body")
		w.Header().Set("HX-Reswap", "innerHTML")
		c.Render.Render(w, "battles/monster_stats.html", v.Lang, statusFor(err), v)
		return
	}
	v.Battle = b
	v.Groups = c.Svc.Groups(b, v.Lang)
	c.Render.Render(w, "battles/monster_roster.html", v.Lang, http.StatusOK, v)
}

func groupStatsFromForm(r *http.Request) GroupStats {
	st := GroupStats{
		HPCurrent: platform.FormInt(r, "hp_current"),
		HPMax:     platform.FormInt(r, "hp_max"),
		AC:        platform.FormInt(r, "ac"),
	}
	st.Scores.STR = platform.FormInt(r, "str")
	st.Scores.DEX = platform.FormInt(r, "dex")
	st.Scores.CON = platform.FormInt(r, "con")
	st.Scores.INT = platform.FormInt(r, "int")
	st.Scores.WIS = platform.FormInt(r, "wis")
	st.Scores.CHA = platform.FormInt(r, "cha")
	st.Resist.Resistances = filterDamageTypes(platform.FormList(r, "resist"))
	st.Resist.Immunities = filterDamageTypes(platform.FormList(r, "immune"))
	st.Resist.Vulnerabilities = filterDamageTypes(platform.FormList(r, "vuln"))
	return st
}

func (c *Controller) renderRosterAfter(w http.ResponseWriter, r *http.Request, fn func(campaignID, userID int64) (*Battle, error)) {
	id, err := platform.PathID(r, "id")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	u := platform.UserFrom(r.Context())
	b, err := fn(id, u.ID)
	v, ok := c.loadView(w, r)
	if !ok {
		return
	}
	if err != nil {
		v.Error = ErrorKey(err)
		w.Header().Set("HX-Retarget", "#monster-roster")
		w.Header().Set("HX-Reswap", "outerHTML")
		c.Render.Render(w, "battles/monster_roster.html", v.Lang, statusFor(err), v)
		return
	}
	v.Battle = b
	v.Groups = c.Svc.Groups(b, v.Lang)
	c.Render.Render(w, "battles/monster_roster.html", v.Lang, http.StatusOK, v)
}

func (c *Controller) searchMonsters(w http.ResponseWriter, r *http.Request) {
	v, ok := c.loadView(w, r)
	if !ok {
		return
	}
	if !v.IsDM {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	v.Query = r.URL.Query().Get("q")
	v.CR = r.URL.Query().Get("cr")
	res, err := c.Catalog.SearchMonsters(v.Query, v.CR, 20)
	if err != nil {
		http.Error(w, "error", http.StatusInternalServerError)
		return
	}
	v.Results = res
	c.Render.Render(w, "battles/monster_results.html", v.Lang, http.StatusOK, v)
}

func (c *Controller) previewMonster(w http.ResponseWriter, r *http.Request) {
	v, ok := c.loadView(w, r)
	if !ok {
		return
	}
	if !v.IsDM {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	mid, err := platform.PathID(r, "mid")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	m, err := c.Catalog.Monster(mid)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	v.Preview = m
	c.Render.Render(w, "battles/monster_preview.html", v.Lang, http.StatusOK, v)
}

func (c *Controller) unitFrom(r *http.Request, v *boardView) *Unit {
	uid, err := platform.PathID(r, "uid")
	if err != nil || v.Battle == nil {
		return nil
	}
	for i := range v.Battle.Units {
		if v.Battle.Units[i].ID == uid {
			return &v.Battle.Units[i]
		}
	}
	return nil
}

func validAdjust(pool, sign string) bool {
	if pool != "hp" && pool != "temp" {
		return false
	}
	return sign == "+" || sign == "-"
}

func (c *Controller) hpAdjustModal(w http.ResponseWriter, r *http.Request) {
	v, ok := c.loadView(w, r)
	if !ok {
		return
	}
	if !v.IsDM {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	u := c.unitFrom(r, v)
	if u == nil {
		http.NotFound(w, r)
		return
	}
	pool := r.URL.Query().Get("pool")
	sign := r.URL.Query().Get("sign")
	if !validAdjust(pool, sign) {
		http.Error(w, "error.generic", http.StatusBadRequest)
		return
	}
	v.AdjustUnit = u
	v.AdjustPool = pool
	v.AdjustSign = sign
	v.AdjustAmount = 1
	v.AdjustDamageType = "slashing"
	c.Render.Render(w, "battles/hp_adjust.html", v.Lang, http.StatusOK, v)
}

func (c *Controller) applyHPAdjust(w http.ResponseWriter, r *http.Request) {
	v, ok := c.loadView(w, r)
	if !ok {
		return
	}
	if !v.IsDM {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	u := platform.UserFrom(r.Context())
	unit := c.unitFrom(r, v)
	if unit == nil {
		http.NotFound(w, r)
		return
	}
	pool := platform.FormTrim(r, "pool")
	sign := platform.FormTrim(r, "sign")
	amount := platform.FormInt(r, "amount")
	dmgType := platform.FormTrim(r, "damage_type")
	if !validAdjust(pool, sign) {
		c.hpAdjustErr(w, r, v, unit, pool, sign, amount, dmgType, rules.ErrAmount)
		return
	}
	lang := platform.LangFrom(r.Context())
	var note *characters.HPResult
	var b *Battle
	var err error
	if pool == "temp" {
		delta := amount
		if sign == "-" {
			delta = -amount
		}
		if amount < 1 {
			c.hpAdjustErr(w, r, v, unit, pool, sign, amount, dmgType, rules.ErrAmount)
			return
		}
		b, err = c.Svc.AdjustUnitTemp(v.Campaign.ID, u.ID, unit.ID, delta)
		kind := "temp_up"
		if sign == "-" {
			kind = "temp_down"
		}
		note = &characters.HPResult{Kind: kind, Amount: amount}
	} else if sign == "+" {
		b, err = c.Svc.ApplyUnitHeal(v.Campaign.ID, u.ID, unit.ID, amount)
		note = &characters.HPResult{Kind: "heal", Amount: amount}
	} else {
		var res rules.DamageResult
		res, b, err = c.Svc.ApplyUnitDamage(v.Campaign.ID, u.ID, unit.ID, amount, dmgType)
		note = &characters.HPResult{
			Kind: "damage", Incoming: res.Incoming, TypeName: rules.DamageTypeName(res.DamageType, lang),
			ModKey: res.NoteKey(), Applied: res.Applied, Absorbed: res.AbsorbedTemp,
		}
	}
	if err != nil {
		c.hpAdjustErr(w, r, v, unit, pool, sign, amount, dmgType, err)
		return
	}
	v.Battle = b
	v.Groups = c.Svc.Groups(b, lang)
	v.Stats = b.ComputeStats()
	v.AdjustUnit = unit
	v.AdjustPool = pool
	v.AdjustSign = sign
	v.AdjustAmount = amount
	v.AdjustDamageType = dmgType
	v.HPResult = note
	v.OOBBattle = true
	c.Render.Render(w, "battles/hp_adjust_after.html", v.Lang, http.StatusOK, v)
}

func (c *Controller) hpAdjustErr(w http.ResponseWriter, r *http.Request, v *boardView, unit *Unit, pool, sign string, amount int, dmgType string, err error) {
	v.AdjustUnit = unit
	v.AdjustPool = pool
	v.AdjustSign = sign
	v.AdjustAmount = amount
	v.AdjustDamageType = dmgType
	v.Error = ErrorKey(err)
	w.Header().Set("HX-Retarget", "#hp-adjust-body")
	w.Header().Set("HX-Reswap", "innerHTML")
	c.Render.Render(w, "battles/hp_adjust.html", v.Lang, http.StatusUnprocessableEntity, v)
}

func (c *Controller) deathSaveModal(w http.ResponseWriter, r *http.Request) {
	v, ok := c.loadView(w, r)
	if !ok {
		return
	}
	if !v.IsDM {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	u := c.unitFrom(r, v)
	if u == nil {
		http.NotFound(w, r)
		return
	}
	v.DeathUnit = u
	c.Render.Render(w, "battles/death_save.html", v.Lang, http.StatusOK, v)
}

func (c *Controller) rollDeathSave(w http.ResponseWriter, r *http.Request) {
	v, ok := c.loadView(w, r)
	if !ok {
		return
	}
	uid, err := platform.PathID(r, "uid")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	user := platform.UserFrom(r.Context())
	res, b, err := c.Svc.RollDeathSave(v.Campaign.ID, user.ID, uid)
	if err != nil {
		v.Error = ErrorKey(err)
		v.DeathUnit = c.unitFrom(r, v)
		c.Render.Render(w, "battles/death_save.html", v.Lang, statusFor(err), v)
		return
	}
	v.Battle = b
	v.DeathUnit = c.unitFrom(r, v)
	v.RollResult = &res
	v.DeathRoll = res.Total
	c.Render.Render(w, "battles/death_save.html", v.Lang, http.StatusOK, v)
}

func (c *Controller) recordDeathSave(w http.ResponseWriter, r *http.Request) {
	uid, err := platform.PathID(r, "uid")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	roll := platform.FormInt(r, "roll")
	c.mutateBoard(w, r, func(campaignID, userID int64) (*Battle, error) {
		return c.Svc.RecordDeathSave(campaignID, userID, uid, roll)
	})
}

func (c *Controller) events(w http.ResponseWriter, r *http.Request) {
	id, err := platform.PathID(r, "id")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	u := platform.UserFrom(r.Context())
	if _, err := c.Campaigns.RequireMember(id, u.ID); err != nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	if _, err := fmt.Fprintf(w, "retry: 2000\n\n"); err != nil {
		return
	}
	flusher.Flush()

	hub := c.Svc.Events
	if hub == nil {
		hub = NewHub()
		c.Svc.Events = hub
	}
	notify := hub.Subscribe(id)
	defer hub.Unsubscribe(id, notify)

	ping := time.NewTicker(20 * time.Second)
	defer ping.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-notify:
			if _, err := fmt.Fprintf(w, "event: battle\ndata: %d\n\n", id); err != nil {
				slog.Info("sse write", "err", err, "campaign", id)
				return
			}
			flusher.Flush()
		case <-ping.C:
			if _, err := fmt.Fprintf(w, ": ping\n\n"); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

func statusFor(err error) int {
	switch {
	case errors.Is(err, ErrForbidden), errors.Is(err, characters.ErrForbidden), errors.Is(err, ErrNotMember):
		return http.StatusForbidden
	case errors.Is(err, ErrNotFound), errors.Is(err, catalog.ErrNotFound):
		return http.StatusNotFound
	default:
		return http.StatusUnprocessableEntity
	}
}
