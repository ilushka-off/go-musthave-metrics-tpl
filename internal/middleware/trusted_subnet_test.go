package middleware

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"
)

func TestTrustedSubnet(t *testing.T) {
	_, subnet, _ := net.ParseCIDR("192.168.1.0/24")
	ok := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

	tests := []struct {
		name   string
		subnet *net.IPNet
		header string
		want   int
	}{
		{"inside subnet", subnet, "192.168.1.42", http.StatusOK},
		{"outside subnet", subnet, "10.0.0.1", http.StatusForbidden},
		{"missing header", subnet, "", http.StatusForbidden},
		{"malformed header", subnet, "not-an-ip", http.StatusForbidden},
		{"no subnet configured", nil, "", http.StatusOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/updates", nil)
			if tt.header != "" {
				req.Header.Set("X-Real-IP", tt.header)
			}
			rec := httptest.NewRecorder()
			TrustedSubnet(tt.subnet, zap.NewNop())(ok).ServeHTTP(rec, req)
			if rec.Code != tt.want {
				t.Fatalf("got %d, want %d", rec.Code, tt.want)
			}
		})
	}
}
