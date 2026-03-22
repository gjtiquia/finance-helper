package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/gjtiquia/finance-helper/internal/pdf"
)

const pdfChunkClusterTolerance = 2.0

type raw9Chunk struct {
	text string
	x    float64
	y    float64
}

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

		chunks, err := page.GetTextChunks()
		if err != nil {
			return "", err
		}

		if builder.Len() > 0 {
			builder.WriteString("\n\n")
		}

		fmt.Fprintf(&builder, "Page %d", pageNumber)

		lines := renderRaw9Lines(chunks)
		if len(lines) == 0 {
			continue
		}

		builder.WriteByte('\n')
		builder.WriteString(strings.Join(lines, "\n"))
	}

	return builder.String(), nil
}

func renderRaw9Lines(chunks []pdf.TextChunk) []string {
	rawChunks := make([]raw9Chunk, 0, len(chunks))
	for _, chunk := range chunks {
		text := strings.TrimSpace(chunk.S)
		if text == "" {
			continue
		}
		rawChunks = append(rawChunks, raw9Chunk{text: text, x: chunk.X, y: chunk.Y})
	}
	if len(rawChunks) == 0 {
		return nil
	}

	sort.SliceStable(rawChunks, func(i int, j int) bool {
		if delta := rawChunks[i].y - rawChunks[j].y; absFloat(delta) > pdfChunkClusterTolerance {
			return rawChunks[i].y > rawChunks[j].y
		}
		if rawChunks[i].x != rawChunks[j].x {
			return rawChunks[i].x < rawChunks[j].x
		}
		return rawChunks[i].text < rawChunks[j].text
	})

	groups := [][]raw9Chunk{{rawChunks[0]}}
	lineY := []float64{rawChunks[0].y}
	for _, chunk := range rawChunks[1:] {
		placed := false
		for i := range groups {
			if absFloat(chunk.y-lineY[i]) > pdfChunkClusterTolerance {
				continue
			}
			groups[i] = append(groups[i], chunk)
			lineY[i] = averageRaw9Y(groups[i])
			placed = true
			break
		}
		if !placed {
			groups = append(groups, []raw9Chunk{chunk})
			lineY = append(lineY, chunk.y)
		}
	}

	sort.SliceStable(groups, func(i, j int) bool {
		return lineY[i] > lineY[j]
	})

	lines := make([]string, 0, len(groups))
	for _, group := range groups {
		sort.SliceStable(group, func(i int, j int) bool {
			if group[i].x != group[j].x {
				return group[i].x < group[j].x
			}
			return group[i].text < group[j].text
		})

		parts := make([]string, 0, len(group))
		for _, chunk := range group {
			parts = append(parts, chunk.text)
		}
		line := strings.TrimSpace(strings.Join(parts, " "))
		if line == "" {
			continue
		}
		lines = append(lines, line)
	}

	return lines
}

func averageRaw9Y(chunks []raw9Chunk) float64 {
	if len(chunks) == 0 {
		return 0
	}
	total := 0.0
	for _, chunk := range chunks {
		total += chunk.y
	}
	return total / float64(len(chunks))
}
