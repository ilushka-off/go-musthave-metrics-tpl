package audit

import (
	"encoding/json"
	"os"
	"sync"
)

// FileObserver is an Observer that appends each audit Event as a single
// JSON line to a file, creating the file if it does not exist.
type FileObserver struct {
	file *os.File
	mu   sync.Mutex
}

// NewFileObserver creates a FileObserver that appends events to the file at
// path.
func NewFileObserver(path string) (*FileObserver, error) {
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	return &FileObserver{
		file: file,
	}, nil
}

func (fo *FileObserver) Close() error {
	return fo.file.Close()
}

// Notify appends event as a JSON-encoded line to the observer's file.
func (fo *FileObserver) Notify(event Event) error {

	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	fo.mu.Lock()
	defer fo.mu.Unlock()
	_, err = fo.file.Write(data)
	if err != nil {
		return err
	}

	return nil
}
