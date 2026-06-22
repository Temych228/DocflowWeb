package service

import (
	"sync"

	"github.com/Temych228/DocflowWeb/services/notification-service/internal/domain"
)

type Hub struct {
	mu   sync.RWMutex
	subs map[string]map[chan *domain.Notification]struct{}
}

func NewHub() *Hub {
	return &Hub{subs: make(map[string]map[chan *domain.Notification]struct{})}
}

func (h *Hub) Subscribe(userID string) (<-chan *domain.Notification, func()) {
	ch := make(chan *domain.Notification, 32)

	h.mu.Lock()
	if h.subs[userID] == nil {
		h.subs[userID] = make(map[chan *domain.Notification]struct{})
	}
	h.subs[userID][ch] = struct{}{}
	h.mu.Unlock()

	unsubscribe := func() {
		h.mu.Lock()
		if m, ok := h.subs[userID]; ok {
			if _, ok := m[ch]; ok {
				delete(m, ch)
				close(ch)
			}
			if len(m) == 0 {
				delete(h.subs, userID)
			}
		}
		h.mu.Unlock()
	}

	return ch, unsubscribe
}

func (h *Hub) Publish(n *domain.Notification) {
	if n == nil {
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	if m, ok := h.subs[n.UserID]; ok {
		for ch := range m {
			select {
			case ch <- n:
			default:
			}
		}
	}
}
