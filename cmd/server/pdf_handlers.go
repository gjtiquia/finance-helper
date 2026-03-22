package main

import (
	"errors"
	"fmt"
	"github.com/gjtiquia/finance-helper/internal/api"
	"net/http"
	"os"
	"strings"
)

func pdfUploadHandler(service pdfService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(32 << 20); err != nil {
			http.Error(w, "Could not read upload form", http.StatusBadRequest)
			return
		}

		file, header, err := r.FormFile(api.PDFFormFile)
		if err != nil {
			http.Error(w, "Missing PDF file", http.StatusBadRequest)
			return
		}
		defer file.Close()

		storedPath, err := service.upload(header.Filename, r.FormValue(api.PDFFormPath), file)
		if err != nil {
			switch {
			case errors.Is(err, errInvalidPDFPath):
				http.Error(w, "Invalid server PDF path", http.StatusBadRequest)
			case errors.Is(err, os.ErrExist):
				http.Error(w, "PDF already exists at that path", http.StatusConflict)
			case strings.Contains(err.Error(), "must end with .pdf"):
				http.Error(w, "Invalid server PDF path", http.StatusBadRequest)
			case strings.Contains(err.Error(), "Local file must be a PDF"):
				http.Error(w, err.Error(), http.StatusBadRequest)
			default:
				http.Error(w, "Could not save PDF", http.StatusInternalServerError)
			}
			return
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprintf(w, "Uploaded: %s\n", storedPath)
	}
}

func pdfListHandler(service pdfService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		paths, err := service.list()
		if err != nil {
			http.Error(w, "Could not list PDFs", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		if len(paths) == 0 {
			return
		}

		fmt.Fprintln(w, strings.Join(paths, "\n"))
	}
}

func pdfParseHandler(service pdfService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Could not read parse request", http.StatusBadRequest)
			return
		}

		result, err := service.parse(r.FormValue(api.PDFFormParser), r.FormValue(api.PDFFormPath))
		if err != nil {
			switch {
			case errors.Is(err, errInvalidPDFPath):
				http.Error(w, "Invalid server PDF path", http.StatusBadRequest)
			case errors.Is(err, os.ErrNotExist):
				http.Error(w, "PDF not found", http.StatusNotFound)
			case strings.Contains(err.Error(), "Unknown parser"):
				http.Error(w, err.Error(), http.StatusBadRequest)
			case strings.Contains(err.Error(), "must end with .pdf"):
				http.Error(w, "Invalid server PDF path", http.StatusBadRequest)
			default:
				http.Error(w, "Could not read PDF", http.StatusInternalServerError)
			}
			return
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprintln(w, result)
	}
}
