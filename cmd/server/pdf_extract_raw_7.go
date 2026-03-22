package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/gjtiquia/finance-helper/internal/pdf"
)

const pdfChunkLineMergeTolerance = 1.5

type extractedChunk struct {
	text string
	x    float64
	y    float64
}

func extractRaw7PlainTextFromPDF(path string) (string, error) {
	return extractWithGhostscriptFallback(path, extractRaw7PlainTextDirect)
}

func extractRaw7PlainTextDirect(path string) (string, error) {
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

		rows, err := page.GetTextByRow()
		if err != nil {
			return "", err
		}

		if builder.Len() > 0 {
			builder.WriteString("\n\n")
		}

		fmt.Fprintf(&builder, "Page %d", pageNumber)

		lines := clusterChunkRows(rows)
		if len(lines) == 0 {
			continue
		}

		builder.WriteByte('\n')
		builder.WriteString(strings.Join(lines, "\n"))
	}

	return builder.String(), nil
}

func clusterChunkRows(rows pdf.Rows) []string {
	chunks := flattenRowChunks(rows)
	if len(chunks) == 0 {
		return nil
	}

	sort.SliceStable(chunks, func(i int, j int) bool {
		if chunks[i].y != chunks[j].y {
			return chunks[i].y > chunks[j].y
		}
		if chunks[i].x != chunks[j].x {
			return chunks[i].x < chunks[j].x
		}
		return chunks[i].text < chunks[j].text
	})

	lineGroups := make([][]extractedChunk, 0)
	currentLine := []extractedChunk{chunks[0]}
	currentY := chunks[0].y

	for _, chunk := range chunks[1:] {
		if absFloat(chunk.y-currentY) <= pdfChunkLineMergeTolerance {
			currentLine = append(currentLine, chunk)
			currentY = averageChunkY(currentLine)
			continue
		}

		lineGroups = append(lineGroups, currentLine)
		currentLine = []extractedChunk{chunk}
		currentY = chunk.y
	}
	lineGroups = append(lineGroups, currentLine)

	lines := make([]string, 0, len(lineGroups))
	for _, line := range lineGroups {
		sort.SliceStable(line, func(i int, j int) bool {
			if line[i].x != line[j].x {
				return line[i].x < line[j].x
			}
			return line[i].text < line[j].text
		})

		parts := make([]string, 0, len(line))
		for _, chunk := range line {
			value := strings.TrimSpace(chunk.text)
			if value == "" {
				continue
			}
			parts = append(parts, value)
		}
		if len(parts) == 0 {
			continue
		}

		lines = append(lines, strings.Join(parts, " "))
	}

	return lines
}

func flattenRowChunks(rows pdf.Rows) []extractedChunk {
	chunks := make([]extractedChunk, 0)
	for _, row := range rows {
		if row == nil {
			continue
		}

		for _, text := range row.Content {
			value := strings.TrimSpace(text.S)
			if value == "" {
				continue
			}

			chunks = append(chunks, extractedChunk{
				text: value,
				x:    text.X,
				y:    text.Y,
			})
		}
	}

	return chunks
}

func averageChunkY(chunks []extractedChunk) float64 {
	if len(chunks) == 0 {
		return 0
	}

	total := 0.0
	for _, chunk := range chunks {
		total += chunk.y
	}

	return total / float64(len(chunks))
}
