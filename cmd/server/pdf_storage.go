package main

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

var errInvalidPDFPath = errors.New("invalid pdf path")

type pdfStorage struct {
	root string
}

func newPDFStorage(root string) pdfStorage {
	return pdfStorage{root: root}
}

func (s pdfStorage) save(relativePath string, src io.Reader) error {
	fullPath, err := s.fullPath(relativePath)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		return err
	}

	file, err := os.OpenFile(fullPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, src)
	return err
}

func (s pdfStorage) listPaths() ([]string, error) {
	var paths []string

	err := filepath.WalkDir(s.root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(s.root, path)
		if err != nil {
			return err
		}

		paths = append(paths, filepath.ToSlash(relPath))
		return nil
	})
	if errors.Is(err, os.ErrNotExist) {
		return []string{}, nil
	}

	return paths, err
}

func (s pdfStorage) fileSize(relativePath string) (int64, error) {
	fullPath, err := s.fullPath(relativePath)
	if err != nil {
		return 0, err
	}

	info, err := os.Stat(fullPath)
	if err != nil {
		return 0, err
	}

	return info.Size(), nil
}

func (s pdfStorage) fullPath(relativePath string) (string, error) {
	cleanPath, err := cleanPDFPath(relativePath)
	if err != nil {
		return "", err
	}

	return filepath.Join(s.root, filepath.FromSlash(cleanPath)), nil
}

func cleanPDFPath(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", errInvalidPDFPath
	}

	path = filepath.ToSlash(path)
	if strings.HasPrefix(path, "/") {
		return "", errInvalidPDFPath
	}

	cleanPath := filepath.ToSlash(filepath.Clean(path))
	if cleanPath == "." || cleanPath == "" {
		return "", errInvalidPDFPath
	}

	if cleanPath == ".." || strings.HasPrefix(cleanPath, "../") {
		return "", errInvalidPDFPath
	}

	if strings.ToLower(filepath.Ext(cleanPath)) != ".pdf" {
		return "", fmt.Errorf("pdf path must end with .pdf")
	}

	return cleanPath, nil
}
