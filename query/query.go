package query

import (
	"fmt"

	"github.com/dmavrotas/pitwall/nlp"
	"github.com/dmavrotas/pitwall/store"
)

// Engine ties together the NLP parser and SQLite store. Questions are answered
// by the deterministic rule engine, completed where needed by the offline gonlu
// layer inside the parser; there is no network or LLM involvement.
type Engine struct {
	db     *store.DB
	parser *nlp.Parser
}

// NewEngine creates a new query engine.
func NewEngine(db *store.DB, parser *nlp.Parser) *Engine {
	return &Engine{db: db, parser: parser}
}

// Result holds the structured output of a query.
type Result struct {
	Description string
	Columns     []string
	Rows        [][]string
	Source      string // "rules" or "rules+gonlu" — useful for UI and debugging
}

// Ask processes a natural language question and returns structured results.
func (e *Engine) Ask(question string) (*Result, error) {
	plan, err := e.parser.Parse(question)
	if err != nil {
		return nil, err
	}
	cols, rows, err := e.db.Query(plan.SQL, plan.Args)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	return &Result{
		Description: plan.Description,
		Columns:     cols,
		Rows:        rows,
		Source:      source(plan),
	}, nil
}

// source names the path that produced a plan.
func source(plan *nlp.QueryPlan) string {
	if plan.Learned {
		return "rules+gonlu"
	}
	return "rules"
}
