package main

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/gjtiquia/finance-helper/internal/api"
	"html/template"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const serverURLCookieName = "finance_helper_server_url"

type webApp struct {
	tmpl *template.Template
}

type pageData struct {
	SuggestedServerURL string
	ConnectedServerURL string
}

type panelData struct {
	Title string
	Body  string
	Error bool
}

func newWebApp() (*webApp, error) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		return nil, fmt.Errorf("could not resolve template path")
	}

	templateDir := filepath.Join(filepath.Dir(currentFile), "templates", "*.html")
	tmpl, err := template.ParseGlob(templateDir)
	if err != nil {
		return nil, fmt.Errorf("could not load web templates")
	}

	return &webApp{tmpl: tmpl}, nil
}

func (a *webApp) homeHandler(w http.ResponseWriter, r *http.Request) {
	serverURL, _ := configuredServerURLFromCookie(r)
	data := pageData{
		SuggestedServerURL: requestBaseURL(r),
		ConnectedServerURL: serverURL,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := a.tmpl.ExecuteTemplate(w, "index", data); err != nil {
		http.Error(w, "Could not render page", http.StatusInternalServerError)
	}
}

func (a *webApp) statusHandler(w http.ResponseWriter, r *http.Request) {
	serverURL, err := configuredServerURLFromCookie(r)
	if err != nil {
		a.renderPanel(w, panelData{
			Title: "Status",
			Body:  "Config: not found\nRun: Connect to a server URL",
			Error: true,
		})
		return
	}

	body := fmt.Sprintf("Config: browser session\nServer URL: %s\n", serverURL)
	if err := pingServer(serverURL); err != nil {
		body += "Server: offline"
		a.renderPanel(w, panelData{Title: "Status", Body: body, Error: true})
		return
	}

	body += "Server: online"
	a.renderPanel(w, panelData{Title: "Status", Body: body})
}

func (a *webApp) connectHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		a.renderPanel(w, panelData{Title: "Connect", Body: "Could not read form", Error: true})
		return
	}

	serverURL, err := normalizeURL(r.FormValue("server_url"))
	if err != nil {
		a.renderPanel(w, panelData{Title: "Connect", Body: err.Error(), Error: true})
		return
	}

	if err := pingServer(serverURL); err != nil {
		a.renderPanel(w, panelData{Title: "Connect", Body: fmt.Sprintf("No server is running at %s. Ensure the server is running first.", serverURL), Error: true})
		return
	}

	setConfiguredServerURLCookie(w, serverURL)
	a.renderPanel(w, panelData{Title: "Connect", Body: fmt.Sprintf("Connected to %s", serverURL)})
}

func (a *webApp) pdfListHandler(w http.ResponseWriter, r *http.Request) {
	body, err := getTextForRequest(r, api.PDFListPath)
	if err != nil {
		a.renderPanel(w, panelData{Title: "PDF List", Body: err.Error(), Error: true})
		return
	}

	if strings.TrimSpace(body) == "" {
		body = "No PDFs found"
	}

	a.renderPanel(w, panelData{Title: "PDF List", Body: body})
}

func (a *webApp) pdfUploadHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		a.renderPanel(w, panelData{Title: "PDF Upload", Body: "Could not read upload form", Error: true})
		return
	}

	file, header, err := r.FormFile(api.PDFFormFile)
	if err != nil {
		a.renderPanel(w, panelData{Title: "PDF Upload", Body: "Missing PDF file", Error: true})
		return
	}
	defer file.Close()

	serverPath := r.FormValue(api.PDFFormPath)
	body, err := postMultipartTextForRequest(r, api.PDFUploadPath, func(writer *multipart.Writer) error {
		part, err := writer.CreateFormFile(api.PDFFormFile, header.Filename)
		if err != nil {
			return fmt.Errorf("Could not create upload request")
		}

		if _, err := io.Copy(part, file); err != nil {
			return fmt.Errorf("Could not read local PDF")
		}

		if err := writer.WriteField(api.PDFFormPath, serverPath); err != nil {
			return fmt.Errorf("Could not create upload request")
		}

		return nil
	})
	if err != nil {
		a.renderPanel(w, panelData{Title: "PDF Upload", Body: err.Error(), Error: true})
		return
	}

	a.renderPanel(w, panelData{Title: "PDF Upload", Body: body})
}

