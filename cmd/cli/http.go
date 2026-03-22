package main

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func getText(apiPath string) (string, error) {
	serverURL, err := configuredServerURL()
	if err != nil {
		return "", err
	}

	resp, err := cliHTTPClient().Get(serverURL + apiPath)
	if err != nil {
		return "", fmt.Errorf("Could not reach server")
	}
	defer resp.Body.Close()

	return readTextResponse(resp)
}

func postFormText(apiPath string, form url.Values) (string, error) {
	serverURL, err := configuredServerURL()
	if err != nil {
		return "", err
	}

	resp, err := cliHTTPClient().Post(serverURL+apiPath, "application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("Could not reach server")
	}
	defer resp.Body.Close()

	return readTextResponse(resp)
}

func postMultipartText(apiPath string, build func(*multipart.Writer) error) (string, error) {
	serverURL, err := configuredServerURL()
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

	resp, err := cliHTTPClient().Post(serverURL+apiPath, writer.FormDataContentType(), &body)
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

func cliHTTPClient() *http.Client {
	return &http.Client{Timeout: 30 * time.Second}
}
