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
	webApp, err := newWebApp()
	if err != nil {
		log.Fatal(err)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /", webApp.homeHandler)
	mux.HandleFunc("GET /ui/parser-builder", webApp.parserBuilderHandler)
	mux.HandleFunc("GET /ui/status", webApp.statusHandler)
	mux.HandleFunc("POST /ui/connect", webApp.connectHandler)
	mux.HandleFunc("GET /ui/pdf/list", webApp.pdfListHandler)
	mux.HandleFunc("GET /ui/pdf/preview", webApp.pdfPreviewHandler)
	mux.HandleFunc("POST /ui/pdf/upload", webApp.pdfUploadHandler)
	mux.HandleFunc("POST /ui/pdf/parse", webApp.pdfParseHandler)
	mux.HandleFunc("GET /api/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/" {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprintln(w, "server is running")
	})
	mux.HandleFunc("GET "+api.PDFListPath, pdfListHandler(pdfService))
	mux.HandleFunc("GET "+api.PDFFilePath, pdfFileHandler(pdfService))
	mux.HandleFunc("POST "+api.PDFUploadPath, pdfUploadHandler(pdfService))
	mux.HandleFunc("POST "+api.PDFParsePath, pdfParseHandler(pdfService))

	addr := ":" + port
	log.Printf("server listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
