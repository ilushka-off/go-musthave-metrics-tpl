package main

import (
	"crypto/rsa"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/ilushka-off/go-musthave-metrics-tpl/internal/agent"
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
	reportInterval := flag.Int("r", 10, "Report interval in seconds")
	pollInterval := flag.Int("p", 2, "Poll interval in seconds")
	key := flag.String("k", "", "Key for hashing")
	rateLimit := flag.Int("l", 1, "Rate limit")
	cryptoKey := flag.String("crypto-key", "", "Path to the public key file for encryption")
	flag.Parse()

	if envAddr, ok := os.LookupEnv("ADDRESS"); ok {
		*addr = envAddr
	}

	if envReportInterval, ok := os.LookupEnv("REPORT_INTERVAL"); ok {
		var err error
		*reportInterval, err = strconv.Atoi(envReportInterval)

		if err != nil {
			log.Fatal(err)
		}

	}

	if envPollInterval, ok := os.LookupEnv("POLL_INTERVAL"); ok {
		var err error

		*pollInterval, err = strconv.Atoi(envPollInterval)

		if err != nil {
			log.Fatal(err)
		}
	}

	if hashKey, ok := os.LookupEnv("KEY"); ok {
		*key = hashKey
	}

	if envRateLimit, ok := os.LookupEnv("RATE_LIMIT"); ok {
		var err error

		*rateLimit, err = strconv.Atoi(envRateLimit)
		if err != nil {
			log.Fatal(err)
		}
	}

	if envCryptoKey, ok := os.LookupEnv("CRYPTO_KEY"); ok {
		*cryptoKey = envCryptoKey
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

	a := agent.NewAgent(serverAddress, time.Duration(*pollInterval)*time.Second, time.Duration(*reportInterval)*time.Second, *key, *rateLimit, publicKey)
	a.Run()

}
