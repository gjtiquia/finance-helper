package main

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func pdfCommand(w io.Writer, args []string) error {
	if len(args) == 0 {
		printPDFHelp(w)
		return nil
	}

	switch args[0] {
	case "--help", "-h":
		printPDFHelp(w)
		return nil
	case "upload":
		if len(args) != 3 {
			fmt.Fprintln(w, "Usage: finance-helper pdf upload <local-pdf-path> <server-relative-path>")
			return nil
		}
		return pdfUpload(w, args[1], args[2])
	case "list":
		if len(args) != 1 {
			fmt.Fprintln(w, "Usage: finance-helper pdf list")
			return nil
		}
		return pdfList(w)
	case "parse":
		if len(args) != 3 {
			fmt.Fprintln(w, "Usage: finance-helper pdf parse <parser-name> <server-relative-path>")
			return nil
		}
		return pdfParse(w, args[1], args[2])
	default:
		printPDFHelp(w)
		return nil
	}
}

func printPDFHelp(w io.Writer) {
	fmt.Fprintln(w, "finance-helper pdf")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  finance-helper pdf upload <local-pdf-path> <server-relative-path>")
	fmt.Fprintln(w, "  finance-helper pdf list")
	fmt.Fprintln(w, "  finance-helper pdf parse <parser-name> <server-relative-path>")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Examples:")
	fmt.Fprintln(w, "  finance-helper pdf upload ./statement.pdf statements/chase/2026-03.pdf")
	fmt.Fprintln(w, "  finance-helper pdf list")
	fmt.Fprintln(w, "  finance-helper pdf parse raw statements/chase/2026-03.pdf")
}

func pdfUpload(w io.Writer, localPath string, serverPath string) error {
	if strings.ToLower(filepath.Ext(localPath)) != ".pdf" {
		return fmt.Errorf("Local file must be a PDF")
	}

	serverURL, err := configuredServerURL()
	if err != nil {
		return err
	}

	file, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("Could not open local PDF")
	}
	defer file.Close()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", filepath.Base(localPath))
	if err != nil {
		return fmt.Errorf("Could not create upload request")
	}

	if _, err := io.Copy(part, file); err != nil {
		return fmt.Errorf("Could not read local PDF")
	}

	if err := writer.WriteField("path", serverPath); err != nil {
		return fmt.Errorf("Could not create upload request")
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("Could not create upload request")
	}

	resp, err := httpClient().Post(serverURL+"/api/v1/pdf/upload", writer.FormDataContentType(), &body)
	if err != nil {
		return fmt.Errorf("Could not reach server")
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("Could not read server response")
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s", strings.TrimSpace(string(responseBody)))
	}

	fmt.Fprint(w, string(responseBody))
	return nil
}

func pdfList(w io.Writer) error {
	serverURL, err := configuredServerURL()
	if err != nil {
		return err
	}

	resp, err := httpClient().Get(serverURL + "/api/v1/pdf")
	if err != nil {
		return fmt.Errorf("Could not reach server")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("Could not read server response")
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s", strings.TrimSpace(string(body)))
	}

	fmt.Fprint(w, string(body))
	return nil
}

func pdfParse(w io.Writer, parserName string, serverPath string) error {
	if parserName != "raw" {
		return fmt.Errorf("Unknown parser: %s", parserName)
	}

	serverURL, err := configuredServerURL()
	if err != nil {
		return err
	}

	form := strings.NewReader(url.Values{
		"parser": []string{parserName},
		"path":   []string{serverPath},
	}.Encode())
	req, err := http.NewRequest(http.MethodPost, serverURL+"/api/v1/pdf/parse", form)
	if err != nil {
		return fmt.Errorf("Could not create parse request")
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := httpClient().Do(req)
	if err != nil {
		return fmt.Errorf("Could not reach server")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("Could not read server response")
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s", strings.TrimSpace(string(body)))
	}

	fmt.Fprint(w, string(body))
	return nil
}

func httpClient() *http.Client {
	return &http.Client{Timeout: 30 * time.Second}
}
