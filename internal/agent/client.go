package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	models "github.com/ilushka-off/go-musthave-metrics-tpl/internal/model"
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

func sendMetricsJSON(serverAddress string, metrics models.Metrics) error {
	data, err := json.Marshal(metrics)
	if err != nil {
		return err
	}

	serverAddress = fmt.Sprintf("%s/update", serverAddress)
	resp, err := http.Post(serverAddress, "application/json", bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code %d for %s", resp.StatusCode, serverAddress)
	}
	return nil
}
