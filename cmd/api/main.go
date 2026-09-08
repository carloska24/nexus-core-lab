package main

import (
	"log"
	"net/http"

	"github.com/carloska24/nexus-core-lab/internal/platform/httpserver"
)

func main() {
	handler := httpserver.New()

	log.Println("NEXUS Core Lab API listening on :8080")

	if err := http.ListenAndServe(":8080", handler); err != nil {
		log.Fatal(err)
	}
}
