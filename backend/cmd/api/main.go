package main

import (
	"log"
	"net/http"

	"github.com/Low-Stack-Technologies/Orivis/backend/internal/apigen"
	"github.com/go-chi/chi/v5"
)

func main() {
	r := chi.NewRouter()
	h := apigen.NewStrictHandler(apigen.NewUnimplementedStrictServer(), nil)
	apigen.HandlerFromMux(h, r)

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
