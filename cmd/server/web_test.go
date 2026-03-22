package main

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gjtiquia/finance-helper/internal/api"
)

func TestStatusWithoutConnect(t *testing.T) {
	app, err := newWebApp()
	if err != nil {
		t.Fatalf("newWebApp returned error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/ui/status", nil)
	rec := httptest.NewRecorder()

	app.statusHandler(rec, req)

	body := rec.Body.String()
	if !strings.Contains(body, "Config: not found") {
		t.Fatalf("status body missing config message: %q", body)
	}
}

func TestConnectAndStatus(t *testing.T) {
	downstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/api/" {
			io.WriteString(w, "server is running\n")
			return
		}

		http.NotFound(w, r)
	}))
	defer downstream.Close()

	app, err := newWebApp()
	if err != nil {
		t.Fatalf("newWebApp returned error: %v", err)
	}

	connectForm := url.Values{"server_url": []string{downstream.URL}}
	connectReq := httptest.NewRequest(http.MethodPost, "/ui/connect", strings.NewReader(connectForm.Encode()))
	connectReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	connectRec := httptest.NewRecorder()

	app.connectHandler(connectRec, connectReq)

	if !strings.Contains(connectRec.Body.String(), "Connected to "+downstream.URL) {
		t.Fatalf("connect response missing success message: %q", connectRec.Body.String())
	}

	cookie := connectRec.Result().Cookies()[0]
	statusReq := httptest.NewRequest(http.MethodGet, "/ui/status", nil)
	statusReq.AddCookie(cookie)
	statusRec := httptest.NewRecorder()

	app.statusHandler(statusRec, statusReq)

	statusBody := statusRec.Body.String()
	if !strings.Contains(statusBody, "Server URL: "+downstream.URL) {
		t.Fatalf("status body missing server url: %q", statusBody)
	}
	if !strings.Contains(statusBody, "Server: online") {
		t.Fatalf("status body missing online state: %q", statusBody)
	}
}

func TestPDFListParseAndUpload(t *testing.T) {
	downstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == api.PDFListPath:
			io.WriteString(w, "statements/chase/2026-03.pdf\n")
		case r.Method == http.MethodPost && r.URL.Path == api.PDFParsePath:
			if err := r.ParseForm(); err != nil {
				http.Error(w, "bad parse form", http.StatusBadRequest)
				return
			}
			if r.FormValue(api.PDFFormParser) != api.PDFParserRaw {
				http.Error(w, "Unknown parser", http.StatusBadRequest)
				return
			}
			io.WriteString(w, "parsed text")
		case r.Method == http.MethodPost && r.URL.Path == api.PDFUploadPath:
			if err := r.ParseMultipartForm(32 << 20); err != nil {
				http.Error(w, "bad upload form", http.StatusBadRequest)
				return
			}
			if r.FormValue(api.PDFFormPath) == "" {
				http.Error(w, "missing path", http.StatusBadRequest)
				return
			}
			file, _, err := r.FormFile(api.PDFFormFile)
			if err != nil {
				http.Error(w, "missing file", http.StatusBadRequest)
				return
			}
			file.Close()
			io.WriteString(w, "Uploaded: statements/chase/2026-03.pdf\n")
		default:
			http.NotFound(w, r)
		}
	}))
	defer downstream.Close()

	app, err := newWebApp()
	if err != nil {
		t.Fatalf("newWebApp returned error: %v", err)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/ui/pdf/list", nil)
	listReq.AddCookie(&http.Cookie{Name: serverURLCookieName, Value: url.QueryEscape(downstream.URL)})
	listRec := httptest.NewRecorder()
	app.pdfListHandler(listRec, listReq)
	if !strings.Contains(listRec.Body.String(), "statements/chase/2026-03.pdf") {
		t.Fatalf("list response missing path: %q", listRec.Body.String())
	}

	parseForm := url.Values{
		api.PDFFormParser: []string{api.PDFParserRaw},
		api.PDFFormPath:   []string{"statements/chase/2026-03.pdf"},
	}
	parseReq := httptest.NewRequest(http.MethodPost, "/ui/pdf/parse", strings.NewReader(parseForm.Encode()))
	parseReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	parseReq.AddCookie(&http.Cookie{Name: serverURLCookieName, Value: url.QueryEscape(downstream.URL)})
	parseRec := httptest.NewRecorder()
	app.pdfParseHandler(parseRec, parseReq)
	if !strings.Contains(parseRec.Body.String(), "parsed text") {
		t.Fatalf("parse response missing result: %q", parseRec.Body.String())
	}

	uploadBody := &bytes.Buffer{}
	uploadWriter := multipart.NewWriter(uploadBody)
	part, err := uploadWriter.CreateFormFile(api.PDFFormFile, "statement.pdf")
	if err != nil {
		t.Fatalf("CreateFormFile returned error: %v", err)
	}
	if _, err := io.WriteString(part, "%PDF-1.4\nexample\n"); err != nil {
		t.Fatalf("WriteString returned error: %v", err)
	}
	if err := uploadWriter.WriteField(api.PDFFormPath, "statements/chase/2026-03.pdf"); err != nil {
		t.Fatalf("WriteField returned error: %v", err)
	}
	if err := uploadWriter.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}

	uploadReq := httptest.NewRequest(http.MethodPost, "/ui/pdf/upload", uploadBody)
	uploadReq.Header.Set("Content-Type", uploadWriter.FormDataContentType())
	uploadReq.AddCookie(&http.Cookie{Name: serverURLCookieName, Value: url.QueryEscape(downstream.URL)})
	uploadRec := httptest.NewRecorder()
	app.pdfUploadHandler(uploadRec, uploadReq)
	if !strings.Contains(uploadRec.Body.String(), "Uploaded: statements/chase/2026-03.pdf") {
		t.Fatalf("upload response missing confirmation: %q", uploadRec.Body.String())
	}
}
