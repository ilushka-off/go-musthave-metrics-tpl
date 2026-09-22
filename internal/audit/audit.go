package audit

import "go.uber.org/zap"

type Event struct {
	Timestamp int64    `json:"ts"`
	Metrics   []string `json:"metrics"`
	IPAddress string   `json:"ip_address"`
}

type Observer interface {
	Notify(event Event) error
}

type Auditor struct {
	observers []Observer
	log       *zap.Logger
}

func NewAuditor(log *zap.Logger) *Auditor {
	return &Auditor{
		log: log,
	}
}

func (a *Auditor) Attach(o Observer) {
	a.observers = append(a.observers, o)
}

func (a *Auditor) Notify(event Event) {
	for _, o := range a.observers {
		err := o.Notify(event)
		if err != nil {
			a.log.Error("error", zap.Error(err))
		}
	}
}
