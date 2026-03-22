package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestStatusWithoutConfig(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	var output bytes.Buffer
	if err := statusAtPath(&output, configPath); err != nil {
		t.Fatalf("status returned error: %v", err)
	}

	got := output.String()
	if !strings.Contains(got, "Config: not found") {
		t.Fatalf("status output missing config message: %q", got)
	}

	if !strings.Contains(got, "Run: finance-helper connect <url>") {
		t.Fatalf("status output missing connect guidance: %q", got)
	}
}

func TestSaveAndLoadConfig(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "finance-helper", "config.json")
	want := config{Server: "http://localhost:3000"}
	if err := saveConfigAtPath(configPath, want); err != nil {
		t.Fatalf("saveConfigAtPath returned error: %v", err)
	}

	got, err := loadConfigAtPath(configPath)
	if err != nil {
		t.Fatalf("loadConfigAtPath returned error: %v", err)
	}

	if got != want {
		t.Fatalf("loadConfigAtPath returned %+v, want %+v", got, want)
	}
}
