package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ledongthuc/pdf"
)

type extractedPDFDocument struct {
	PageCount int                `json:"page_count"`
	Pages     []extractedPDFPage `json:"pages"`
}

type extractedPDFPage struct {
	Page     int                `json:"page"`
	MediaBox []float64          `json:"media_box,omitempty"`
	CropBox  []float64          `json:"crop_box,omitempty"`
	Text     []extractedPDFText `json:"text"`
}

type extractedPDFText struct {
	Text     string  `json:"text"`
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
	Width    float64 `json:"width"`
	Font     string  `json:"-"`
	FontSize float64 `json:"-"`
}

func extractPlainTextFromPDF(path string) (string, error) {
	return extractRaw1PlainTextFromPDF(path)
}

func extractRawJSONFromPDF(path string) (string, error) {
	document, err := extractPDFDocument(path)
	if err != nil {
		return "", err
	}

	formatted, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return "", err
	}

	return string(formatted), nil
}

func extractPDFDocument(path string) (extractedPDFDocument, error) {
	return extractWithGhostscriptFallback(path, extractPDFDocumentDirect)
}

func extractPDFDocumentDirect(path string) (extractedPDFDocument, error) {
	file, reader, err := pdf.Open(path)
	if err != nil {
		return extractedPDFDocument{}, err
	}
	defer closePDFFile(file)

	document := extractedPDFDocument{
		PageCount: reader.NumPage(),
		Pages:     make([]extractedPDFPage, 0, reader.NumPage()),
	}

	for pageNumber := 1; pageNumber <= reader.NumPage(); pageNumber++ {
		page := reader.Page(pageNumber)
		if page.V.IsNull() {
			continue
		}

		content, err := pageContent(page)
		if err != nil {
			return extractedPDFDocument{}, err
		}

		mediaBox := pageBox(page, "MediaBox")
		cropBox := pageBox(page, "CropBox")
		if len(cropBox) == 0 && len(mediaBox) > 0 {
			cropBox = append([]float64(nil), mediaBox...)
		}

		document.Pages = append(document.Pages, extractedPDFPage{
			Page:     pageNumber,
			MediaBox: mediaBox,
			CropBox:  cropBox,
			Text:     extractPageTextItems(content.Text),
		})
	}

	return document, nil
}

func extractWithGhostscriptFallback[T any](path string, extractor func(string) (T, error)) (T, error) {
	result, err := extractor(path)
	if err == nil {
		return result, nil
	}

	repairedPath, repairErr := rewritePDFWithGhostscript(path)
	if repairErr != nil {
		var zero T
		return zero, fmt.Errorf("%v (ghostscript fallback failed: %v)", err, repairErr)
	}
	defer os.RemoveAll(filepath.Dir(repairedPath))

	result, repairedErr := extractor(repairedPath)
	if repairedErr != nil {
		var zero T
		return zero, fmt.Errorf("%v (ghostscript retry failed: %v)", err, repairedErr)
	}

	return result, nil
}

func closePDFFile(file io.Closer) {
	if file != nil {
		_ = file.Close()
	}
}

func rewritePDFWithGhostscript(path string) (string, error) {
	gsPath, err := exec.LookPath("gs")
	if err != nil {
		return "", err
	}

	tempDir, err := os.MkdirTemp("", "finance-helper-pdf-*")
	if err != nil {
		return "", err
	}

	repairedPath := filepath.Join(tempDir, "repaired.pdf")
	cmd := exec.Command(gsPath, "-o", repairedPath, "-sDEVICE=pdfwrite", "-dPDFSETTINGS=/prepress", path)
	output, err := cmd.CombinedOutput()
	if err != nil {
		os.RemoveAll(tempDir)
		return "", fmt.Errorf("ghostscript rewrite failed: %w: %s", err, strings.TrimSpace(string(output)))
	}

	return repairedPath, nil
}

func pageContent(page pdf.Page) (content pdf.Content, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("pdf extraction panicked: %v", recovered)
		}
	}()

	return page.Content(), nil
}

func pageBox(page pdf.Page, key string) []float64 {
	for value := page.V; !value.IsNull(); value = value.Key("Parent") {
		box := value.Key(key)
		if box.IsNull() || box.Len() != 4 {
			continue
		}

		return []float64{
			box.Index(0).Float64(),
			box.Index(1).Float64(),
			box.Index(2).Float64(),
			box.Index(3).Float64(),
		}
	}

	return nil
}

func extractPageTextItems(texts []pdf.Text) []extractedPDFText {
	items := make([]extractedPDFText, 0, len(texts))
	for _, text := range texts {
		value := strings.TrimSpace(text.S)
		if value == "" {
			continue
		}

		items = append(items, extractedPDFText{
			Text:     value,
			X:        text.X,
			Y:        text.Y,
			Width:    text.W,
			Font:     text.Font,
			FontSize: text.FontSize,
		})
	}

	return items
}
