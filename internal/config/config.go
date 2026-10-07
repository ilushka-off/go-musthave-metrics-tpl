// Package config loads the optional JSON configuration file of the agent and
// the server. File values have the lowest priority: they are applied only to
// options that were not set by a command-line flag or an environment variable.
package config

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
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

// Server is the server configuration file. Absent fields are nil.
type Server struct {
	Address       *string   `json:"address"`
	Restore       *bool     `json:"restore"`
	StoreInterval *Duration `json:"store_interval"`
	StoreFile     *string   `json:"store_file"`
	DatabaseDSN   *string   `json:"database_dsn"`
	CryptoKey     *string   `json:"crypto_key"`
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

// ExplicitFlags returns the names of the flags of fs set on the command line.
// Call it after fs.Parse.
func ExplicitFlags(fs *flag.FlagSet) map[string]bool {
	set := make(map[string]bool)
	fs.Visit(func(f *flag.Flag) { set[f.Name] = true })
	return set
}

// Merger applies config file values to options whose flag was not set
// explicitly on the command line. A nil file value leaves the option as is.
type Merger struct {
	explicit map[string]bool
}

// NewMerger creates a Merger for the flags of fs. Call it after fs.Parse.
func NewMerger(fs *flag.FlagSet) Merger {
	return Merger{explicit: ExplicitFlags(fs)}
}

// String sets *dst to *v unless the flag name was set explicitly.
func (m Merger) String(name string, dst, v *string) { merge(m, name, dst, v) }

// Int sets *dst to *v unless the flag name was set explicitly.
func (m Merger) Int(name string, dst, v *int) { merge(m, name, dst, v) }

// Bool sets *dst to *v unless the flag name was set explicitly.
func (m Merger) Bool(name string, dst, v *bool) { merge(m, name, dst, v) }

// Duration sets *dst to *v unless the flag name was set explicitly. The value
// is kept as a time.Duration, so sub-second intervals are not truncated.
func (m Merger) Duration(name string, dst *time.Duration, v *Duration) {
	merge(m, name, dst, (*time.Duration)(v))
}

func merge[T any](m Merger, name string, dst, v *T) {
	if v != nil && !m.explicit[name] {
		*dst = *v
	}
}

// EnvString sets *dst to the value of the environment variable name, if set.
func EnvString(name string, dst *string) {
	if v, ok := os.LookupEnv(name); ok {
		*dst = v
	}
}

// EnvInt sets *dst to the integer value of the environment variable name, if set.
func EnvInt(name string, dst *int) error {
	return env(name, dst, strconv.Atoi)
}

// EnvBool sets *dst to the boolean value of the environment variable name, if set.
func EnvBool(name string, dst *bool) error {
	return env(name, dst, strconv.ParseBool)
}

// EnvSeconds sets *dst from the environment variable name, if set, holding
// a whole number of seconds.
func EnvSeconds(name string, dst *time.Duration) error {
	return env(name, dst, func(s string) (time.Duration, error) {
		n, err := strconv.Atoi(s)
		return time.Duration(n) * time.Second, err
	})
}

func env[T any](name string, dst *T, parse func(string) (T, error)) error {
	s, ok := os.LookupEnv(name)
	if !ok {
		return nil
	}
	v, err := parse(s)
	if err != nil {
		return fmt.Errorf("env %s: %w", name, err)
	}
	*dst = v
	return nil
}
