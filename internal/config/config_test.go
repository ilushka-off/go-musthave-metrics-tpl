package config

import (
	"flag"
	"os"
	"path/filepath"
	"testing"
	"time"
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
	if *c.Address != "h:1" || !*c.Restore || time.Duration(*c.StoreInterval) != 2*time.Second ||
		*c.StoreFile != "f.db" || *c.DatabaseDSN != "dsn" || *c.CryptoKey != "k.pem" {
		t.Fatalf("unexpected config: %+v", c)
	}
}

func TestLoadAgentAbsentFieldsAreNil(t *testing.T) {
	c, err := LoadAgent(write(t, `{"poll_interval":"3s"}`))
	if err != nil {
		t.Fatal(err)
	}
	if c.Address != nil || c.ReportInterval != nil || time.Duration(*c.PollInterval) != 3*time.Second {
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

func TestMergerSkipsExplicitFlags(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	addr := fs.String("a", "default", "")
	key := fs.String("k", "default", "")
	interval := time.Second
	fs.Int("r", 1, "")
	if err := fs.Parse([]string{"-a", "cli"}); err != nil {
		t.Fatal(err)
	}

	c, err := LoadAgent(write(t, `{"address":"file","key":"file","report_interval":"500ms"}`))
	if err != nil {
		t.Fatal(err)
	}

	m := NewMerger(fs)
	m.String("a", addr, c.Address)
	m.String("k", key, c.Key)
	m.Duration("r", &interval, c.ReportInterval)
	m.Int("l", new(int), c.RateLimit)

	if *addr != "cli" || *key != "file" || interval != 500*time.Millisecond {
		t.Fatalf("got addr=%q key=%q interval=%v", *addr, *key, interval)
	}
}

func TestEnvHelpers(t *testing.T) {
	t.Setenv("TEST_SECONDS", "5")
	t.Setenv("TEST_BAD_INT", "x")

	d := time.Second
	if err := EnvSeconds("TEST_SECONDS", &d); err != nil || d != 5*time.Second {
		t.Fatalf("got %v, %v", d, err)
	}
	if err := EnvSeconds("TEST_UNSET", &d); err != nil || d != 5*time.Second {
		t.Fatalf("unset env changed value: %v, %v", d, err)
	}

	n := 1
	if err := EnvInt("TEST_BAD_INT", &n); err == nil || n != 1 {
		t.Fatalf("expected error and unchanged value, got %d, %v", n, err)
	}
}
