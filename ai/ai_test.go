package ai

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
)

func TestValidateSQLAcceptsSelect(t *testing.T) {
	cases := []string{
		"SELECT 1",
		"SELECT id FROM drivers WHERE id = 1",
		"SELECT d.surname FROM drivers d JOIN results r ON r.driver_id = d.id",
		"WITH all_points AS (SELECT driver_id, points FROM results UNION ALL SELECT driver_id, points FROM sprint_results) SELECT driver_id, SUM(points) FROM all_points GROUP BY driver_id",
		"SELECT 1;", // trailing semicolon is stripped
	}
	for _, sql := range cases {
		t.Run(sql, func(t *testing.T) {
			out, err := ValidateSQL(sql)
			if err != nil {
				t.Fatalf("ValidateSQL(%q) failed: %v", sql, err)
			}
			if strings.Contains(out, ";") {
				t.Errorf("output should be free of semicolons, got %q", out)
			}
		})
	}
}

func TestValidateSQLRejectsForbidden(t *testing.T) {
	cases := []string{
		"INSERT INTO drivers VALUES (1)",
		"UPDATE results SET points = 0",
		"DELETE FROM races",
		"DROP TABLE drivers",
		"PRAGMA table_info(drivers)",
		"SELECT 1; DROP TABLE drivers",
		"SELECT * FROM secret_table",
		"",
		"-- just a comment",
	}
	for _, sql := range cases {
		t.Run(sql, func(t *testing.T) {
			if _, err := ValidateSQL(sql); err == nil {
				t.Errorf("ValidateSQL(%q) should have failed", sql)
			}
		})
	}
}

func TestParseLLMOutputHappyPath(t *testing.T) {
	raw := "SELECT 1\n\nSimple sanity check"
	res, err := parseLLMOutput(raw)
	if err != nil {
		t.Fatalf("parseLLMOutput failed: %v", err)
	}
	if res.SQL != "SELECT 1" {
		t.Errorf("SQL = %q, want %q", res.SQL, "SELECT 1")
	}
	if res.Description != "Simple sanity check" {
		t.Errorf("Description = %q", res.Description)
	}
}

func TestParseLLMOutputStripsMarkdownFences(t *testing.T) {
	raw := "```sql\nSELECT 1\n```\n\ndesc"
	res, err := parseLLMOutput(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(res.SQL, "SELECT") {
		t.Errorf("expected SQL after stripping fences, got %q", res.SQL)
	}
}

func TestParseLLMOutputUnanswerable(t *testing.T) {
	if _, err := parseLLMOutput("UNANSWERABLE"); !errors.Is(err, ErrUnanswerable) {
		t.Errorf("expected ErrUnanswerable, got %v", err)
	}
}

func TestParseLLMOutputRejectsUnsafeSQL(t *testing.T) {
	if _, err := parseLLMOutput("DROP TABLE drivers\n\nbad"); err == nil {
		t.Error("expected error for DROP statement, got nil")
	}
}

// stubTranslator is a minimal Translator for cache testing.
type stubTranslator struct {
	calls atomic.Int64
	res   Result
	err   error
}

func (s *stubTranslator) Translate(_ context.Context, _ string) (Result, error) {
	s.calls.Add(1)
	return s.res, s.err
}

func TestCacheDeduplicatesIdenticalQuestions(t *testing.T) {
	stub := &stubTranslator{res: Result{SQL: "SELECT 1", Description: "x"}}
	cache := NewCache(stub)

	for i := 0; i < 3; i++ {
		if _, err := cache.Translate(context.Background(), "How many wins?"); err != nil {
			t.Fatalf("Translate failed: %v", err)
		}
	}
	// Whitespace and case should normalize.
	if _, err := cache.Translate(context.Background(), "  HOW many wins?  "); err != nil {
		t.Fatalf("Translate failed: %v", err)
	}
	if got := stub.calls.Load(); got != 1 {
		t.Errorf("inner translator called %d times, want 1", got)
	}
}

func TestCacheDoesNotStoreErrors(t *testing.T) {
	stub := &stubTranslator{err: errors.New("nope")}
	cache := NewCache(stub)

	for i := 0; i < 3; i++ {
		if _, err := cache.Translate(context.Background(), "How many wins?"); err == nil {
			t.Error("expected error to propagate")
		}
	}
	if got := stub.calls.Load(); got != 3 {
		t.Errorf("inner translator called %d times, want 3 (errors should not cache)", got)
	}
}
