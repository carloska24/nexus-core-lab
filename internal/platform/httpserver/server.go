package httpserver

import (
	"encoding/json"
	"log"
	"net/http"
)

type healthResponse struct {
	Status  string `json:"status"`
	Service string `json:"service"`
}

// RouteRegistrar define uma função capaz de acoplar rotas a um ServeMux.
type RouteRegistrar func(mux *http.ServeMux)

// New instancia o roteador HTTP com o endpoint /health e registra módulos adicionais fornecidos.
func New(registrars ...RouteRegistrar) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", healthHandler)

	for _, register := range registrars {
		if register != nil {
			register(mux)
		}
	}

	return mux
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	response := healthResponse{
		Status:  "ok",
		Service: "nexus-core-lab",
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("failed to encode health response: %v", err)
	}
}
