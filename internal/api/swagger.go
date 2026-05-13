package api

import (
	_ "embed"
	"log"
	"net/http"
)

//go:embed openapi.json
var openapiSpec []byte

func handleOpenAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if _, err := w.Write(openapiSpec); err != nil {
		log.Printf("ERROR writing OpenAPI response: %v", err)
	}
}

//go:embed swagger.html
var swaggerUI []byte

func handleDocs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if _, err := w.Write(swaggerUI); err != nil {
		log.Printf("ERROR writing docs response: %v", err)
	}
}
