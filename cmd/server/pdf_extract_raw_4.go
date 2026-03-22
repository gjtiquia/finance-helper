package main

import (
	"fmt"
	"strings"

	"github.com/ledongthuc/pdf"
)

func extractRaw4PlainTextFromPDF(path string) (string, error) {
	return extractWithGhostscriptFallback(path, extractRaw4PlainTextDirect)
}

func extractRaw4PlainTextDirect(path string) (string, error) {
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
		merged := mergeStyledRuns(texts)

		if builder.Len() > 0 {
			builder.WriteString("\n\n")
		}

		fmt.Fprintf(&builder, "Page %d", pageNumber)
		if len(merged) == 0 {
			continue
		}

		builder.WriteByte('\n')
		builder.WriteString(strings.Join(renderRunLines(merged), "\n"))
	}

	return builder.String(), nil
}

func mergeStyledRuns(texts []extractedPDFText) []extractedTextRun {
	runs := sortedTextRuns(texts)
	if len(runs) == 0 {
		return nil
	}

	merged := make([]extractedTextRun, 0, len(runs))
	current := runs[0]
	for _, next := range runs[1:] {
		if sameStyledText(current, next) {
			if shouldInsertSpace(current, next) {
				current.text += " "
			}
			current.text += next.text
			current.width = (next.x + next.width) - current.x
			continue
		}

		merged = append(merged, current)
		current = next
	}

	merged = append(merged, current)
	return merged
}

func renderRunLines(runs []extractedTextRun) []string {
	if len(runs) == 0 {
		return nil
	}

	var lines []string
	currentLine := []extractedTextRun{runs[0]}
	currentY := runs[0].y

	for _, run := range runs[1:] {
		if sameVisualLine(currentY, run.y) {
			currentLine = append(currentLine, run)
			continue
		}

		lines = append(lines, renderLine(currentLine))
		currentLine = []extractedTextRun{run}
		currentY = run.y
	}

	lines = append(lines, renderLine(currentLine))
	return lines
}

func sameStyledText(previous extractedTextRun, current extractedTextRun) bool {
	return pdf.IsSameSentence(toPDFText(previous), toPDFText(current))
}

func sameVisualLine(previousY float64, currentY float64) bool {
	return absFloat(previousY-currentY) <= pdfLineMergeTolerance
}

func toPDFText(run extractedTextRun) pdf.Text {
	return pdf.Text{
		Font:     run.font,
		FontSize: run.fontSize,
		X:        run.x,
		Y:        run.y,
		W:        run.width,
		S:        run.text,
	}
}

func absFloat(value float64) float64 {
	if value < 0 {
		return -value
	}

	return value
}
