package main

import (
	"fmt"
	"github.com/gjtiquia/finance-helper/internal/api"
	"io"
	"mime/multipart"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

func pdfCommand(w io.Writer, args []string) error {
	if len(args) == 0 {
		printPDFHelp(w)
		return nil
	}

	switch args[0] {
	case "--help", "-h":
		printPDFHelp(w)
		return nil
	case "upload":
		if len(args) != 3 {
			fmt.Fprintln(w, "Usage: finance-helper pdf upload <local-pdf-path> <server-relative-path>")
			return nil
		}
		return pdfUpload(w, args[1], args[2])
	case "list":
		if len(args) != 1 {
			fmt.Fprintln(w, "Usage: finance-helper pdf list")
			return nil
		}
		return pdfList(w)
	case "parse":
		if len(args) != 3 {
			fmt.Fprintln(w, "Usage: finance-helper pdf parse <parser-name> <server-relative-path>")
			return nil
		}
		return pdfParse(w, args[1], args[2])
	default:
		printPDFHelp(w)
		return nil
	}
}

func printPDFHelp(w io.Writer) {
	fmt.Fprintln(w, "finance-helper pdf")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  finance-helper pdf upload <local-pdf-path> <server-relative-path>")
	fmt.Fprintln(w, "  finance-helper pdf list")
	fmt.Fprintln(w, "  finance-helper pdf parse <parser-name> <server-relative-path>")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Examples:")
	fmt.Fprintln(w, "  finance-helper pdf upload ./statement.pdf statements/chase/2026-03.pdf")
	fmt.Fprintln(w, "  finance-helper pdf list")
	fmt.Fprintln(w, "  finance-helper pdf parse raw statements/chase/2026-03.pdf")
	fmt.Fprintln(w, "  finance-helper pdf parse raw-1 statements/chase/2026-03.pdf")
	fmt.Fprintln(w, "  finance-helper pdf parse raw-2 statements/chase/2026-03.pdf")
	fmt.Fprintln(w, "  finance-helper pdf parse raw-3 statements/chase/2026-03.pdf")
	fmt.Fprintln(w, "  finance-helper pdf parse raw-4 statements/chase/2026-03.pdf")
	fmt.Fprintln(w, "  finance-helper pdf parse raw-5 statements/chase/2026-03.pdf")
	fmt.Fprintln(w, "  finance-helper pdf parse raw-json statements/chase/2026-03.pdf")
}

func pdfUpload(w io.Writer, localPath string, serverPath string) error {
	if strings.ToLower(filepath.Ext(localPath)) != ".pdf" {
		return fmt.Errorf("Local file must be a PDF")
	}

	file, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("Could not open local PDF")
	}
	defer file.Close()

	responseBody, err := postMultipartText(api.PDFUploadPath, func(writer *multipart.Writer) error {
		part, err := writer.CreateFormFile(api.PDFFormFile, filepath.Base(localPath))
		if err != nil {
			return fmt.Errorf("Could not create upload request")
		}

		if _, err := io.Copy(part, file); err != nil {
			return fmt.Errorf("Could not read local PDF")
		}

		if err := writer.WriteField(api.PDFFormPath, serverPath); err != nil {
			return fmt.Errorf("Could not create upload request")
		}

		return nil
	})
	if err != nil {
		return err
	}

	fmt.Fprint(w, responseBody)
	return nil
}

func pdfList(w io.Writer) error {
	body, err := getText(api.PDFListPath)
	if err != nil {
		return err
	}

	fmt.Fprint(w, body)
	return nil
}

func pdfParse(w io.Writer, parserName string, serverPath string) error {
	if !isSupportedPDFParser(parserName) {
		return fmt.Errorf("Unknown parser: %s", parserName)
	}

	body, err := postFormText(api.PDFParsePath, url.Values{
		api.PDFFormParser: []string{parserName},
		api.PDFFormPath:   []string{serverPath},
	})
	if err != nil {
		return err
	}

	fmt.Fprint(w, body)
	return nil
}

func isSupportedPDFParser(parserName string) bool {
	switch parserName {
	case api.PDFParserRaw, api.PDFParserRaw1, api.PDFParserRaw2, api.PDFParserRaw3, api.PDFParserRaw4, api.PDFParserRaw5, api.PDFParserRawJSON:
		return true
	default:
		return false
	}
}
