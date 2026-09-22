package audit

import (
	"errors"
	"reflect"
	"testing"

	"go.uber.org/zap"
)

var errBoom = errors.New("boom")

type fakeObserver struct {
	calls []Event
	err   error
}

func (f *fakeObserver) Notify(event Event) error {
	f.calls = append(f.calls, event)
	return f.err
}

func TestAuditor_Notify_NoObservers(t *testing.T) {
	a := NewAuditor(zap.NewNop())

	// Should be a no-op: no observers attached, must not panic.
	a.Notify(Event{Timestamp: 1, Metrics: []string{"Alloc"}, IPAddress: "127.0.0.1"})
}

func TestAuditor_Notify_CallsAllObservers(t *testing.T) {
	a := NewAuditor(zap.NewNop())
	first := &fakeObserver{}
	second := &fakeObserver{}
	a.Attach(first)
	a.Attach(second)

	event := Event{Timestamp: 42, Metrics: []string{"Alloc", "Frees"}, IPAddress: "127.0.0.1"}
	a.Notify(event)
	a.Close() // drains the per-observer goroutines so calls below are visible

	for name, obs := range map[string]*fakeObserver{"first": first, "second": second} {
		if len(obs.calls) != 1 {
			t.Fatalf("%s observer: got %d calls; want 1", name, len(obs.calls))
		}
		if !reflect.DeepEqual(obs.calls[0], event) {
			t.Fatalf("%s observer: got %+v; want %+v", name, obs.calls[0], event)
		}
	}
}

func TestAuditor_Notify_ContinuesAfterObserverError(t *testing.T) {
	a := NewAuditor(zap.NewNop())
	failing := &fakeObserver{err: errBoom}
	ok := &fakeObserver{}
	a.Attach(failing)
	a.Attach(ok)

	a.Notify(Event{Timestamp: 1})
	a.Close() // drains the per-observer goroutines so calls below are visible

	if len(failing.calls) != 1 {
		t.Fatalf("failing observer: got %d calls; want 1", len(failing.calls))
	}
	if len(ok.calls) != 1 {
		t.Fatalf("ok observer: got %d calls; want 1 (must still be notified after the previous observer failed)", len(ok.calls))
	}
}
