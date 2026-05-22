package ai

import (
	"fmt"
	"strings"
)

// allowedTables is the closed set of tables the LLM may reference.
var allowedTables = map[string]bool{
	"circuits": true, "constructors": true, "drivers": true,
	"races": true, "results": true, "sprint_results": true,
	"lap_times": true, "pit_stops": true, "qualifying": true,
	"driver_standings": true, "constructor_standings": true,
	"constructor_results": true, "status": true, "seasons": true,
}

// forbiddenKeywords would mutate state or escape the read-only sandbox.
var forbiddenKeywords = []string{
	"insert", "update", "delete", "drop", "create", "alter",
	"attach", "detach", "pragma", "vacuum", "replace",
}

// ValidateSQL enforces that the model's output is a single SELECT-only query
// against the allowed tables. It returns the (trimmed, single-statement) SQL on
// success, or an error explaining why the input was rejected.
func ValidateSQL(sql string) (string, error) {
	trimmed := strings.TrimSpace(sql)
	if trimmed == "" {
		return "", fmt.Errorf("empty SQL")
	}

	// Strip a single trailing semicolon. Anything else with `;` likely chains
	// statements — refuse.
	trimmed = strings.TrimSuffix(trimmed, ";")
	if strings.Contains(trimmed, ";") {
		return "", fmt.Errorf("multiple statements not allowed")
	}

	lower := strings.ToLower(trimmed)
	if !strings.HasPrefix(lower, "select") && !strings.HasPrefix(lower, "with") {
		return "", fmt.Errorf("only SELECT/WITH queries are allowed")
	}

	for _, kw := range forbiddenKeywords {
		// Pad with spaces to avoid matching substrings of column/table names.
		if strings.Contains(" "+lower+" ", " "+kw+" ") {
			return "", fmt.Errorf("forbidden keyword: %s", kw)
		}
	}

	// Collect CTE aliases declared with WITH/," <name> AS (" so we can permit
	// them in subsequent FROM/JOIN positions.
	cteAliases := collectCTEAliases(lower)

	// Surface table references and reject any outside the allowlist. This is a
	// best-effort lexical scan, not a SQL parse — it catches obvious slips.
	for _, kw := range []string{" from ", " join "} {
		idx := 0
		for {
			pos := strings.Index(lower[idx:], kw)
			if pos < 0 {
				break
			}
			start := idx + pos + len(kw)
			end := start
			for end < len(lower) && isIdentChar(lower[end]) {
				end++
			}
			name := lower[start:end]
			idx = end
			if name == "" {
				continue
			}
			if allowedTables[name] || cteAliases[name] {
				continue
			}
			if isProbablyTable(lower, end) {
				return "", fmt.Errorf("unknown table: %s", name)
			}
		}
	}

	return trimmed, nil
}

// collectCTEAliases scans for "<name> as (" patterns to find CTE names.
// Lexical only — does not handle nested CTEs perfectly, but the validator only
// needs to recognize the names as permitted; a stricter parser elsewhere can
// catch genuine errors.
func collectCTEAliases(lower string) map[string]bool {
	out := map[string]bool{}
	idx := 0
	for {
		pos := strings.Index(lower[idx:], " as (")
		if pos < 0 {
			break
		}
		nameEnd := idx + pos
		nameStart := nameEnd
		for nameStart > 0 && isIdentChar(lower[nameStart-1]) {
			nameStart--
		}
		if nameStart < nameEnd {
			out[lower[nameStart:nameEnd]] = true
		}
		idx = nameEnd + len(" as (")
	}
	return out
}

func isIdentChar(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= '0' && b <= '9') || b == '_'
}

// isProbablyTable looks ahead from the end of an identifier to decide if it's a
// table reference (followed by whitespace, alias, or end) rather than a CTE/
// subquery name. CTEs are typically followed by " as " and an opening paren on
// the WITH side, but on the FROM side they look identical to tables — we err
// on the side of allowing identifiers we can't classify.
func isProbablyTable(s string, end int) bool {
	// If the next non-space char is '(', this is a subquery/function call.
	for end < len(s) && s[end] == ' ' {
		end++
	}
	if end < len(s) && s[end] == '(' {
		return false
	}
	return true
}
