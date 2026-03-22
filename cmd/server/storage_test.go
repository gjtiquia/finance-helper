package main

import "testing"

func TestCleanPDFPath(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "nested path", input: "statements/chase/2026-03.pdf", want: "statements/chase/2026-03.pdf"},
		{name: "clean dot segment", input: "statements/../statements/chase.pdf", want: "statements/chase.pdf"},
		{name: "absolute path", input: "/tmp/file.pdf", wantErr: true},
		{name: "parent traversal", input: "../file.pdf", wantErr: true},
		{name: "missing extension", input: "file.txt", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := cleanPDFPath(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("cleanPDFPath(%q) expected error", tt.input)
				}
				return
			}

			if err != nil {
				t.Fatalf("cleanPDFPath(%q) returned error: %v", tt.input, err)
			}

			if got != tt.want {
				t.Fatalf("cleanPDFPath(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
