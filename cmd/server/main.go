package main

import (
	"database/sql"
	"flag"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

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

	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatal(err)
	}

	defer logger.Sync()

	var storage repository.Storage
	var pingHandler *handler.PingHandler

	if *databaseDsn != "" {
		db, err := sql.Open("pgx", *databaseDsn)
		if err != nil {
			logger.Fatal("Failed to connect to database", zap.Error(err))
		}
		err = repository.RunMigrations(db)
		if err != nil {
			logger.Error("Failed to run migrations", zap.Error(err))
		}
		storage = repository.NewPostgresStorage(db, logger)
		pingHandler = handler.NewPingHandler(db, logger)
		defer db.Close()
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

	h := handler.NewMetricsHandler(storage, logger)

	router := handler.NewRouter(h, logger, pingHandler)
	if err := http.ListenAndServe(*addr, router); err != nil {
		log.Fatal(err)
	}

}
