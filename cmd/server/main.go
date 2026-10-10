package main

import (
	"context"
	"crypto/rsa"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ilushka-off/go-musthave-metrics-tpl/internal/audit"
	"github.com/ilushka-off/go-musthave-metrics-tpl/internal/config"
	"github.com/ilushka-off/go-musthave-metrics-tpl/internal/crypto"
	"github.com/ilushka-off/go-musthave-metrics-tpl/internal/grpcserver"
	"github.com/ilushka-off/go-musthave-metrics-tpl/internal/handler"
	pb "github.com/ilushka-off/go-musthave-metrics-tpl/internal/proto"
	"github.com/ilushka-off/go-musthave-metrics-tpl/internal/repository"
	_ "github.com/jackc/pgx/v5/stdlib"
	"go.uber.org/zap"
	"google.golang.org/grpc"
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
	storeSeconds := flag.Int("i", 300, "Store interval in seconds")
	filePath := flag.String("f", "metrics.json", "File path to store metrics")
	restore := flag.Bool("r", false, "Restore metrics from file, if true")
	databaseDsn := flag.String("d", "", "Database DSN")
	key := flag.String("k", "", "Key for hashing")
	auditFile := flag.String("audit-file", "", "Audit flag")
	auditURL := flag.String("audit-url", "", "Audit HTTP by URL")
	cryptoKey := flag.String("crypto-key", "", "Path to the private key file for decryption")
	trustedSubnet := flag.String("t", "", "Trusted subnet in CIDR notation")
	grpcAddress := flag.String("grpc-address", "", "gRPC server address (gRPC is disabled if empty)")
	configShort := flag.String("c", "", "Path to JSON config file")
	configLong := flag.String("config", "", "Path to JSON config file")
	flag.Parse()

	// Kept as time.Duration so a sub-second store_interval from the config
	// file is not truncated to zero (which would switch to synchronous saving).
	storeInterval := time.Duration(*storeSeconds) * time.Second

	if path := config.Path(*configShort, *configLong); path != "" {
		fc, err := config.LoadServer(path)
		if err != nil {
			log.Fatal(err)
		}

		m := config.NewMerger(flag.CommandLine)
		m.String("a", addr, fc.Address)
		m.Bool("r", restore, fc.Restore)
		m.Duration("i", &storeInterval, fc.StoreInterval)
		m.String("f", filePath, fc.StoreFile)
		m.String("d", databaseDsn, fc.DatabaseDSN)
		m.String("crypto-key", cryptoKey, fc.CryptoKey)
		m.String("t", trustedSubnet, fc.TrustedSubnet)
		m.String("grpc-address", grpcAddress, fc.GRPCAddress)
		m.String("k", key, fc.Key)
		m.String("audit-file", auditFile, fc.AuditFile)
		m.String("audit-url", auditURL, fc.AuditURL)
	}

	config.EnvString("ADDRESS", addr)
	config.EnvString("TRUSTED_SUBNET", trustedSubnet)
	config.EnvString("GRPC_ADDRESS", grpcAddress)
	// FILE_STORAGE_PATH takes precedence over STORE_FILE.
	config.EnvString("STORE_FILE", filePath)
	config.EnvString("FILE_STORAGE_PATH", filePath)
	config.EnvString("KEY", key)
	config.EnvString("CRYPTO_KEY", cryptoKey)
	config.EnvString("AUDIT_FILE", auditFile)
	config.EnvString("AUDIT_URL", auditURL)
	if err := errors.Join(
		config.EnvSeconds("STORE_INTERVAL", &storeInterval),
		config.EnvBool("RESTORE", restore),
	); err != nil {
		log.Fatal(err)
	}
	if envDatabaseDsn := os.Getenv("DATABASE_DSN"); envDatabaseDsn != "" {
		*databaseDsn = envDatabaseDsn
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
	defer stop()

	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatal(err)
	}

	defer func() { _ = logger.Sync() }()

	var trusted *net.IPNet
	if *trustedSubnet != "" {
		_, trusted, err = net.ParseCIDR(*trustedSubnet)
		if err != nil {
			logger.Fatal("invalid trusted subnet", zap.Error(err))
		}
	}

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

		if storeInterval > 0 {
			saverDone = make(chan struct{})
			go func() {
				defer close(saverDone)
				ticker := time.NewTicker(storeInterval)
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

	router := handler.NewRouter(h, logger, pingHandler, *key, privateKey, trusted)

	srv := &http.Server{Addr: *addr, Handler: router}

	var grpcSrv *grpc.Server
	if *grpcAddress != "" {
		var lc net.ListenConfig
		lis, err := lc.Listen(ctx, "tcp", *grpcAddress)
		if err != nil {
			logger.Fatal("failed to listen for gRPC", zap.Error(err))
		}

		grpcSrv = grpc.NewServer(grpc.ChainUnaryInterceptor(grpcserver.TrustedSubnetInterceptor(trusted, logger)))
		pb.RegisterMetricsServer(grpcSrv, grpcserver.New(storage, logger, auditor))

		go func() {
			if err := grpcSrv.Serve(lis); err != nil {
				logger.Fatal("gRPC server failed", zap.Error(err))
			}
		}()
	}

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

	if grpcSrv != nil {
		stopped := make(chan struct{})
		go func() {
			grpcSrv.GracefulStop()
			close(stopped)
		}()
		select {
		case <-stopped:
		case <-shutdownCtx.Done():
			grpcSrv.Stop()
		}
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
