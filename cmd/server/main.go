package main

import (
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/ilushka-off/go-musthave-metrics-tpl/internal/handler"
	"github.com/ilushka-off/go-musthave-metrics-tpl/internal/repository"
	"go.uber.org/zap"
)

func main() {

	addr := flag.String("a", "localhost:8080", "HTTP server address")
	flag.Parse()

	if envAddr := os.Getenv("ADDRESS"); envAddr != "" {
		*addr = envAddr
	}

	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatal(err)
	}

	defer logger.Sync()

	storage := repository.NewMemStorage()
	h := handler.NewMetricsHandler(storage)
	router := handler.NewRouter(h, logger)
	if err := http.ListenAndServe(*addr, router); err != nil {
		log.Fatal(err)
	}

}
