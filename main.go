package main

import (
	"net/http"

	"github.com/thalesraymond/go-http-server/internal/handler"
)

func main() {

	serverMux := http.NewServeMux()

	server := &http.Server{
		Addr:    ":8080",
		Handler: serverMux,
	}

	serverMux.Handle("/app/", http.StripPrefix("/app", http.FileServer(http.Dir("./"))))

	serverMux.HandleFunc("/healthz", handler.HealthzHandler)

	server.ListenAndServe()
}
