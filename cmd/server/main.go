package main

import (
	"fmt"
	"github.com/gjtiquia/finance-helper/internal/api"
	"log"
	"net/http"
	"os"
)

func main() {
	pdfService := newPDFService(newPDFStorage("data/pdf"))

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprintln(w, "TODO : web app home page")
	})
	mux.HandleFunc("GET /api/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprintln(w, "server is running")
	})
	mux.HandleFunc("GET "+api.PDFListPath, pdfListHandler(pdfService))
	mux.HandleFunc("POST "+api.PDFUploadPath, pdfUploadHandler(pdfService))
	mux.HandleFunc("POST "+api.PDFParsePath, pdfParseHandler(pdfService))

	addr := ":" + port
	log.Printf("server listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
