package config

import (
	"os"
	"path/filepath"
	"testing"
)

func write(t *testing.T, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "cfg.json")
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLoadServer(t *testing.T) {
	c, err := LoadServer(write(t, `{"address":"h:1","restore":true,"store_interval":"2s","store_file":"f.db","database_dsn":"dsn","crypto_key":"k.pem"}`))
	if err != nil {
		t.Fatal(err)
	}
	if *c.Address != "h:1" || !*c.Restore || c.StoreInterval.Seconds() != 2 ||
		*c.StoreFile != "f.db" || *c.DatabaseDSN != "dsn" || *c.CryptoKey != "k.pem" {
		t.Fatalf("unexpected config: %+v", c)
	}
}

func TestLoadAgentAbsentFieldsAreNil(t *testing.T) {
	c, err := LoadAgent(write(t, `{"poll_interval":"3s"}`))
	if err != nil {
		t.Fatal(err)
	}
	if c.Address != nil || c.ReportInterval != nil || c.PollInterval.Seconds() != 3 {
		t.Fatalf("unexpected config: %+v", c)
	}
}

func TestLoadErrors(t *testing.T) {
	if _, err := LoadAgent(filepath.Join(t.TempDir(), "missing.json")); err == nil {
		t.Fatal("expected error for missing file")
	}
	if _, err := LoadAgent(write(t, `{"poll_interval":"abc"}`)); err == nil {
		t.Fatal("expected error for bad duration")
	}
	if _, err := LoadServer(write(t, `{`)); err == nil {
		t.Fatal("expected error for bad json")
	}
}

func TestPathEnvOverridesFlag(t *testing.T) {
	t.Setenv("CONFIG", "env.json")
	if got := Path("a.json", "b.json"); got != "env.json" {
		t.Fatalf("got %q", got)
	}
}
