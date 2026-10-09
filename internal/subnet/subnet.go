// Package subnet holds the trusted-subnet check shared by the HTTP middleware
// and the gRPC interceptor.
package subnet

import (
	"net"
	"strings"
)

// Allowed reports whether value is an IP address inside trusted. Surrounding
// whitespace is ignored; an empty or malformed value is never allowed. A nil
// trusted subnet allows every value.
func Allowed(trusted *net.IPNet, value string) bool {
	if trusted == nil {
		return true
	}
	ip := net.ParseIP(strings.TrimSpace(value))
	return ip != nil && trusted.Contains(ip)
}
