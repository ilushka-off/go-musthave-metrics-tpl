package audit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestFileObserver_Notify_WritesJSONLine(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.log")
	obs := NewFileObserver(path)

	event := Event{Timestamp: 123, Metrics: []string{"Alloc"}, IPAddress: "127.0.0.1"}
	if err := obs.Notify(event); err != nil {
		t.Fatalf("Notify() error = %v; want nil", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if !strings.HasSuffix(string(data), "\n") {
		t.Fatalf("file content does not end with a newline: %q", data)
	}

	var got Event
	if err := json.Unmarshal([]byte(strings.TrimSuffix(string(data), "\n")), &got); err != nil {
		t.Fatalf("failed to unmarshal written line: %v", err)
	}
	if !reflect.DeepEqual(got, event) {
		t.Fatalf("got %+v; want %+v", got, event)
	}
}

func TestFileObserver_Notify_AppendsOnSecondCall(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.log")
	obs := NewFileObserver(path)

	first := Event{Timestamp: 1, Metrics: []string{"Alloc"}, IPAddress: "127.0.0.1"}
	second := Event{Timestamp: 2, Metrics: []string{"Frees"}, IPAddress: "127.0.0.1"}

	if err := obs.Notify(first); err != nil {
		t.Fatalf("Notify(first) error = %v", err)
	}
	if err := obs.Notify(second); err != nil {
		t.Fatalf("Notify(second) error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("got %d lines; want 2 (content: %q)", len(lines), data)
	}
}

func TestFileObserver_Notify_InvalidPathReturnsError(t *testing.T) {
	// A directory can't be opened for writing as a file.
	obs := NewFileObserver(t.TempDir())

	if err := obs.Notify(Event{}); err == nil {
		t.Fatal("Notify() error = nil; want non-nil for a path that is a directory")
	}
}
