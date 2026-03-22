package main

import (
	"errors"
	"fmt"
	"io"
)

func status(w io.Writer) error {
	path, err := configPath()
	if err != nil {
		return err
	}

	return statusAtPath(w, path)
}

func statusAtPath(w io.Writer, path string) error {
	cfg, err := loadConfigAtPath(path)
	if err != nil {
		if errors.Is(err, errConfigNotFound) {
			fmt.Fprintln(w, "Config: not found")
			fmt.Fprintln(w, "Run: finance-helper connect <url>")
			return nil
		}

		return err
	}

	fmt.Fprintf(w, "Config: %s\n", path)
	fmt.Fprintf(w, "Server URL: %s\n", cfg.Server)

	if err := pingServer(cfg.Server); err != nil {
		fmt.Fprintln(w, "Server: offline")
		return nil
	}

	fmt.Fprintln(w, "Server: online")
	return nil
}
