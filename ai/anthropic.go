package ai

import (
	"context"
	"fmt"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// AnthropicTranslator calls the Anthropic Messages API to translate questions
// into SELECT SQL. It is constructed once at start-up and reused; the SDK
// client itself handles connection pooling.
type AnthropicTranslator struct {
	client anthropic.Client
	model  anthropic.Model
}

// NewAnthropicTranslator builds a translator using the provided API key. If
// apiKey is empty, the SDK falls back to the ANTHROPIC_API_KEY env var. The
// model defaults to Claude Opus 4.7; callers may override via opts.
func NewAnthropicTranslator(apiKey string, model anthropic.Model) *AnthropicTranslator {
	var opts []option.RequestOption
	if apiKey != "" {
		opts = append(opts, option.WithAPIKey(apiKey))
	}
	if model == "" {
		model = anthropic.ModelClaudeOpus4_7
	}
	return &AnthropicTranslator{
		client: anthropic.NewClient(opts...),
		model:  model,
	}
}

const systemPromptTemplate = `You translate natural-language Formula 1 questions into a single SQLite SELECT statement against the schema below.

OUTPUT FORMAT — strict:
- Line 1: the SQL (a single SELECT or WITH ... SELECT, no semicolons, no markdown fences).
- Line 2: blank.
- Line 3: a short, plain-English description of what the query returns (≤ 80 chars).

If you cannot answer the question with a SELECT against this schema, output exactly the single line: UNANSWERABLE

Rules:
- Only the tables and columns shown may be referenced.
- No INSERT/UPDATE/DELETE/DROP/PRAGMA/ATTACH.
- Use ` + "`status_id`" + ` joins to status(id, status) for DNF/finish reasoning, not raw position checks.
- Championship-point totals include sprint points — UNION/SUM both results.points and sprint_results.points.

SCHEMA:
%s

EXAMPLES:
Q: Average points per race for Hamilton since 2018
SQL:
WITH all_points AS (SELECT driver_id, race_id, points FROM results UNION ALL SELECT driver_id, race_id, points FROM sprint_results) SELECT ROUND(SUM(ap.points) * 1.0 / COUNT(DISTINCT ap.race_id), 2) AS points_per_race FROM all_points ap JOIN drivers d ON d.id = ap.driver_id JOIN races ra ON ra.id = ap.race_id WHERE d.surname = 'Hamilton' AND ra.year >= 2018

Average championship points per race for Hamilton since 2018

Q: Which driver led the most laps at Monaco in the 2010s
SQL:
SELECT d.forename || ' ' || d.surname AS driver, COUNT(*) AS laps_led FROM lap_times lt JOIN drivers d ON d.id = lt.driver_id JOIN races ra ON ra.id = lt.race_id JOIN circuits ci ON ci.id = ra.circuit_id WHERE lt.position = 1 AND ci.ref = 'monaco' AND ra.year BETWEEN 2010 AND 2019 GROUP BY lt.driver_id ORDER BY laps_led DESC LIMIT 5

Most laps led at Monaco between 2010 and 2019`

// Translate sends the question to Anthropic and parses out a validated SQL plan.
func (t *AnthropicTranslator) Translate(ctx context.Context, question string) (Result, error) {
	system := fmt.Sprintf(systemPromptTemplate, SchemaDescription)

	resp, err := t.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     t.model,
		MaxTokens: 1024,
		System: []anthropic.TextBlockParam{
			{Text: system},
		},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(question)),
		},
	})
	if err != nil {
		return Result{}, fmt.Errorf("anthropic call: %w", err)
	}

	var raw strings.Builder
	for i := range resp.Content {
		if tb, ok := resp.Content[i].AsAny().(anthropic.TextBlock); ok {
			raw.WriteString(tb.Text)
		}
	}

	return parseLLMOutput(raw.String())
}

// parseLLMOutput extracts SQL + description from the model output and
// validates the SQL is safe.
func parseLLMOutput(raw string) (Result, error) {
	text := strings.TrimSpace(raw)
	if text == "" {
		return Result{}, fmt.Errorf("empty response")
	}

	// Strip optional markdown code fences just in case.
	text = strings.TrimPrefix(text, "```sql")
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimSuffix(text, "```")
	text = strings.TrimSpace(text)

	if strings.EqualFold(text, "UNANSWERABLE") {
		return Result{}, ErrUnanswerable
	}

	// Split: SQL on first non-blank lines, description after the first blank line.
	parts := strings.SplitN(text, "\n\n", 2)
	sqlPart := strings.TrimSpace(parts[0])
	desc := ""
	if len(parts) == 2 {
		desc = strings.TrimSpace(parts[1])
		// Description should be a single line; take the first if more arrive.
		if idx := strings.IndexByte(desc, '\n'); idx >= 0 {
			desc = strings.TrimSpace(desc[:idx])
		}
	}

	validated, err := ValidateSQL(sqlPart)
	if err != nil {
		return Result{}, fmt.Errorf("rejected SQL: %w", err)
	}
	if desc == "" {
		desc = "Result"
	}
	return Result{SQL: validated, Description: desc}, nil
}
