// Package ai provides an optional LLM-backed fallback for natural-language queries
// the rule-based parser cannot handle. It is disabled by default — enabled only
// when the engine is constructed with a non-nil Translator.
package ai

import (
	"context"
	"errors"
)

// ErrUnanswerable is returned when the translator decides the question cannot
// be expressed as a SELECT against the known schema.
var ErrUnanswerable = errors.New("ai: question cannot be answered against the known schema")

// Result is the structured output of a successful translation.
type Result struct {
	SQL         string
	Description string
}

// Translator turns a natural-language question into a SQL SELECT plan.
// Implementations should return ErrUnanswerable when the question is out of scope
// rather than fabricating SQL.
type Translator interface {
	Translate(ctx context.Context, question string) (Result, error)
}
