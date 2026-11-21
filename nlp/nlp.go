package nlp

import (
	"strings"
	"unicode"

	"github.com/dmavrotas/pitwall/loader"
)

// QueryPlan is the result of parsing a natural language question.
type QueryPlan struct {
	Intent      Intent
	SQL         string
	Args        map[string]interface{}
	Description string
	// Learned is true when the learned layer (gonlu) filled in an entity or the
	// intent that the rule engine missed, for example a misspelled name.
	Learned bool
}

// Parser translates natural language questions into SQL query plans.
type Parser struct {
	entities *EntityDict
	learned  *learned // nil for a rules-only parser
}

// NewParser creates a parser with entity dictionaries built from the dataset.
// The rule engine stays primary; a small learned layer (gonlu) only fills in
// entities and intents the rules missed, such as misspelled driver names.
func NewParser(ds *loader.Dataset) *Parser {
	p := NewRulesOnlyParser(ds)
	p.learned = newLearned(p.entities)
	return p
}

// NewRulesOnlyParser creates a purely rule-based, fully deterministic parser.
func NewRulesOnlyParser(ds *loader.Dataset) *Parser {
	return &Parser{entities: BuildEntityDict(ds)}
}

// Parse takes a natural language question and returns a QueryPlan.
func (p *Parser) Parse(input string) (*QueryPlan, error) {
	tokens := tokenize(input)
	if len(tokens) == 0 {
		return nil, &ParseError{Input: input, Reason: "empty query"}
	}

	entities := p.entities.ExtractEntities(tokens)
	intent := DetectIntent(tokens)

	// Let the learned layer fill gaps the rules left (typos, missing second entity).
	learnedUsed := p.learned != nil && p.learned.fill(input, &entities, &intent)

	// If we couldn't detect an intent but have entities, try smart defaults
	if intent == IntentUnknown {
		intent = inferIntent(&entities)
	}

	sql, desc, args := BuildQuery(intent, &entities)
	if sql == "" {
		return nil, &ParseError{
			Input:  input,
			Reason: "could not build a query from your question",
		}
	}

	return &QueryPlan{
		Intent:      intent,
		SQL:         sql,
		Args:        args,
		Description: desc,
		Learned:     learnedUsed,
	}, nil
}

// ExampleQuestions returns sample queries the system can handle.
func ExampleQuestions() []string {
	return []string{
		"Who has the most wins?",
		"Who won the most races in 2020?",
		"How many points did Hamilton score in 2019?",
		"Show me the 2021 championship standings",
		"Compare Verstappen vs Hamilton",
		"What are the most common DNF reasons?",
		"Fastest pit stops in 2023",
		"Who got the most pole positions?",
		"Tell me about Monza",
		"Who were Hamilton's teammates?",
		"Ferrari wins in 2004",
		"Season overview 2010",
		"Fastest laps at Silverstone",
		"Podiums in 2022",
	}
}

func tokenize(input string) []string {
	input = strings.ToLower(input)
	// Remove punctuation except hyphens
	cleaned := strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == ' ' || r == '-' {
			return r
		}
		return ' '
	}, input)

	// Expand common contractions/abbreviations
	cleaned = strings.ReplaceAll(cleaned, "who's", "who is")
	cleaned = strings.ReplaceAll(cleaned, "what's", "what is")
	cleaned = strings.ReplaceAll(cleaned, "how's", "how is")

	parts := strings.Fields(cleaned)

	// Remove stop words
	var filtered []string
	for _, p := range parts {
		if !stopWords[p] {
			filtered = append(filtered, p)
		}
	}

	return filtered
}

// inferIntent tries to determine intent from entities alone when keywords fail.
func inferIntent(e *Entities) Intent {
	if e.DriverID > 0 && e.DriverID2 > 0 {
		return IntentHeadToHead
	}
	if e.ConstructorID > 0 && e.ConstructorID2 > 0 {
		return IntentHeadToHead
	}
	if e.DriverID > 0 {
		return IntentDriverInfo
	}
	if e.ConstructorID > 0 {
		return IntentConstructorInfo
	}
	if e.CircuitID > 0 {
		return IntentCircuitInfo
	}
	if e.Year > 0 {
		return IntentSeasonOverview
	}
	return IntentUnknown
}

// ParseError describes why a query could not be understood.
type ParseError struct {
	Input  string
	Reason string
}

func (e *ParseError) Error() string {
	return "could not understand: " + e.Reason
}

// stopWords are dropped before intent and entity detection.
var stopWords = map[string]bool{
	"the": true, "a": true, "an": true, "is": true, "are": true,
	"was": true, "were": true, "in": true, "at": true, "of": true,
	"for": true, "to": true, "and": true, "or": true, "with": true,
	"has": true, "had": true, "have": true, "do": true, "does": true,
	"did": true, "be": true, "been": true, "being": true,
	"show": true, "me": true, "tell": true, "give": true, "get": true,
	"many": true, "much": true, "most": true, "what": true,
	"who": true, "which": true, "how": true, "when": true, "where": true,
	"i": true, "my": true, "about": true, "from": true, "by": true,
	"on": true, "it": true, "its": true, "that": true, "this": true,
	"all": true, "any": true, "some": true, "their": true, "his": true,
	"her": true, "can": true, "could": true, "would": true, "should": true,
}
