package main

import (
	"log"
	"net/http"

	"github.com/ilushka-off/go-musthave-metrics-tpl/internal/handler"
	"github.com/ilushka-off/go-musthave-metrics-tpl/internal/repository"
)

func main() {

	storage := repository.NewMemStorage()
	h := handler.NewMetricsHandler(storage)
	router := handler.NewRouter(h)
	if err := http.ListenAndServe(":8080", router); err != nil {
		log.Fatal(err)
	}

}
