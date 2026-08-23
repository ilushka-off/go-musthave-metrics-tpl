package agent

import (
	"fmt"
	"net/http"
)

func sendMetrics(serverAddress, mType, name, value string) error {
	url := fmt.Sprintf("%s/update/%s/%s/%s", serverAddress, mType, name, value)

	resp, err := http.Post(url, "text/plain", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code %d for %s", resp.StatusCode, url)
	}

	return nil
}