func (a *webApp) pdfParseHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		a.renderPanel(w, panelData{Title: "PDF Parse", Body: "Could not read parse form", Error: true})
		return
	}

	body, err := postFormTextForRequest(r, api.PDFParsePath, url.Values{
		api.PDFFormParser: []string{r.FormValue(api.PDFFormParser)},
		api.PDFFormPath:   []string{r.FormValue(api.PDFFormPath)},
	})
	if err != nil {
		a.renderPanel(w, panelData{Title: "PDF Parse", Body: err.Error(), Error: true})
		return
	}

	a.renderPanel(w, panelData{Title: "PDF Parse", Body: body})
}

func (a *webApp) renderPanel(w http.ResponseWriter, data panelData) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := a.tmpl.ExecuteTemplate(w, "panel", data); err != nil {
		http.Error(w, "Could not render response", http.StatusInternalServerError)
	}
}

func requestBaseURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}

	return scheme + "://" + r.Host
}

func setConfiguredServerURLCookie(w http.ResponseWriter, serverURL string) {
	http.SetCookie(w, &http.Cookie{
		Name:     serverURLCookieName,
		Value:    url.QueryEscape(serverURL),
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   30 * 24 * 60 * 60,
	})
}

func configuredServerURLFromCookie(r *http.Request) (string, error) {
	cookie, err := r.Cookie(serverURLCookieName)
	if err != nil {
		if errors.Is(err, http.ErrNoCookie) {
			return "", fmt.Errorf("No server configured. Connect first.")
		}

		return "", fmt.Errorf("Could not read server config")
	}

	serverURL, err := url.QueryUnescape(cookie.Value)
	if err != nil {
		return "", fmt.Errorf("Could not read server config")
	}

	if strings.TrimSpace(serverURL) == "" {
		return "", fmt.Errorf("No server configured. Connect first.")
	}

	return serverURL, nil
}

func getTextForRequest(r *http.Request, apiPath string) (string, error) {
	serverURL, err := configuredServerURLFromCookie(r)
	if err != nil {
		return "", err
	}

	resp, err := webHTTPClient().Get(serverURL + apiPath)
	if err != nil {
		return "", fmt.Errorf("Could not reach server")
	}
	defer resp.Body.Close()

	return readTextResponse(resp)
}

func postFormTextForRequest(r *http.Request, apiPath string, form url.Values) (string, error) {
	serverURL, err := configuredServerURLFromCookie(r)
	if err != nil {
		return "", err
	}

	resp, err := webHTTPClient().Post(serverURL+apiPath, "application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("Could not reach server")
	}
	defer resp.Body.Close()

	return readTextResponse(resp)
}

func postMultipartTextForRequest(r *http.Request, apiPath string, build func(*multipart.Writer) error) (string, error) {
	serverURL, err := configuredServerURLFromCookie(r)
	if err != nil {
		return "", err
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := build(writer); err != nil {
		return "", err
	}

	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("Could not create upload request")
	}

	resp, err := webHTTPClient().Post(serverURL+apiPath, writer.FormDataContentType(), &body)
	if err != nil {
		return "", fmt.Errorf("Could not reach server")
	}
	defer resp.Body.Close()

	return readTextResponse(resp)
}

func normalizeURL(rawURL string) (string, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return "", fmt.Errorf("Invalid URL: %s", rawURL)
	}

	if isPort(rawURL) {
		rawURL = "http://localhost:" + rawURL
	} else if strings.HasPrefix(rawURL, "localhost:") {
		rawURL = "http://" + rawURL
	}

	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("Invalid URL: %s", rawURL)
	}

	if !parsedURL.IsAbs() || parsedURL.Host == "" {
		return "", fmt.Errorf("Invalid URL: %s", rawURL)
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return "", fmt.Errorf("Invalid URL: %s", rawURL)
	}

	return strings.TrimRight(parsedURL.String(), "/"), nil
}

func isPort(value string) bool {
	port, err := strconv.Atoi(value)
	if err != nil {
		return false
	}

	return port >= 1 && port <= 65535
}

func pingServer(serverURL string) error {
	response, err := getTextAtServerURL(serverURL, "/api/")
	if err != nil {
		return err
	}

	if strings.TrimSpace(response) != "server is running" {
		return fmt.Errorf("unexpected response body")
	}

	return nil
}

func getTextAtServerURL(serverURL string, apiPath string) (string, error) {
	resp, err := webHTTPClient().Get(serverURL + apiPath)
	if err != nil {
		return "", fmt.Errorf("Could not reach server")
	}
	defer resp.Body.Close()

	return readTextResponse(resp)
}

func readTextResponse(resp *http.Response) (string, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("Could not read server response")
	}

	text := string(body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%s", strings.TrimSpace(text))
	}

	return text, nil
}

func webHTTPClient() *http.Client {
	return &http.Client{Timeout: 30 * time.Second}
}
