// Package config loads the optional JSON configuration file of the agent and
// the server. File values have the lowest priority: they are applied only to
// options that were not set by a command-line flag or an environment variable.
package config

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"
)

// Duration is a time.Duration that is decoded from a JSON string such as "1s".
type Duration time.Duration

// UnmarshalJSON parses a duration string in time.ParseDuration format.
func (d *Duration) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return fmt.Errorf("duration must be a string like \"1s\": %w", err)
	}
	v, err := time.ParseDuration(s)
	if err != nil {
		return fmt.Errorf("parse duration %q: %w", s, err)
	}
	*d = Duration(v)
	return nil
}

// Seconds returns d as a whole number of seconds.
func (d Duration) Seconds() int { return int(time.Duration(d).Seconds()) }

// Server is the server configuration file. Absent fields are nil.
type Server struct {
	Address       *string   `json:"address"`
	Restore       *bool     `json:"restore"`
	StoreInterval *Duration `json:"store_interval"`
	StoreFile     *string   `json:"store_file"`
	DatabaseDSN   *string   `json:"database_dsn"`
	CryptoKey     *string   `json:"crypto_key"`
	TrustedSubnet *string   `json:"trusted_subnet"`
	Key           *string   `json:"key"`
	AuditFile     *string   `json:"audit_file"`
	AuditURL      *string   `json:"audit_url"`
}

// Agent is the agent configuration file. Absent fields are nil.
type Agent struct {
	Address        *string   `json:"address"`
	ReportInterval *Duration `json:"report_interval"`
	PollInterval   *Duration `json:"poll_interval"`
	CryptoKey      *string   `json:"crypto_key"`
	Key            *string   `json:"key"`
	RateLimit      *int      `json:"rate_limit"`
}

// LoadServer reads the server configuration from path.
func LoadServer(path string) (*Server, error) {
	var c Server
	if err := load(path, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

// LoadAgent reads the agent configuration from path.
func LoadAgent(path string) (*Agent, error) {
	var c Agent
	if err := load(path, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

func load(path string, v any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read config file: %w", err)
	}
	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("parse config file %s: %w", path, err)
	}
	return nil
}

// Path returns the config file path: the CONFIG environment variable if set,
// otherwise the value of the -c / -config flags (empty if none was given).
func Path(flagShort, flagLong string) string {
	if env, ok := os.LookupEnv("CONFIG"); ok {
		return env
	}
	if flagLong != "" {
		return flagLong
	}
	return flagShort
}

// ExplicitFlags returns the names of the flags set on the command line.
// Call it after flag.Parse.
func ExplicitFlags() map[string]bool {
	set := make(map[string]bool)
	flag.Visit(func(f *flag.Flag) { set[f.Name] = true })
	return set
}
