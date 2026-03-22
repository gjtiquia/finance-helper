package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

func connect(w io.Writer, rawURL string) error {
	serverURL, err := normalizeURL(rawURL)
	if err != nil {
		return err
	}

	if err := pingServer(serverURL); err != nil {
		return fmt.Errorf("No server is running at %s. Ensure the server is running first.", serverURL)
	}

	if err := saveConfig(config{Server: serverURL}); err != nil {
		return err
	}

	fmt.Fprintf(w, "Connected to %s\n", serverURL)
	return nil
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
	client := http.Client{Timeout: 5 * time.Second}

	resp, err := client.Get(serverURL + "/api/")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if strings.TrimSpace(string(body)) != "server is running" {
		return fmt.Errorf("unexpected response body")
	}

	return nil
}
