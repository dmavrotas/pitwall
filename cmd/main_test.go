package main

import (
	"strings"
	"testing"
)

func TestParseArgs(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		wantDir   string
		wantPlain bool
	}{
		{"defaults", nil, "data", false},
		{"data dir only", []string{"mydata/"}, "mydata/", false},
		{"plain then dir", []string{"--plain", "data/"}, "data/", true},
		{"dir then plain", []string{"data/", "--plain"}, "data/", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseArgs(tt.args)
			if err != nil {
				t.Fatal(err)
			}
			if got.dataDir != tt.wantDir || got.plain != tt.wantPlain {
				t.Errorf("got %+v, want dir=%q plain=%v", got, tt.wantDir, tt.wantPlain)
			}
		})
	}
}

// Pitwall no longer has an LLM fallback. The old flags must fail loudly instead
// of being swallowed as a data directory called "--ai".
func TestParseArgsRejectsUnknownAndRemovedFlags(t *testing.T) {
	for _, args := range [][]string{
		{"--ai"},
		{"--ai", "data/"},
		{"--ai-model", "claude-haiku-5-5"},
		{"--bogus"},
		{"-x"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			_, err := parseArgs(args)
			if err == nil {
				t.Fatalf("parseArgs(%v) succeeded, want an error", args)
			}
			if !strings.Contains(err.Error(), "unknown flag") {
				t.Errorf("error %q should say 'unknown flag'", err)
			}
		})
	}
}
