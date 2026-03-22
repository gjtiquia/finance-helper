package main

import (
	"bytes"

	"github.com/gjtiquia/finance-helper/internal/pdf"
)

func extractRaw2PlainTextFromPDF(path string) (string, error) {
	return extractWithGhostscriptFallback(path, extractRaw2PlainTextDirect)
}

func extractRaw2PlainTextDirect(path string) (string, error) {
	file, reader, err := pdf.Open(path)
	if err != nil {
		return "", err
	}
	defer closePDFFile(file)

	body, err := reader.GetPlainText()
	if err != nil {
		return "", err
	}

	var buffer bytes.Buffer
	if _, err := buffer.ReadFrom(body); err != nil {
		return "", err
	}

	return buffer.String(), nil
}
