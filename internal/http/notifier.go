package http

import "sync"

type liveEvent string

const (
	liveChanged liveEvent = "changed"
	liveStarted liveEvent = "started"
)

type liveNotifier struct {
	mu       sync.Mutex
	nextID   uint64
	sessions map[string]map[uint64]chan liveEvent
}

func newLiveNotifier() *liveNotifier {
	return &liveNotifier{sessions: make(map[string]map[uint64]chan liveEvent)}
}

func (n *liveNotifier) subscribe(code string) (uint64, <-chan liveEvent) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.nextID++
	id := n.nextID
	if n.sessions[code] == nil {
		n.sessions[code] = make(map[uint64]chan liveEvent)
	}
	ch := make(chan liveEvent, 1)
	n.sessions[code][id] = ch
	return id, ch
}

func (n *liveNotifier) unsubscribe(code string, id uint64) {
	n.mu.Lock()
	defer n.mu.Unlock()
	subs := n.sessions[code]
	if subs == nil {
		return
	}
	if ch, ok := subs[id]; ok {
		delete(subs, id)
		close(ch)
	}
	if len(subs) == 0 {
		delete(n.sessions, code)
	}
}

func (n *liveNotifier) publish(code string, ev liveEvent) {
	n.mu.Lock()
	defer n.mu.Unlock()
	subs := n.sessions[code]
	for id, ch := range subs {
		select {
		case ch <- ev:
		default:
			if ev == liveStarted {
				delete(subs, id)
				close(ch)
			}
			// changed is an invalidation signal; a queued change already covers it.
		}
	}
	if len(subs) == 0 {
		delete(n.sessions, code)
	}
}

func (n *liveNotifier) subscriberCount(code string) int {
	n.mu.Lock()
	defer n.mu.Unlock()
	return len(n.sessions[code])
}
