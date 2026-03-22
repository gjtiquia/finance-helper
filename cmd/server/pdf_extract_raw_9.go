package main

import (
	"fmt"
	"strings"

	"github.com/gjtiquia/finance-helper/internal/pdf"
)

const pdfRowTolerance = 2.0

func extractRaw9PlainTextFromPDF(path string) (string, error) {
	return extractWithGhostscriptFallback(path, extractRaw9PlainTextDirect)
}

func extractRaw9PlainTextDirect(path string) (string, error) {
	file, reader, err := pdf.Open(path)
	if err != nil {
		return "", err
	}
	defer closePDFFile(file)

	var builder strings.Builder
	for pageNumber := 1; pageNumber <= reader.NumPage(); pageNumber++ {
		page := reader.Page(pageNumber)
		if page.V.IsNull() {
			continue
		}

		rows, err := page.GetTextByRowTolerance(pdfRowTolerance)
		if err != nil {
			return "", err
		}

		if builder.Len() > 0 {
			builder.WriteString("\n\n")
		}

		fmt.Fprintf(&builder, "Page %d", pageNumber)
		if len(rows) == 0 {
			continue
		}

		builder.WriteByte('\n')
		for i, row := range rows {
			if i > 0 {
				builder.WriteByte('\n')
			}

			wroteText := false
			for _, text := range row.Content {
				value := strings.TrimSpace(text.S)
				if value == "" {
					continue
				}
				if wroteText {
					builder.WriteByte(' ')
				}
				builder.WriteString(value)
				wroteText = true
			}
		}
	}

	return builder.String(), nil
}
