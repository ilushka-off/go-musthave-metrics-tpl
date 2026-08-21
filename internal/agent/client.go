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

	return nil
}
