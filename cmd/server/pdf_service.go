package main

import (
	"errors"
	"fmt"
	"github.com/gjtiquia/finance-helper/internal/api"
	"io"
	"os"
	"sort"
	"strings"
)

type pdfService struct {
	storage pdfStorage
}

func newPDFService(storage pdfStorage) pdfService {
	return pdfService{storage: storage}
}

func (s pdfService) upload(localFilename string, relativePath string, src io.Reader) (string, error) {
	if !strings.HasSuffix(strings.ToLower(localFilename), ".pdf") {
		return "", fmt.Errorf("Local file must be a PDF")
	}

	cleanPath, err := cleanPDFPath(relativePath)
	if err != nil {
		return "", err
	}

	if err := s.storage.save(cleanPath, src); err != nil {
		return "", err
	}

	return cleanPath, nil
}

func (s pdfService) list() ([]string, error) {
	paths, err := s.storage.listPaths()
	if err != nil {
		return nil, err
	}

	sort.Strings(paths)
	return paths, nil
}

func (s pdfService) parse(parserName string, relativePath string) (string, error) {
	if parserName != api.PDFParserRaw && parserName != api.PDFParserRawJSON {
		return "", fmt.Errorf("Unknown parser: %s", parserName)
	}

	cleanPath, err := cleanPDFPath(relativePath)
	if err != nil {
		return "", err
	}

	fullPath, err := s.storage.fullPath(cleanPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", err
		}

		return "", err
	}

	if _, err := os.Stat(fullPath); err != nil {
		return "", err
	}

	switch parserName {
	case api.PDFParserRaw:
		result, err := extractPlainTextFromPDF(fullPath)
		if err != nil {
			return "", err
		}
		return result, nil
	case api.PDFParserRawJSON:
		result, err := extractRawJSONFromPDF(fullPath)
		if err != nil {
			return "", err
		}
		return result, nil
	}

	return "", fmt.Errorf("Unknown parser: %s", parserName)
}

func (s pdfService) pdfFullPath(relativePath string) (string, error) {
	cleanPath, err := cleanPDFPath(relativePath)
	if err != nil {
		return "", err
	}

	fullPath, err := s.storage.fullPath(cleanPath)
	if err != nil {
		return "", err
	}

	if _, err := os.Stat(fullPath); err != nil {
		return "", err
	}

	return fullPath, nil
}
