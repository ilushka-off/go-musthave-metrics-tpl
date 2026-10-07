package main

import (
	"context"
	"crypto/rsa"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ilushka-off/go-musthave-metrics-tpl/internal/agent"
	"github.com/ilushka-off/go-musthave-metrics-tpl/internal/config"
	"github.com/ilushka-off/go-musthave-metrics-tpl/internal/crypto"
)

// Build information, set at link time via -ldflags "-X main.buildVersion=...".
var (
	buildVersion = "N/A"
	buildDate    = "N/A"
	buildCommit  = "N/A"
)

func main() {
	fmt.Printf("Build version: %s\n", buildVersion)
	fmt.Printf("Build date: %s\n", buildDate)
	fmt.Printf("Build commit: %s\n", buildCommit)

	addr := flag.String("a", "localhost:8080", "HTTP server address")
	reportSeconds := flag.Int("r", 10, "Report interval in seconds")
	pollSeconds := flag.Int("p", 2, "Poll interval in seconds")
	key := flag.String("k", "", "Key for hashing")
	rateLimit := flag.Int("l", 1, "Rate limit")
	cryptoKey := flag.String("crypto-key", "", "Path to the public key file for encryption")
	configShort := flag.String("c", "", "Path to JSON config file")
	configLong := flag.String("config", "", "Path to JSON config file")
	flag.Parse()

	// Intervals are kept as time.Duration from here on, so sub-second values
	// from the config file (e.g. "500ms") are not truncated to zero.
	reportInterval := time.Duration(*reportSeconds) * time.Second
	pollInterval := time.Duration(*pollSeconds) * time.Second

	if path := config.Path(*configShort, *configLong); path != "" {
		fc, err := config.LoadAgent(path)
		if err != nil {
			log.Fatal(err)
		}

		m := config.NewMerger(flag.CommandLine)
		m.String("a", addr, fc.Address)
		m.Duration("r", &reportInterval, fc.ReportInterval)
		m.Duration("p", &pollInterval, fc.PollInterval)
		m.String("crypto-key", cryptoKey, fc.CryptoKey)
		m.String("k", key, fc.Key)
		m.Int("l", rateLimit, fc.RateLimit)
	}

	config.EnvString("ADDRESS", addr)
	config.EnvString("KEY", key)
	config.EnvString("CRYPTO_KEY", cryptoKey)
	if err := errors.Join(
		config.EnvSeconds("REPORT_INTERVAL", &reportInterval),
		config.EnvSeconds("POLL_INTERVAL", &pollInterval),
		config.EnvInt("RATE_LIMIT", rateLimit),
	); err != nil {
		log.Fatal(err)
	}

	if reportInterval <= 0 || pollInterval <= 0 {
		log.Fatalf("report and poll intervals must be positive, got %v and %v", reportInterval, pollInterval)
	}

	var publicKey *rsa.PublicKey
	if *cryptoKey != "" {
		var err error
		publicKey, err = crypto.LoadPublicKey(*cryptoKey)
		if err != nil {
			log.Fatal(err)
		}
	}

	serverAddress := "http://" + *addr

	a := agent.NewAgent(serverAddress, pollInterval, reportInterval, *key, *rateLimit, publicKey)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
	defer stop()

	a.RunContext(ctx)

}
