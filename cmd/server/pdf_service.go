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
	if !isSupportedPDFParser(parserName) {
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
	case api.PDFParserRaw1:
		result, err := extractRaw1PlainTextFromPDF(fullPath)
		if err != nil {
			return "", err
		}
		return result, nil
	case api.PDFParserRaw2:
		result, err := extractRaw2PlainTextFromPDF(fullPath)
		if err != nil {
			return "", err
		}
		return result, nil
	case api.PDFParserRaw3:
		result, err := extractRaw3PlainTextFromPDF(fullPath)
		if err != nil {
			return "", err
		}
		return result, nil
	case api.PDFParserRaw4:
		result, err := extractRaw4PlainTextFromPDF(fullPath)
		if err != nil {
			return "", err
		}
		return result, nil
	case api.PDFParserRaw5:
		result, err := extractRaw5PlainTextFromPDF(fullPath)
		if err != nil {
			return "", err
		}
		return result, nil
	case api.PDFParserRaw6:
		result, err := extractRaw6PlainTextFromPDF(fullPath)
		if err != nil {
			return "", err
		}
		return result, nil
	case api.PDFParserRaw7:
		result, err := extractRaw7PlainTextFromPDF(fullPath)
		if err != nil {
			return "", err
		}
		return result, nil
	case api.PDFParserRaw8:
		result, err := extractRaw8PlainTextFromPDF(fullPath)
		if err != nil {
			return "", err
		}
		return result, nil
	case api.PDFParserRaw9:
		result, err := extractRaw9PlainTextFromPDF(fullPath)
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

func isSupportedPDFParser(parserName string) bool {
	switch parserName {
	case api.PDFParserRaw, api.PDFParserRaw1, api.PDFParserRaw2, api.PDFParserRaw3, api.PDFParserRaw4, api.PDFParserRaw5, api.PDFParserRaw6, api.PDFParserRaw7, api.PDFParserRaw8, api.PDFParserRaw9, api.PDFParserRawJSON:
		return true
	default:
		return false
	}
}
