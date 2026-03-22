package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestStatusWithoutConfig(t *testing.T) {
	tempDir := t.TempDir()
	originalUserConfigDir := userConfigDir
	userConfigDir = func() (string, error) { return tempDir, nil }
	t.Cleanup(func() {
		userConfigDir = originalUserConfigDir
	})

	var output bytes.Buffer
	if err := status(&output); err != nil {
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
	tempDir := t.TempDir()
	originalUserConfigDir := userConfigDir
	userConfigDir = func() (string, error) { return tempDir, nil }
	t.Cleanup(func() {
		userConfigDir = originalUserConfigDir
	})

	want := config{Server: "http://localhost:3000"}
	if err := saveConfig(want); err != nil {
		t.Fatalf("saveConfig returned error: %v", err)
	}

	got, path, err := loadConfig()
	if err != nil {
		t.Fatalf("loadConfig returned error: %v", err)
	}

	if got != want {
		t.Fatalf("loadConfig returned %+v, want %+v", got, want)
	}

	if !strings.HasSuffix(path, "finance-helper/config.json") {
		t.Fatalf("loadConfig returned unexpected path: %q", path)
	}
}
