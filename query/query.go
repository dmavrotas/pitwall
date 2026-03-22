package query

import (
	"fmt"

	"github.com/dmavrotas/pitwall/nlp"
	"github.com/dmavrotas/pitwall/store"
)

// Engine ties together the NLP parser and SQLite store.
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
	}, nil
}
