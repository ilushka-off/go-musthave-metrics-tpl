package subnet

import (
	"net"
	"testing"
)

func TestAllowed(t *testing.T) {
	_, trusted, err := net.ParseCIDR("192.168.1.0/24")
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		trusted *net.IPNet
		value   string
		want    bool
	}{
		{"inside subnet", trusted, "192.168.1.7", true},
		{"surrounding spaces", trusted, " 192.168.1.7 ", true},
		{"outside subnet", trusted, "10.0.0.1", false},
		{"malformed", trusted, "nope", false},
		{"empty", trusted, "", false},
		{"nil subnet", nil, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Allowed(tt.trusted, tt.value); got != tt.want {
				t.Fatalf("Allowed(%v, %q) = %v, want %v", tt.trusted, tt.value, got, tt.want)
			}
		})
	}
}
