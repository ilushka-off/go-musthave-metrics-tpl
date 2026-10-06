package agent

import (
	"bytes"
	"context"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"

	"github.com/ilushka-off/go-musthave-metrics-tpl/internal/compress"
	"github.com/ilushka-off/go-musthave-metrics-tpl/internal/crypto"
	"github.com/ilushka-off/go-musthave-metrics-tpl/internal/hash"
	models "github.com/ilushka-off/go-musthave-metrics-tpl/internal/model"
	"github.com/ilushka-off/go-musthave-metrics-tpl/internal/retry"
)

func sendMetrics(serverAddress, mType, name, value string) error {
	reqURL, err := url.JoinPath(serverAddress, "update", mType, name, value)
	if err != nil {
		return fmt.Errorf("build request url: %w", err)
	}

	realIP := localIP(serverAddress)

	return retry.Do(retry.Delays, isConnRetriable, func() error {
		req, err := http.NewRequest("POST", reqURL, nil)
		if err != nil {
			return fmt.Errorf("build request: %w", err)
		}
		req.Header.Set("Content-Type", "text/plain")
		req.Header.Set("X-Real-IP", realIP)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return fmt.Errorf("post metric: %w", err)
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("unexpected status code %d for %s", resp.StatusCode, reqURL)
		}

		return nil
	})
}

func sendMetricsJSON(serverAddress string, metrics models.Metrics) error {
	data, err := json.Marshal(metrics)
	if err != nil {
		return fmt.Errorf("marshal metrics: %w", err)
	}

	gzData, err := compress.Compress(data)
	if err != nil {
		return fmt.Errorf("compress metrics: %w", err)
	}

	reqURL, err := url.JoinPath(serverAddress, "update")
	if err != nil {
		return fmt.Errorf("build request url: %w", err)
	}

	realIP := localIP(serverAddress)

	return retry.Do(retry.Delays, isConnRetriable, func() error {
		req, err := http.NewRequest("POST", reqURL, bytes.NewReader(gzData))
		if err != nil {
			return fmt.Errorf("build request: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Content-Encoding", "gzip")
		req.Header.Set("X-Real-IP", realIP)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return fmt.Errorf("post metrics: %w", err)
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("unexpected status code %d for %s", resp.StatusCode, reqURL)
		}

		return nil
	})
}

func sendMetricsBatch(serverAddress string, metrics []models.Metrics, hashKey string, publicKey *rsa.PublicKey) error {
	data, err := json.Marshal(metrics)
	if err != nil {
		return fmt.Errorf("marshal metrics: %w", err)
	}

	var signature string

	if hashKey != "" {
		signature = hash.Sign(data, hashKey)
	}
	gzData, err := compress.Compress(data)
	if err != nil {
		return fmt.Errorf("compress metrics: %w", err)
	}

	body := gzData
	if publicKey != nil {
		body, err = crypto.Encrypt(publicKey, gzData)
		if err != nil {
			return fmt.Errorf("encrypt metrics: %w", err)
		}
	}
	reqURL, err := url.JoinPath(serverAddress, "updates")
	if err != nil {
		return fmt.Errorf("build request url: %w", err)
	}

	realIP := localIP(serverAddress)

	return retry.Do(retry.Delays, isConnRetriable, func() error {
		req, err := http.NewRequest("POST", reqURL, bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("build request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Content-Encoding", "gzip")
		req.Header.Set("X-Real-IP", realIP)

		if signature != "" {
			req.Header.Set("HashSHA256", signature)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return fmt.Errorf("post metrics: %w", err)
		}
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("unexpected status code %d for %s", resp.StatusCode, reqURL)
		}
		return nil
	})
}

func isConnRetriable(err error) bool {
	_, ok := errors.AsType[*net.OpError](err)
	return ok
}

// localIP returns the IP address of this host that is used to reach
// serverAddress, for the X-Real-IP header. It "connects" a UDP socket (which
// sends no packets) and reads the chosen local address, falling back to the
// first non-loopback interface address and finally to 127.0.0.1.
func localIP(serverAddress string) string {
	if u, err := url.Parse(serverAddress); err == nil && u.Host != "" {
		host := u.Host
		if _, _, err := net.SplitHostPort(host); err != nil {
			host = net.JoinHostPort(u.Hostname(), "80")
		}
		// IPv4 is tried first: a name such as "localhost" may resolve to
		// ::1 for UDP although the HTTP connection goes over IPv4.
		var d net.Dialer
		for _, network := range []string{"udp4", "udp"} {
			conn, err := d.DialContext(context.Background(), network, host)
			if err != nil {
				continue
			}
			addr, ok := conn.LocalAddr().(*net.UDPAddr)
			_ = conn.Close()
			if ok {
				return addr.IP.String()
			}
		}
	}

	if addrs, err := net.InterfaceAddrs(); err == nil {
		for _, a := range addrs {
			if ipNet, ok := a.(*net.IPNet); ok && !ipNet.IP.IsLoopback() && ipNet.IP.To4() != nil {
				return ipNet.IP.String()
			}
		}
	}

	return "127.0.0.1"
}
