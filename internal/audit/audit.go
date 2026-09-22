// Package audit implements request auditing via the Observer pattern: an
// Auditor notifies every attached Observer whenever metrics are successfully
// received, without the Auditor knowing how each observer stores or
// forwards the event.
package audit

import (
	"sync"

	"go.uber.org/zap"
)

// Event is a single audit record: which metrics were received, from which
// IP address, and when.
type Event struct {
	Timestamp int64    `json:"ts"`
	Metrics   []string `json:"metrics"`
	IPAddress string   `json:"ip_address"`
}

// Observer receives audit events. Implementations decide how to persist or
// forward the event (e.g. to a file or over HTTP).
type Observer interface {
	Notify(event Event) error
}

// observerQueueSize bounds how many pending events an observer's goroutine
// may buffer before Notify starts dropping events for it.
const observerQueueSize = 100

// Auditor is the subject in the Observer pattern: it fans out every Event
// passed to Notify to all attached Observers. Each observer runs in its own
// goroutine, consuming from a buffered channel, so a slow observer (e.g. a
// remote HTTP endpoint) cannot delay Notify's caller or other observers.
// An Auditor with no observers attached is a no-op, which represents
// "auditing disabled".
type Auditor struct {
	channels []chan Event
	log      *zap.Logger
	mu       sync.Mutex
	wg       sync.WaitGroup
}

// NewAuditor creates an Auditor with no observers attached.
func NewAuditor(log *zap.Logger) *Auditor {
	return &Auditor{
		log: log,
	}
}

// Attach subscribes o to receive future audit events. o is notified from a
// dedicated goroutine that runs until Close is called.
func (a *Auditor) Attach(o Observer) {
	ch := make(chan Event, observerQueueSize)

	a.mu.Lock()
	a.channels = append(a.channels, ch)
	a.mu.Unlock()

	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		for event := range ch {
			if err := o.Notify(event); err != nil {
				a.log.Error("failed to notify audit observer", zap.Error(err))
			}
		}
	}()
}

// Notify enqueues event for every attached observer without blocking. If an
// observer's queue is full, the event is dropped for that observer and the
// drop is logged.
func (a *Auditor) Notify(event Event) {
	a.mu.Lock()
	channels := a.channels
	a.mu.Unlock()

	for _, ch := range channels {
		select {
		case ch <- event:
		default:
			a.log.Error("audit observer queue full, dropping event")
		}
	}
}

// Close stops accepting new events and blocks until every observer has
// finished processing whatever was already queued. The caller must ensure
// Notify is no longer being called concurrently (e.g. by shutting down the
// HTTP server first), otherwise a send on a closed channel can panic.
func (a *Auditor) Close() {
	a.mu.Lock()
	channels := a.channels
	a.channels = nil
	a.mu.Unlock()

	for _, ch := range channels {
		close(ch)
	}
	a.wg.Wait()
}
