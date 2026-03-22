package main

import (
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"rsc.io/pdf"
)

const pdfLineTolerance = 3.0

type extractedTextRun struct {
	text     string
	x        float64
	y        float64
	width    float64
	fontSize float64
}

func extractPlainTextFromPDF(path string) (string, error) {
	result, err := extractPlainText(path)
	if err == nil {
		return result, nil
	}

	repairedPath, repairErr := rewritePDFWithGhostscript(path)
	if repairErr != nil {
		return "", fmt.Errorf("%v (ghostscript fallback failed: %v)", err, repairErr)
	}
	defer os.RemoveAll(filepath.Dir(repairedPath))

	return extractPlainText(repairedPath)
}

func extractPlainText(path string) (string, error) {
	reader, err := pdf.Open(path)
	if err != nil {
		return "", err
	}

	var builder strings.Builder
	totalPages := reader.NumPage()
	for pageNumber := 1; pageNumber <= totalPages; pageNumber++ {
		page := reader.Page(pageNumber)
		if page.V.IsNull() {
			continue
		}

		if builder.Len() > 0 {
			builder.WriteString("\n\n")
		}

		fmt.Fprintf(&builder, "Page %d", pageNumber)

		content, err := pageContent(page)
		if err != nil {
			return "", err
		}

		lines := pageLines(content.Text)
		if len(lines) == 0 {
			continue
		}

		builder.WriteByte('\n')
		builder.WriteString(strings.Join(lines, "\n"))
	}

	return builder.String(), nil
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

func pageLines(texts []pdf.Text) []string {
	runs := make([]extractedTextRun, 0, len(texts))
	for _, text := range texts {
		value := strings.TrimSpace(text.S)
		if value == "" {
			continue
		}

		runs = append(runs, extractedTextRun{
			text:     value,
			x:        text.X,
			y:        text.Y,
			width:    text.W,
			fontSize: text.FontSize,
		})
	}

	if len(runs) == 0 {
		return nil
	}

	sort.SliceStable(runs, func(i int, j int) bool {
		if delta := runs[i].y - runs[j].y; math.Abs(delta) > pdfLineTolerance {
			return runs[i].y > runs[j].y
		}

		if runs[i].x != runs[j].x {
			return runs[i].x < runs[j].x
		}

		return runs[i].text < runs[j].text
	})

	var lines []string
	currentLine := []extractedTextRun{runs[0]}
	currentY := runs[0].y

	for _, run := range runs[1:] {
		if math.Abs(run.y-currentY) <= pdfLineTolerance {
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
	threshold := math.Max(1, math.Min(previous.fontSize, current.fontSize)*0.25)
	return gap > threshold
}
