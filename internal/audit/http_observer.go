package audit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// HTTPObserver is an Observer that forwards each audit Event as a JSON POST
// request to a remote URL.
type HTTPObserver struct {
	url    string
	client *http.Client
}

// NewHTTPObserver creates an HTTPObserver that POSTs events to url using a
// client with a bounded timeout.
func NewHTTPObserver(url string) *HTTPObserver {
	return &HTTPObserver{
		url: url,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Notify POSTs event as JSON to the observer's URL. It returns an error if
// the request fails or the remote server responds with a non-200 status.
func (ho *HTTPObserver) Notify(event Event) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	body := bytes.NewReader(data)
	resp, err := ho.client.Post(ho.url, "application/json", body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("http status code: %d", resp.StatusCode)
	}
	return nil

}
