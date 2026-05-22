package query

import (
	"context"
	"fmt"

	"github.com/dmavrotas/pitwall/ai"
	"github.com/dmavrotas/pitwall/nlp"
	"github.com/dmavrotas/pitwall/store"
)

// Engine ties together the NLP parser and SQLite store. An optional ai.Translator
// is consulted when the rule-based parser cannot produce a query.
type Engine struct {
	db         *store.DB
	parser     *nlp.Parser
	translator ai.Translator
}

// NewEngine creates a new query engine without AI fallback.
func NewEngine(db *store.DB, parser *nlp.Parser) *Engine {
	return &Engine{db: db, parser: parser}
}

// NewEngineWithAI creates a query engine that falls back to the given translator
// when the rule-based parser fails. Pass nil to disable the fallback.
func NewEngineWithAI(db *store.DB, parser *nlp.Parser, translator ai.Translator) *Engine {
	return &Engine{db: db, parser: parser, translator: translator}
}

// Result holds the structured output of a query.
type Result struct {
	Description string
	Columns     []string
	Rows        [][]string
	Source      string // "rules" or "ai" — useful for UI and debugging
}

// Ask processes a natural language question and returns structured results.
func (e *Engine) Ask(question string) (*Result, error) {
	plan, parseErr := e.parser.Parse(question)
	if parseErr == nil {
		cols, rows, err := e.db.Query(plan.SQL, plan.Args)
		if err != nil {
			return nil, fmt.Errorf("query failed: %w", err)
		}
		return &Result{
			Description: plan.Description,
			Columns:     cols,
			Rows:        rows,
			Source:      "rules",
		}, nil
	}

	// Rule-based parser couldn't handle it. Try the AI fallback if configured.
	if e.translator != nil {
		aiRes, aiErr := e.translator.Translate(context.Background(), question)
		if aiErr == nil {
			cols, rows, err := e.db.Query(aiRes.SQL, nil)
			if err != nil {
				return nil, fmt.Errorf("ai-generated query failed: %w", err)
			}
			return &Result{
				Description: aiRes.Description,
				Columns:     cols,
				Rows:        rows,
				Source:      "ai",
			}, nil
		}
	}

	return nil, parseErr
}
