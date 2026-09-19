package battles

import (
	"sync"
)

// Hub fans out in-process battle-board notifications keyed by campaign ID.
type Hub struct {
	mu   sync.RWMutex
	subs map[int64]map[chan struct{}]struct{}
}

func NewHub() *Hub {
	return &Hub{subs: map[int64]map[chan struct{}]struct{}{}}
}

func (h *Hub) Subscribe(campaignID int64) chan struct{} {
	ch := make(chan struct{}, 1)
	if h == nil {
		return ch
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.subs == nil {
		h.subs = map[int64]map[chan struct{}]struct{}{}
	}
	if h.subs[campaignID] == nil {
		h.subs[campaignID] = map[chan struct{}]struct{}{}
	}
	h.subs[campaignID][ch] = struct{}{}
	return ch
}

func (h *Hub) Unsubscribe(campaignID int64, ch chan struct{}) {
	if h == nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	set := h.subs[campaignID]
	if set == nil {
		return
	}
	delete(set, ch)
	if len(set) == 0 {
		delete(h.subs, campaignID)
	}
}

func (h *Hub) Broadcast(campaignID int64) {
	if h == nil {
		return
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for ch := range h.subs[campaignID] {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}

func (s *Service) broadcast(campaignID int64) {
	if s == nil || s.Events == nil {
		return
	}
	s.Events.Broadcast(campaignID)
}
