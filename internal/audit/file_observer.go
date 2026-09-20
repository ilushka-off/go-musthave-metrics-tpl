package audit

import (
	"encoding/json"
	"os"
)

// FileObserver is an Observer that appends each audit Event as a single
// JSON line to a file, creating the file if it does not exist.
type FileObserver struct {
	path string
}

// NewFileObserver creates a FileObserver that appends events to the file at
// path.
func NewFileObserver(path string) *FileObserver {
	return &FileObserver{
		path: path,
	}
}

// Notify appends event as a JSON-encoded line to the observer's file.
func (fo *FileObserver) Notify(event Event) error {
	file, err := os.OpenFile(fo.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	data = append(data, '\n')
	_, err = file.Write(data)
	if err != nil {
		return err
	}

	return nil
}
