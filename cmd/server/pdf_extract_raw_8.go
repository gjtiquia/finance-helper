package main

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/gjtiquia/finance-helper/internal/pdf"
)

func extractRaw8PlainTextFromPDF(path string) (string, error) {
	return extractWithGhostscriptFallback(path, extractRaw8PlainTextDirect)
}

func extractRaw8PlainTextDirect(path string) (string, error) {
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

		baseLines, err := raw4LegacyPageLines(page)
		if err != nil {
			return "", err
		}
		spacingText, err := raw3PageSpacingText(page)
		if err != nil {
			return "", err
		}

		if builder.Len() > 0 {
			builder.WriteString("\n\n")
		}

		fmt.Fprintf(&builder, "Page %d", pageNumber)
		if len(baseLines) == 0 {
			continue
		}

		builder.WriteByte('\n')
		builder.WriteString(mergeSpacingIntoBaseLines(baseLines, spacingText))
	}

	return builder.String(), nil
}

func raw4LegacyPageLines(page pdf.Page) ([]string, error) {
	texts, err := pageContentTexts(page)
	if err != nil {
		return nil, err
	}

	return mergeStyledTextsLegacy(texts), nil
}

func raw3PageSpacingText(page pdf.Page) (string, error) {
	rows, err := page.GetTextByRow()
	if err != nil {
		return "", err
	}

	parts := make([]string, 0)
	for _, row := range rows {
		if row == nil {
			continue
		}

		for _, text := range row.Content {
			value := strings.TrimSpace(text.S)
			if value == "" {
				continue
			}
			parts = append(parts, value)
		}
	}

	return strings.Join(parts, " "), nil
}

func mergeSpacingIntoBaseLines(baseLines []string, spacingText string) string {
	base := strings.Join(baseLines, "\n")
	baseRunes := []rune(base)
	spacingRunes := []rune(spacingText)

	var builder strings.Builder
	spacingIndex := 0

	for _, baseRune := range baseRunes {
		if baseRune == '\n' {
			builder.WriteRune(baseRune)
			continue
		}

		if unicode.IsSpace(baseRune) {
			continue
		}

		insertSpace, nextIndex := shouldInsertSpacingBefore(spacingRunes, spacingIndex, baseRune)
		if insertSpace && !endsWithWhitespaceRune(&builder) {
			builder.WriteByte(' ')
		}
		if nextIndex > spacingIndex {
			spacingIndex = nextIndex
		}

		builder.WriteRune(baseRune)
	}

	return builder.String()
}

func shouldInsertSpacingBefore(spacingRunes []rune, start int, target rune) (bool, int) {
	insertSpace := false
	for i := start; i < len(spacingRunes); i++ {
		current := spacingRunes[i]
		if current == target {
			return insertSpace, i + 1
		}
		if current == ' ' {
			insertSpace = true
			continue
		}
	}

	return false, start
}

func endsWithWhitespaceRune(builder *strings.Builder) bool {
	value := builder.String()
	if value == "" {
		return true
	}

	runes := []rune(value)
	return unicode.IsSpace(runes[len(runes)-1])
}
