package main

import (
	"fmt"
	"strings"

	"github.com/ledongthuc/pdf"
)

func extractRaw3PlainTextFromPDF(path string) (string, error) {
	return extractWithGhostscriptFallback(path, extractRaw3PlainTextDirect)
}

func extractRaw3PlainTextDirect(path string) (string, error) {
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

		rows, err := page.GetTextByRow()
		if err != nil {
			return "", err
		}
		if len(rows) == 0 {
			continue
		}

		builder.WriteByte('\n')
		for i, row := range rows {
			if i > 0 {
				builder.WriteByte('\n')
			}

			for j, text := range row.Content {
				if j > 0 {
					builder.WriteByte(' ')
				}
				builder.WriteString(strings.TrimSpace(text.S))
			}
		}
	}

	return builder.String(), nil
}
