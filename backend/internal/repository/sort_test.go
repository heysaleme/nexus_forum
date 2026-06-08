package repository

import (
	"strings"
	"testing"
)

func TestParsePostSort_HotNewTop(t *testing.T) {
	tests := []struct {
		sortSpec string
		dialect  string
		wantSub  string
	}{
		{"hot", "sqlite", "julianday"},
		{"hot", "postgres", "EXTRACT(EPOCH"},
		{"new", "sqlite", "created_at DESC"},
		{"top", "sqlite", "score DESC"},
		{"-score", "sqlite", "score ASC"},
	}

	for _, tc := range tests {
		got := parsePostSort(tc.dialect, tc.sortSpec)
		if !strings.Contains(got, tc.wantSub) {
			t.Errorf("parsePostSort(%q, %q) = %q, want substring %q", tc.sortSpec, tc.dialect, got, tc.wantSub)
		}
	}
}
