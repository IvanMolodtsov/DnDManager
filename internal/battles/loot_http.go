package battles

import (
	"net/http"

	"dndmanager/internal/catalog"
	"dndmanager/internal/platform"
)

func (c *Controller) generateLoot(w http.ResponseWriter, r *http.Request) {
	c.mutateBoard(w, r, c.Svc.GenerateLoot)
}

func (c *Controller) addLootItem(w http.ResponseWriter, r *http.Request) {
	id, err := platform.PathID(r, "id")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	itemID := platform.FormInt64(r, "item_id")
	if itemID == 0 {
		itemID, _ = platform.PathID(r, "iid")
	}
	u := platform.UserFrom(r.Context())
	if _, err := c.Svc.AddLootItem(id, u.ID, itemID); err != nil {
		c.lootErr(w, r, err)
		return
	}
	w.Header().Set("HX-Trigger", "close-loot-modal")
	c.renderBoard(w, r, nil, http.StatusOK)
}

func (c *Controller) addQuestItem(w http.ResponseWriter, r *http.Request) {
	id, err := platform.PathID(r, "id")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	u := platform.UserFrom(r.Context())
	if _, err := c.Svc.AddQuestItem(id, u.ID, platform.FormTrim(r, "name")); err != nil {
		c.lootQuestErr(w, r, err)
		return
	}
	w.Header().Set("HX-Trigger", "close-loot-modal")
	c.renderBoard(w, r, nil, http.StatusOK)
}

func (c *Controller) removeLoot(w http.ResponseWriter, r *http.Request) {
	id, err := platform.PathID(r, "id")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	lootID, err := platform.PathID(r, "lid")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	u := platform.UserFrom(r.Context())
	b, err := c.Svc.RemoveLoot(id, u.ID, lootID)
	if err != nil {
		c.mutateBoard(w, r, func(campaignID, userID int64) (*Battle, error) { return nil, err })
		return
	}
	c.renderBoard(w, r, b, http.StatusOK)
}

func (c *Controller) lootAddModal(w http.ResponseWriter, r *http.Request) {
	v, ok := c.dmLootView(w, r)
	if !ok {
		return
	}
	c.Render.Render(w, "battles/loot_add.html", v.Lang, http.StatusOK, v)
}

func (c *Controller) lootSearch(w http.ResponseWriter, r *http.Request) {
	v, ok := c.dmLootView(w, r)
	if !ok {
		return
	}
	q := r.URL.Query().Get("q")
	if q == "" {
		q = platform.FormTrim(r, "q")
	}
	v.LootQuery = q
	if len(q) >= 1 {
		hits, _ := c.Catalog.SearchItems(q, 20)
		v.LootResults = filterLootSearch(hits)
	}
	c.Render.Render(w, "battles/loot_search_results.html", v.Lang, http.StatusOK, v)
}

func (c *Controller) lootPreview(w http.ResponseWriter, r *http.Request) {
	v, ok := c.dmLootView(w, r)
	if !ok {
		return
	}
	iid, err := platform.PathID(r, "iid")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	item, err := c.Catalog.Item(iid)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	v.LootItem = item
	c.Render.Render(w, "battles/loot_preview.html", v.Lang, http.StatusOK, v)
}

func (c *Controller) lootQuestModal(w http.ResponseWriter, r *http.Request) {
	v, ok := c.dmLootView(w, r)
	if !ok {
		return
	}
	c.Render.Render(w, "battles/loot_quest.html", v.Lang, http.StatusOK, v)
}

func (c *Controller) dmLootView(w http.ResponseWriter, r *http.Request) (*boardView, bool) {
	v, ok := c.loadView(w, r)
	if !ok {
		return nil, false
	}
	if !v.IsDM {
		http.Error(w, "forbidden", http.StatusForbidden)
		return nil, false
	}
	if v.Battle == nil || !v.Battle.Ended() {
		http.Error(w, "error", http.StatusUnprocessableEntity)
		return nil, false
	}
	return v, true
}

func (c *Controller) lootErr(w http.ResponseWriter, r *http.Request, err error) {
	v, ok := c.loadView(w, r)
	if !ok {
		return
	}
	v.Error = ErrorKey(err)
	w.Header().Set("HX-Retarget", "#loot-add-body")
	w.Header().Set("HX-Reswap", "innerHTML")
	c.Render.Render(w, "battles/loot_add.html", v.Lang, http.StatusUnprocessableEntity, v)
}

func (c *Controller) lootQuestErr(w http.ResponseWriter, r *http.Request, err error) {
	v, ok := c.loadView(w, r)
	if !ok {
		return
	}
	v.Error = ErrorKey(err)
	w.Header().Set("HX-Retarget", "#loot-add-body")
	w.Header().Set("HX-Reswap", "innerHTML")
	c.Render.Render(w, "battles/loot_quest.html", v.Lang, http.StatusUnprocessableEntity, v)
}

func filterLootSearch(items []catalog.Item) []catalog.Item {
	var out []catalog.Item
	for _, it := range items {
		if it.IsJewelrySlotBase() {
			continue
		}
		out = append(out, it)
	}
	return out
}
