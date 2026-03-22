package main

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
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
	x        float64
	y        float64
	width    float64
	fontSize float64
}

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
	FontSize float64 `json:"-"`
}

func extractPlainTextFromPDF(path string) (string, error) {
	document, err := extractPDFDocument(path)
	if err != nil {
		return "", err
	}

	return renderPlainText(document), nil
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
	document, err := extractPDFDocumentDirect(path)
	if err == nil {
		return document, nil
	}

	repairedPath, repairErr := rewritePDFWithGhostscript(path)
	if repairErr != nil {
		return extractedPDFDocument{}, fmt.Errorf("%v (ghostscript fallback failed: %v)", err, repairErr)
	}
	defer os.RemoveAll(filepath.Dir(repairedPath))

	document, repairedErr := extractPDFDocumentDirect(repairedPath)
	if repairedErr != nil {
		return extractedPDFDocument{}, fmt.Errorf("%v (ghostscript retry failed: %v)", err, repairedErr)
	}

	return document, nil
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
	runs := sortedTextRuns(texts)
	items := make([]extractedPDFText, 0, len(runs))
	for _, run := range runs {
		items = append(items, extractedPDFText{
			Text:     run.text,
			X:        run.x,
			Y:        run.y,
			Width:    run.width,
			FontSize: run.fontSize,
		})
	}

	return items
}

func sortedTextRuns(texts []pdf.Text) []extractedTextRun {
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

func renderPlainText(document extractedPDFDocument) string {
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
	if len(texts) == 0 {
		return nil
	}

	runs := make([]extractedTextRun, 0, len(texts))
	for _, text := range texts {
		runs = append(runs, extractedTextRun{
			text:     text.Text,
			x:        text.X,
			y:        text.Y,
			width:    text.Width,
			fontSize: text.FontSize,
		})
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
