package main

import (
	"bytes"
	"github.com/gjtiquia/finance-helper/internal/api"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPDFUploadHandler(t *testing.T) {
	tempDir := t.TempDir()
	service := newPDFService(newPDFStorage(tempDir))

	requestBody := &bytes.Buffer{}
	writer := multipart.NewWriter(requestBody)

	part, err := writer.CreateFormFile(api.PDFFormFile, "statement.pdf")
	if err != nil {
		t.Fatalf("CreateFormFile returned error: %v", err)
	}

	if _, err := io.WriteString(part, "%PDF-1.4\nstatement\n"); err != nil {
		t.Fatalf("WriteString returned error: %v", err)
	}

	if err := writer.WriteField(api.PDFFormPath, "statements/chase/test.pdf"); err != nil {
		t.Fatalf("WriteField returned error: %v", err)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, api.PDFUploadPath, requestBody)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()

	pdfUploadHandler(service).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "Uploaded:") {
		t.Fatalf("body missing upload confirmation: %q", body)
	}

	if !strings.Contains(body, "statements/chase/test.pdf") {
		t.Fatalf("body missing uploaded path: %q", body)
	}

	storedPath := filepath.Join(tempDir, "statements", "chase", "test.pdf")
	if _, err := os.Stat(storedPath); err != nil {
		t.Fatalf("stored file missing: %v", err)
	}
}

func TestPDFUploadHandlerRejectsNonPDF(t *testing.T) {
	tempDir := t.TempDir()
	service := newPDFService(newPDFStorage(tempDir))

	requestBody := &bytes.Buffer{}
	writer := multipart.NewWriter(requestBody)

	part, err := writer.CreateFormFile(api.PDFFormFile, "statement.txt")
	if err != nil {
		t.Fatalf("CreateFormFile returned error: %v", err)
	}

	if _, err := io.WriteString(part, "not a pdf"); err != nil {
		t.Fatalf("WriteString returned error: %v", err)
	}

	if err := writer.WriteField(api.PDFFormPath, "statements/chase/test.pdf"); err != nil {
		t.Fatalf("WriteField returned error: %v", err)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, api.PDFUploadPath, requestBody)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()

	pdfUploadHandler(service).ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestPDFListHandlerRecursive(t *testing.T) {
	tempDir := t.TempDir()
	service := newPDFService(newPDFStorage(tempDir))

	createDummyFile(t, tempDir, "z-last.pdf", "%PDF-1.4\nlast\n")
	createDummyFile(t, tempDir, "statements/chase/a.pdf", "%PDF-1.4\na\n")
	createDummyFile(t, tempDir, "statements/amex/b.pdf", "%PDF-1.4\nb\n")

	req := httptest.NewRequest(http.MethodGet, api.PDFListPath, nil)
	rec := httptest.NewRecorder()

	pdfListHandler(service).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	want := []string{
		"statements/amex/b.pdf",
		"statements/chase/a.pdf",
		"z-last.pdf",
	}

	got := splitNonEmptyLines(rec.Body.String())
	if len(got) != len(want) {
		t.Fatalf("line count = %d, want %d, body = %q", len(got), len(want), rec.Body.String())
	}

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("line %d = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestPDFParseHandlerMissingFile(t *testing.T) {
	tempDir := t.TempDir()
	service := newPDFService(newPDFStorage(tempDir))

	form := url.Values{
		api.PDFFormParser: []string{api.PDFParserRaw},
		api.PDFFormPath:   []string{"statements/chase/missing.pdf"},
	}
	req := httptest.NewRequest(http.MethodPost, api.PDFParsePath, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	pdfParseHandler(service).ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestPDFParseHandlerUnknownParser(t *testing.T) {
	tempDir := t.TempDir()
	service := newPDFService(newPDFStorage(tempDir))
	createDummyFile(t, tempDir, "statements/chase/test.pdf", "%PDF-1.4\nhello\n")

	form := url.Values{
		api.PDFFormParser: []string{"unknown"},
		api.PDFFormPath:   []string{"statements/chase/test.pdf"},
	}
	req := httptest.NewRequest(http.MethodPost, api.PDFParsePath, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	pdfParseHandler(service).ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func splitNonEmptyLines(value string) []string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}

	return strings.Split(trimmed, "\n")
}

func createDummyFile(t *testing.T, root string, relativePath string, contents string) {
	t.Helper()

	fullPath := filepath.Join(root, filepath.FromSlash(relativePath))
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}

	if err := os.WriteFile(fullPath, []byte(contents), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
}
