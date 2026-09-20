package audit

import (
	"encoding/json"
	"os"
)

type FileObserver struct {
	path string
}

func NewFileObserver(path string) *FileObserver {
	return &FileObserver{
		path: path,
	}
}

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
