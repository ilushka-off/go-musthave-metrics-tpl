package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/ilushka-off/go-musthave-metrics-tpl/internal/handler"
	"github.com/ilushka-off/go-musthave-metrics-tpl/internal/repository"
	"go.uber.org/zap"
)

func main() {

	addr := flag.String("a", "localhost:8080", "HTTP server address")
	storeInterval := flag.Int("i", 300, "Store interval in seconds")
	filePath := flag.String("f", "metrics.json", "File path to store metrics")
	restore := flag.Bool("r", false, "Restore metrics from file, if true")
	flag.Parse()

	if envAddr := os.Getenv("ADDRESS"); envAddr != "" {
		*addr = envAddr
	}

	if envStoreInterval := os.Getenv("STORE_INTERVAL"); envStoreInterval != "" {
		var err error
		*storeInterval, err = strconv.Atoi(envStoreInterval)
		if err != nil {
			log.Fatal(err)
		}
	}

	if envFilePath := os.Getenv("FILE_STORAGE_PATH"); envFilePath != "" {
		*filePath = envFilePath
	}

	if envRestore := os.Getenv("RESTORE"); envRestore != "" {
		var err error
		*restore, err = strconv.ParseBool(envRestore)
		if err != nil {
			log.Fatal(err)
		}
	}

	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatal(err)
	}

	defer logger.Sync()

	var storage repository.Storage = repository.NewMemStorage()

	if *restore == true {
		err := repository.LoadFromFile(storage, *filePath)
		if err != nil {
			logger.Error("Failed to restore metrics from file", zap.Error(err))
		}
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
	}

	if *storeInterval == 0 {
		storage = repository.NewSyncStorage(storage, *filePath)
	}

	h := handler.NewMetricsHandler(storage)
	router := handler.NewRouter(h, logger)
	if err := http.ListenAndServe(*addr, router); err != nil {
		log.Fatal(err)
	}

}
