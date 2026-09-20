// Package audit implements request auditing via the Observer pattern: an
// Auditor notifies every attached Observer whenever metrics are successfully
// received, without the Auditor knowing how each observer stores or
// forwards the event.
package audit

import "go.uber.org/zap"

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

// Auditor is the subject in the Observer pattern: it fans out every Event
// passed to Notify to all attached Observers, logging (but not propagating)
// individual observer failures. An Auditor with no observers attached is a
// no-op, which represents "auditing disabled".
type Auditor struct {
	observers []Observer
	log       *zap.Logger
}

// NewAuditor creates an Auditor with no observers attached.
func NewAuditor(log *zap.Logger) *Auditor {
	return &Auditor{
		log: log,
	}
}

// Attach subscribes o to receive future audit events.
func (a *Auditor) Attach(o Observer) {
	a.observers = append(a.observers, o)
}

// Notify delivers event to every attached observer. If an observer returns
// an error, it is logged and the remaining observers are still notified.
func (a *Auditor) Notify(event Event) {
	for _, o := range a.observers {
		err := o.Notify(event)
		if err != nil {
			a.log.Error("error", zap.Error(err))
		}
	}
}
