package main

import "testing"

func TestNormalizeURL(t *testing.T) {
	tests := []struct {
		name    string
		rawURL  string
		want    string
		wantErr bool
	}{
		{name: "port only", rawURL: "3000", want: "http://localhost:3000"},
		{name: "localhost without scheme", rawURL: "localhost:3000", want: "http://localhost:3000"},
		{name: "http localhost with trailing slash", rawURL: "http://localhost:3000/", want: "http://localhost:3000"},
		{name: "https url", rawURL: "https://example.com", want: "https://example.com"},
		{name: "empty", rawURL: "", wantErr: true},
		{name: "invalid scheme", rawURL: "ftp://example.com", wantErr: true},
		{name: "malformed url", rawURL: "://example.com", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeURL(tt.rawURL)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("normalizeURL(%q) expected error", tt.rawURL)
				}
				return
			}

			if err != nil {
				t.Fatalf("normalizeURL(%q) returned error: %v", tt.rawURL, err)
			}

			if got != tt.want {
				t.Fatalf("normalizeURL(%q) = %q, want %q", tt.rawURL, got, tt.want)
			}
		})
	}
}
