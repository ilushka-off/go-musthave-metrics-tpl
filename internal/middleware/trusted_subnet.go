package middleware

import (
	"net"
	"net/http"

	"github.com/ilushka-off/go-musthave-metrics-tpl/internal/subnet"
	"go.uber.org/zap"
)

// TrustedSubnet returns middleware that allows only requests whose "X-Real-IP"
// header holds an IP address inside trusted; any other request (missing,
// malformed or foreign address) is rejected with 403. If trusted is nil, the
// middleware is a no-op.
func TrustedSubnet(trusted *net.IPNet, log *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if trusted == nil {
				next.ServeHTTP(w, r)
				return
			}

			if !subnet.Allowed(trusted, r.Header.Get("X-Real-IP")) {
				log.Warn("request from untrusted address rejected",
					zap.String("x_real_ip", r.Header.Get("X-Real-IP")))
				w.WriteHeader(http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
