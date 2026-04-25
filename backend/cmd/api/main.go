package main

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"syscall"

	"github.com/Low-Stack-Technologies/Orivis/backend/internal/apigen"
	"github.com/Low-Stack-Technologies/Orivis/backend/internal/server"
	"github.com/go-chi/chi/v5"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	cfg, err := server.LoadConfig()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	apiServer, err := server.New(ctx, cfg)
	if err != nil {
		log.Fatalf("init server: %v", err)
	}
	defer apiServer.Close()

	r := chi.NewRouter()
	apigen.HandlerFromMux(apiServer, r)

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	addr := ":8080"
	log.Printf("orivis api listening on %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatal(err)
	}
}
