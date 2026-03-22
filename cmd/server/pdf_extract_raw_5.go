package main

import (
	"fmt"
	"strings"

	"github.com/ledongthuc/pdf"
)

func extractRaw5PlainTextFromPDF(path string) (string, error) {
	return extractWithGhostscriptFallback(path, extractRaw5PlainTextDirect)
}

func extractRaw5PlainTextDirect(path string) (string, error) {
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

		if builder.Len() > 0 {
			builder.WriteString("\n\n")
		}

		fmt.Fprintf(&builder, "Page %d", pageNumber)

		pageText, err := pageContentTexts(page)
		if err != nil {
			return "", err
		}

		rows, err := page.GetTextByRow()
		if err != nil {
			return "", err
		}

		lines := renderRaw5Lines(rows, pageText)
		if len(lines) == 0 {
			continue
		}

		builder.WriteByte('\n')
		builder.WriteString(strings.Join(lines, "\n"))
	}

	return builder.String(), nil
}

func renderRaw5Lines(rows pdf.Rows, pageText []extractedPDFText) []string {
	if len(rows) <= 1 {
		return renderRunLines(mergeStyledRuns(pageText))
	}

	lines := make([]string, 0, len(rows))
	for _, row := range rows {
		line := strings.TrimSpace(renderRaw5Row(row))
		if line == "" {
			continue
		}
		lines = append(lines, line)
	}

	if len(lines) == 0 {
		return renderRunLines(mergeStyledRuns(pageText))
	}

	return lines
}

func renderRaw5Row(row *pdf.Row) string {
	if row == nil || len(row.Content) == 0 {
		return ""
	}

	var builder strings.Builder
	for _, text := range row.Content {
		value := strings.TrimSpace(text.S)
		if value == "" {
			continue
		}

		if builder.Len() > 0 {
			builder.WriteByte(' ')
		}
		builder.WriteString(value)
	}

	return builder.String()
}
