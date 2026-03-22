package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/gjtiquia/finance-helper/internal/pdf"
)

const (
	pdfTightLineMergeTolerance = 1.5
)

func extractRaw6PlainTextFromPDF(path string) (string, error) {
	return extractWithGhostscriptFallback(path, extractRaw6PlainTextDirect)
}

func extractRaw6PlainTextDirect(path string) (string, error) {
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

		texts, err := pageContentTexts(page)
		if err != nil {
			return "", err
		}

		if builder.Len() > 0 {
			builder.WriteString("\n\n")
		}

		fmt.Fprintf(&builder, "Page %d", pageNumber)

		lines := renderRaw6Lines(texts)
		if len(lines) == 0 {
			continue
		}

		builder.WriteByte('\n')
		builder.WriteString(strings.Join(lines, "\n"))
	}

	return builder.String(), nil
}

func renderRaw6Lines(texts []extractedPDFText) []string {
	runs := sequentialTextRuns(texts)
	if len(runs) == 0 {
		return nil
	}

	lineGroups := make([][]extractedTextRun, 0)
	currentLine := []extractedTextRun{runs[0]}
	currentY := runs[0].y

	for _, run := range runs[1:] {
		if absFloat(run.y-currentY) <= pdfTightLineMergeTolerance {
			currentLine = append(currentLine, run)
			currentY = averageLineY(currentLine)
			continue
		}

		lineGroups = append(lineGroups, currentLine)
		currentLine = []extractedTextRun{run}
		currentY = run.y
	}

	lineGroups = append(lineGroups, currentLine)

	lines := make([]string, 0, len(lineGroups))
	for _, group := range lineGroups {
		sort.SliceStable(group, func(i int, j int) bool {
			if group[i].x != group[j].x {
				return group[i].x < group[j].x
			}
			return group[i].text < group[j].text
		})

		line := strings.TrimSpace(renderLine(group))
		if line == "" {
			continue
		}
		lines = append(lines, line)
	}

	return lines
}

func sequentialTextRuns(texts []extractedPDFText) []extractedTextRun {
	runs := make([]extractedTextRun, 0, len(texts))
	for _, text := range texts {
		if strings.TrimSpace(text.Text) == "" {
			continue
		}

		runs = append(runs, extractedTextRun{
			text:     text.Text,
			font:     text.Font,
			x:        text.X,
			y:        text.Y,
			width:    text.Width,
			fontSize: text.FontSize,
		})
	}

	return runs
}

func averageLineY(runs []extractedTextRun) float64 {
	if len(runs) == 0 {
		return 0
	}

	total := 0.0
	for _, run := range runs {
		total += run.y
	}

	return total / float64(len(runs))
}
