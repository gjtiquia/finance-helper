package main

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/ledongthuc/pdf"
)

const (
	pdfLineMergeTolerance = 3.0
	pdfMinimumWordGap     = 1.0
	pdfWordGapFontRatio   = 0.25
)

type extractedTextRun struct {
	text     string
	font     string
	x        float64
	y        float64
	width    float64
	fontSize float64
}

func extractRaw1PlainTextFromPDF(path string) (string, error) {
	document, err := extractPDFDocument(path)
	if err != nil {
		return "", err
	}

	return renderRaw1PlainText(document), nil
}

func renderRaw1PlainText(document extractedPDFDocument) string {
	var builder strings.Builder
	for _, page := range document.Pages {
		if builder.Len() > 0 {
			builder.WriteString("\n\n")
		}

		fmt.Fprintf(&builder, "Page %d", page.Page)

		lines := groupTextIntoLines(page.Text)
		if len(lines) == 0 {
			continue
		}

		builder.WriteByte('\n')
		builder.WriteString(strings.Join(lines, "\n"))
	}

	return builder.String()
}

func groupTextIntoLines(texts []extractedPDFText) []string {
	runs := sortedTextRuns(texts)
	if len(runs) == 0 {
		return nil
	}

	var lines []string
	currentLine := []extractedTextRun{runs[0]}
	currentY := runs[0].y

	for _, run := range runs[1:] {
		if math.Abs(run.y-currentY) <= pdfLineMergeTolerance {
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

func sortedTextRuns(texts []extractedPDFText) []extractedTextRun {
	runs := make([]extractedTextRun, 0, len(texts))
	for _, text := range texts {
		runs = append(runs, extractedTextRun{
			text:     text.Text,
			font:     text.Font,
			x:        text.X,
			y:        text.Y,
			width:    text.Width,
			fontSize: text.FontSize,
		})
	}

	sort.SliceStable(runs, func(i int, j int) bool {
		if delta := runs[i].y - runs[j].y; math.Abs(delta) > pdfLineMergeTolerance {
			return runs[i].y > runs[j].y
		}

		if runs[i].x != runs[j].x {
			return runs[i].x < runs[j].x
		}

		return runs[i].text < runs[j].text
	})

	return runs
}

func renderLine(runs []extractedTextRun) string {
	if len(runs) == 0 {
		return ""
	}

	var builder strings.Builder
	for i, run := range runs {
		if i > 0 && shouldInsertSpace(runs[i-1], run) {
			builder.WriteByte(' ')
		}
		builder.WriteString(run.text)
	}

	return builder.String()
}

func shouldInsertSpace(previous extractedTextRun, current extractedTextRun) bool {
	gap := current.x - (previous.x + previous.width)
	threshold := math.Max(pdfMinimumWordGap, math.Min(previous.fontSize, current.fontSize)*pdfWordGapFontRatio)
	return gap > threshold
}

func pageContentTexts(page pdf.Page) ([]extractedPDFText, error) {
	content, err := pageContent(page)
	if err != nil {
		return nil, err
	}

	return extractPageTextItems(content.Text), nil
}
