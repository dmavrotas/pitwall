# Pitwall - F1 Data Analysis Engine

## Project Overview
Natural language query engine for Formula 1 historical data (1950-2026). Users ask questions in English ("Who has the most wins in Monaco?") and get structured tabular answers. Data sourced from the Ergast F1 dataset via Kaggle.

## Architecture

```
User Input (English) → NLP Parser → SQL Query → In-Memory SQLite → Formatted Output
```

### Package Structure
- `models/` — Go structs for all 14 F1 data tables (Circuit, Driver, Race, Result, LapTime, PitStop, etc.)
- `loader/` — CSV parser that reads Ergast dataset files into a `Dataset` struct. Handles `\N` null values.
- `store/` — In-memory SQLite layer. `schema.go` has DDL, `store.go` handles load + query execution with named parameter substitution.
- `nlp/` — Natural language to SQL translation:
  - `entities.go` — Builds name-to-ID dictionaries from the dataset. Disambiguates by popularity (result count).
  - `intents.go` — Keyword-based intent detection with priority tiers (action intents > info intents).
  - `templates.go` — SQL template builder per intent, with dynamic filters for driver/constructor/circuit/year.
  - `nlp.go` — Orchestrator: tokenize → extract entities → detect intent → build SQL → return QueryPlan.
- `query/` — Query engine that ties NLP parser to SQLite store. Returns structured `QueryResult`.
- `analysis/` — Standalone batch analysis functions (legacy, still functional).
- `tui/` — Bubbletea-based terminal UI with F1 theming, scrollable history, styled tables.
- `cmd/` — Entry point. Supports `--plain` flag for non-interactive/piped mode.
- `data/` — CSV files (not committed, download from Kaggle).

### Key Design Decisions
- **Pure Go SQLite** (`modernc.org/sqlite`) — no CGO, fully portable, works offline.
- **In-memory database** — data is read-only, loaded fresh each run. No disk artifacts.
- **Pattern-based NLP** — no external AI APIs. Intent detection via keyword scoring with priority tiers. Deterministic, testable, debuggable.
- **Named parameter substitution** — SQL templates use `:param_name` placeholders, converted to positional `?` at query time by left-to-right scanning.
- **Surname disambiguation** — when multiple drivers share a surname (e.g., "Hamilton"), the one with the most career race results wins the mapping.

## Build & Run

```bash
# Build
go build ./...

# Run TUI mode
go run ./cmd data/

# Run plain/pipe mode
go run ./cmd --plain data/

# Lint
golangci-lint run ./...

# Test
go test ./...

# Test with coverage
go test -cover ./...

# Test verbose
go test -v ./...
```

## Dataset
Download from https://www.kaggle.com/datasets/jtrotman/formula-1-race-data and extract CSVs into `data/`. Required files: `circuits.csv`, `constructors.csv`, `drivers.csv`, `races.csv`, `results.csv`, `lap_times.csv`, `pit_stops.csv`, `qualifying.csv`, `driver_standings.csv`, `constructor_standings.csv`, `constructor_results.csv`, `status.csv`, `seasons.csv`, `sprint_results.csv`.

## Testing Conventions
- Table-driven tests using `[]struct{ name string; ... }` pattern.
- Test files live alongside source: `foo_test.go` next to `foo.go`.
- Use `t.Run(tt.name, ...)` for subtests.
- The `nlp` and `store` packages build a small fixture `loader.Dataset` in tests rather than loading CSVs from disk.
- Tests must not depend on external files or network. All test data is inline.

## Adding a New Query Type
1. Add keywords to `nlp/intents.go` (action intent = priority 2, info intent = priority 1).
2. Add a `buildXxxQuery` function in `nlp/templates.go`.
3. Wire it in `BuildQuery` switch statement in `templates.go`.
4. Add table-driven test cases in `nlp/nlp_test.go`.

## Dependencies
- `modernc.org/sqlite` — pure Go SQLite
- `github.com/charmbracelet/bubbletea` — TUI framework
- `github.com/charmbracelet/bubbles` — TUI components (text input, viewport)
- `github.com/charmbracelet/lipgloss` — TUI styling
