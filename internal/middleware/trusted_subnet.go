package middleware

import (
	"net"
	"net/http"
	"strings"

	"go.uber.org/zap"
)

// TrustedSubnet returns middleware that allows only requests whose "X-Real-IP"
// header holds an IP address inside subnet; any other request (missing,
// malformed or foreign address) is rejected with 403. If subnet is nil, the
// middleware is a no-op.
func TrustedSubnet(subnet *net.IPNet, log *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if subnet == nil {
				next.ServeHTTP(w, r)
				return
			}

			ip := net.ParseIP(strings.TrimSpace(r.Header.Get("X-Real-IP")))
			if ip == nil || !subnet.Contains(ip) {
				log.Warn("request from untrusted address rejected",
					zap.String("x_real_ip", r.Header.Get("X-Real-IP")))
				w.WriteHeader(http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
