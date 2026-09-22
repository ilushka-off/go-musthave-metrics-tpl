package audit

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestHTTPObserver_Notify_SendsEventAsJSON(t *testing.T) {
	var receivedContentType string
	var receivedEvent Event

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedContentType = r.Header.Get("Content-Type")
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("failed to read request body: %v", err)
		}
		if err := json.Unmarshal(body, &receivedEvent); err != nil {
			t.Errorf("failed to unmarshal request body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	obs := NewHTTPObserver(server.URL)
	event := Event{Timestamp: 123, Metrics: []string{"Alloc", "Frees"}, IPAddress: "127.0.0.1"}

	if err := obs.Notify(event); err != nil {
		t.Fatalf("Notify() error = %v; want nil", err)
	}
	if receivedContentType != "application/json" {
		t.Fatalf("Content-Type = %q; want application/json", receivedContentType)
	}
	if !reflect.DeepEqual(receivedEvent, event) {
		t.Fatalf("server received %+v; want %+v", receivedEvent, event)
	}
}

func TestHTTPObserver_Notify_NonOKStatusReturnsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	obs := NewHTTPObserver(server.URL)

	err := obs.Notify(Event{})
	if err == nil {
		t.Fatal("Notify() error = nil; want non-nil for a non-200 response")
	}
}

func TestHTTPObserver_Notify_UnreachableServerReturnsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := server.URL
	server.Close() // closed immediately: url is now unreachable

	obs := NewHTTPObserver(url)

	if err := obs.Notify(Event{}); err == nil {
		t.Fatal("Notify() error = nil; want non-nil for an unreachable server")
	}
}
