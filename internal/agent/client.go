package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/ilushka-off/go-musthave-metrics-tpl/internal/compress"
	models "github.com/ilushka-off/go-musthave-metrics-tpl/internal/model"
)

func sendMetrics(serverAddress, mType, name, value string) error {
	reqURL, err := url.JoinPath(serverAddress, "update", mType, name, value)
	if err != nil {
		return err
	}

	resp, err := http.Post(reqURL, "text/plain", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code %d for %s", resp.StatusCode, reqURL)
	}

	return nil
}

func sendMetricsJSON(serverAddress string, metrics models.Metrics) error {
	data, err := json.Marshal(metrics)
	if err != nil {
		return err
	}

	gzData, err := compress.Compress(data)
	if err != nil {
		return err
	}

	reqURL, err := url.JoinPath(serverAddress, "update")
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", reqURL, bytes.NewReader(gzData))
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
		return fmt.Errorf("unexpected status code %d for %s", resp.StatusCode, reqURL)
	}

	return nil
}
