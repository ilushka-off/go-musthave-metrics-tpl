package main

import (
	"context"
	"database/sql"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/ilushka-off/go-musthave-metrics-tpl/internal/audit"
	"github.com/ilushka-off/go-musthave-metrics-tpl/internal/handler"
	"github.com/ilushka-off/go-musthave-metrics-tpl/internal/repository"
	_ "github.com/jackc/pgx/v5/stdlib"
	"go.uber.org/zap"
)

func main() {

	addr := flag.String("a", "localhost:8080", "HTTP server address")
	storeInterval := flag.Int("i", 300, "Store interval in seconds")
	filePath := flag.String("f", "metrics.json", "File path to store metrics")
	restore := flag.Bool("r", false, "Restore metrics from file, if true")
	databaseDsn := flag.String("d", "", "Database DSN")
	key := flag.String("k", "", "Key for hashing")
	auditFile := flag.String("audit-file", "", "Audit flag")
	auditURL := flag.String("audit-url", "", "Audit HTTP by URL")
	flag.Parse()

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

	if envAuditFile, ok := os.LookupEnv("AUDIT_FILE"); ok {
		*auditFile = envAuditFile
	}

	if envAuditURL, ok := os.LookupEnv("AUDIT_URL"); ok {
		*auditURL = envAuditURL
	}

	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatal(err)
	}

	defer func() { _ = logger.Sync() }()

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
	var pingHandler *handler.PingHandler

	if *databaseDsn != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		var db *sql.DB
		db, err = sql.Open("pgx", *databaseDsn)
		if err != nil {
			logger.Fatal("Failed to connect to database", zap.Error(err))
		}
		err = db.PingContext(ctx)
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
			go func() {
				ticker := time.NewTicker(time.Duration(*storeInterval) * time.Second)

				for range ticker.C {
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

	router := handler.NewRouter(h, logger, pingHandler, *key)

	srv := &http.Server{Addr: *addr, Handler: router}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("HTTP server failed", zap.Error(err))
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("HTTP server shutdown error", zap.Error(err))
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
