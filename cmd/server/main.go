package main

import (
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/ilushka-off/go-musthave-metrics-tpl/internal/handler"
	"github.com/ilushka-off/go-musthave-metrics-tpl/internal/repository"
)

func main() {

	addr := flag.String("a", "localhost:8080", "HTTP server address")
	flag.Parse()

	if envAddr := os.Getenv("ADDRESS"); envAddr != "" {
		*addr = envAddr
	}

	storage := repository.NewMemStorage()
	h := handler.NewMetricsHandler(storage)
	router := handler.NewRouter(h)
	if err := http.ListenAndServe(*addr, router); err != nil {
		log.Fatal(err)
	}

}
