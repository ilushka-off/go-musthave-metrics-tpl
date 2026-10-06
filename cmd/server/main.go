package main

import (
	"context"
	"crypto/rsa"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/ilushka-off/go-musthave-metrics-tpl/internal/audit"
	"github.com/ilushka-off/go-musthave-metrics-tpl/internal/config"
	"github.com/ilushka-off/go-musthave-metrics-tpl/internal/crypto"
	"github.com/ilushka-off/go-musthave-metrics-tpl/internal/handler"
	"github.com/ilushka-off/go-musthave-metrics-tpl/internal/repository"
	_ "github.com/jackc/pgx/v5/stdlib"
	"go.uber.org/zap"
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
	storeInterval := flag.Int("i", 300, "Store interval in seconds")
	filePath := flag.String("f", "metrics.json", "File path to store metrics")
	restore := flag.Bool("r", false, "Restore metrics from file, if true")
	databaseDsn := flag.String("d", "", "Database DSN")
	key := flag.String("k", "", "Key for hashing")
	auditFile := flag.String("audit-file", "", "Audit flag")
	auditURL := flag.String("audit-url", "", "Audit HTTP by URL")
	cryptoKey := flag.String("crypto-key", "", "Path to the private key file for decryption")
	configShort := flag.String("c", "", "Path to JSON config file")
	configLong := flag.String("config", "", "Path to JSON config file")
	flag.Parse()

	if path := config.Path(*configShort, *configLong); path != "" {
		fc, err := config.LoadServer(path)
		if err != nil {
			log.Fatal(err)
		}

		set := config.ExplicitFlags()
		if fc.Address != nil && !set["a"] {
			*addr = *fc.Address
		}
		if fc.Restore != nil && !set["r"] {
			*restore = *fc.Restore
		}
		if fc.StoreInterval != nil && !set["i"] {
			*storeInterval = fc.StoreInterval.Seconds()
		}
		if fc.StoreFile != nil && !set["f"] {
			*filePath = *fc.StoreFile
		}
		if fc.DatabaseDSN != nil && !set["d"] {
			*databaseDsn = *fc.DatabaseDSN
		}
		if fc.CryptoKey != nil && !set["crypto-key"] {
			*cryptoKey = *fc.CryptoKey
		}
		if fc.Key != nil && !set["k"] {
			*key = *fc.Key
		}
		if fc.AuditFile != nil && !set["audit-file"] {
			*auditFile = *fc.AuditFile
		}
		if fc.AuditURL != nil && !set["audit-url"] {
			*auditURL = *fc.AuditURL
		}
	}

	if envAddr, ok := os.LookupEnv("ADDRESS"); ok {
		*addr = envAddr
	}

	if envStoreInterval, ok := os.LookupEnv("STORE_INTERVAL"); ok {
		var err error
		*storeInterval, err = strconv.Atoi(envStoreInterval)
		if err != nil {
			log.Fatal(err)
		}
	}

	if envFilePath, ok := os.LookupEnv("FILE_STORAGE_PATH"); ok {
		*filePath = envFilePath
	} else if envFilePath, ok := os.LookupEnv("STORE_FILE"); ok {
		*filePath = envFilePath
	}

	if envRestore, ok := os.LookupEnv("RESTORE"); ok {
		var err error
		*restore, err = strconv.ParseBool(envRestore)
		if err != nil {
			log.Fatal(err)
		}
	}

	if envDatabaseDsn := os.Getenv("DATABASE_DSN"); envDatabaseDsn != "" {
		*databaseDsn = envDatabaseDsn
	}

	if hashKey, ok := os.LookupEnv("KEY"); ok {
		*key = hashKey
	}

	if envCryptoKey, ok := os.LookupEnv("CRYPTO_KEY"); ok {
		*cryptoKey = envCryptoKey
	}

	if envAuditFile, ok := os.LookupEnv("AUDIT_FILE"); ok {
		*auditFile = envAuditFile
	}

	if envAuditURL, ok := os.LookupEnv("AUDIT_URL"); ok {
		*auditURL = envAuditURL
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
	defer stop()

	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatal(err)
	}

	defer func() { _ = logger.Sync() }()

	var privateKey *rsa.PrivateKey
	if *cryptoKey != "" {
		privateKey, err = crypto.LoadPrivateKey(*cryptoKey)
		if err != nil {
			logger.Fatal("failed to load private key", zap.Error(err))
		}
	}

	auditor := audit.NewAuditor(logger)
	var fileObserver *audit.FileObserver
	if *auditFile != "" {
		fileObserver, err = audit.NewFileObserver(*auditFile)
		if err != nil {
			logger.Fatal("failed to open audit file", zap.Error(err))
		}
		auditor.Attach(fileObserver)
	}
	if *auditURL != "" {
		url := audit.NewHTTPObserver(*auditURL)
		auditor.Attach(url)
	}

	var storage repository.Storage
	var saverDone chan struct{}
	var pingHandler *handler.PingHandler

	if *databaseDsn != "" {
		pingCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		var db *sql.DB
		db, err = sql.Open("pgx", *databaseDsn)
		if err != nil {
			logger.Fatal("Failed to connect to database", zap.Error(err))
		}
		err = db.PingContext(pingCtx)
		if err != nil {
			logger.Fatal("Failed to ping database", zap.Error(err))
		}
		err = repository.RunMigrations(db)
		if err != nil {
			logger.Fatal("Failed to run migrations", zap.Error(err))
		}
		storage = repository.NewPostgresStorage(db, logger)
		pingHandler = handler.NewPingHandler(db, logger)
		defer func() { _ = db.Close() }()
	} else if *filePath != "" {
		storage, err = repository.NewFileStorage(*filePath, *restore)
		if err != nil {
			logger.Error("Failed to create file storage", zap.Error(err))
		}

		if *storeInterval > 0 {
			saverDone = make(chan struct{})
			go func() {
				defer close(saverDone)
				ticker := time.NewTicker(time.Duration(*storeInterval) * time.Second)
				defer ticker.Stop()

				for {
					select {
					case <-ctx.Done():
						return
					case <-ticker.C:
					}

					err := repository.SaveToFile(storage, *filePath)
					if err != nil {
						logger.Error("Failed to save metrics to file", zap.Error(err))
					}
				}
			}()
		} else {
			storage = repository.NewSyncStorage(storage, *filePath)
		}
	} else {
		storage = repository.NewMemStorage()
	}

	h := handler.NewMetricsHandler(storage, logger, auditor)

	router := handler.NewRouter(h, logger, pingHandler, *key, privateKey)

	srv := &http.Server{Addr: *addr, Handler: router}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("HTTP server failed", zap.Error(err))
		}
	}()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("HTTP server shutdown error", zap.Error(err))
	}

	// Flush metrics that the periodic saver has not written yet. Requests
	// are drained at this point, so the snapshot is complete.
	if saverDone != nil {
		<-saverDone
		if err := repository.SaveToFile(storage, *filePath); err != nil {
			logger.Error("Failed to save metrics to file on shutdown", zap.Error(err))
		}
	}

	// The HTTP server has stopped accepting and has drained in-flight
	// requests, so no goroutine can call auditor.Notify concurrently anymore
	// -- safe to stop the observers now.
	auditor.Close()
	if fileObserver != nil {
		if err := fileObserver.Close(); err != nil {
			logger.Error("failed to close audit file", zap.Error(err))
		}
	}
}
