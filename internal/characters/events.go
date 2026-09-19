package characters

import (
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"dndmanager/internal/platform"
)

// VitalsHub fans out in-process sheet-vitals notifications keyed by character ID.
type VitalsHub struct {
	mu   sync.RWMutex
	subs map[int64]map[chan struct{}]struct{}
}

func NewVitalsHub() *VitalsHub {
	return &VitalsHub{subs: map[int64]map[chan struct{}]struct{}{}}
}

func (h *VitalsHub) Subscribe(charID int64) chan struct{} {
	ch := make(chan struct{}, 1)
	if h == nil {
		return ch
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.subs == nil {
		h.subs = map[int64]map[chan struct{}]struct{}{}
	}
	if h.subs[charID] == nil {
		h.subs[charID] = map[chan struct{}]struct{}{}
	}
	h.subs[charID][ch] = struct{}{}
	return ch
}

func (h *VitalsHub) Unsubscribe(charID int64, ch chan struct{}) {
	if h == nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	set := h.subs[charID]
	if set == nil {
		return
	}
	delete(set, ch)
	if len(set) == 0 {
		delete(h.subs, charID)
	}
}

func (h *VitalsHub) Broadcast(charID int64) {
	if h == nil {
		return
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for ch := range h.subs[charID] {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}

func (s *Service) broadcastVitals(charID int64) {
	if s == nil || s.Events == nil {
		return
	}
	s.Events.Broadcast(charID)
}

func (c *Controller) vitalsPartial(w http.ResponseWriter, r *http.Request) {
	ch, ok := c.loadLive(w, r)
	if !ok {
		return
	}
	u := platform.UserFrom(r.Context())
	readonly, _ := c.Svc.CanView(ch, u.ID)
	c.renderCombat(w, r, ch, readonly)
}

func (c *Controller) vitalsEvents(w http.ResponseWriter, r *http.Request) {
	ch, ok := c.loadLive(w, r)
	if !ok {
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
		hub = NewVitalsHub()
		c.Svc.Events = hub
	}
	notify := hub.Subscribe(ch.ID)
	defer hub.Unsubscribe(ch.ID, notify)

	ping := time.NewTicker(20 * time.Second)
	defer ping.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-notify:
			if _, err := fmt.Fprintf(w, "event: vitals\ndata: %d\n\n", ch.ID); err != nil {
				slog.Info("sse write", "err", err, "character", ch.ID)
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
