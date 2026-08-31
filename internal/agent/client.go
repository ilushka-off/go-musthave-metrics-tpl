package agent

import (
	"bytes"
	"compress/gzip"
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
	var buf bytes.Buffer

	data, err := json.Marshal(metrics)
	if err != nil {
		return err
	}

	gz := gzip.NewWriter(&buf)
	if _, err := gz.Write(data); err != nil {
		return err
	}
	gz.Close()

	serverAddress = fmt.Sprintf("%s/update", serverAddress)
	req, err := http.NewRequest("POST", serverAddress, &buf)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code %d for %s", resp.StatusCode, serverAddress)
	}

	return nil
}
