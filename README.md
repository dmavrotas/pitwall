# Pitwall

A natural language query engine for Formula 1 historical data, built in Go.

Ask questions about F1 in plain English and get instant answers from 75+ years of race data.

## Demo

```
pitwall> Who has the most wins?

  Top drivers by race wins
  ─────────────────────────────────────────
  DRIVER               │ TEAM      │ WINS
  ─────────────────────────────────────────
  Lewis Hamilton       │ McLaren   │ 105
  Michael Schumacher   │ Ferrari   │ 91
  Max Verstappen       │ Red Bull  │ 71
  ...

pitwall> How many championships does Alain Prost have?

  Driver championship history
  ───────────────────────────────────────────────
  SEASON │ POSITION │ POINTS │ WINS │ TITLE
  ───────────────────────────────────────────────
  1985   │ 1        │ 73     │ 5    │ CHAMPION
  1986   │ 1        │ 72     │ 4    │ CHAMPION
  1989   │ 1        │ 76     │ 4    │ CHAMPION
  1993   │ 1        │ 99     │ 7    │ CHAMPION
  ...

pitwall> Compare Verstappen vs Hamilton

  Head to head comparison
  ─────────────────────────────────────────────────────────────────
  DRIVER          │ RACES │ WINS │ PODIUMS │ POINTS  │ AVG FINISH
  ─────────────────────────────────────────────────────────────────
  Lewis Hamilton  │ 382   │ 105  │ 203     │ 4982.5  │ 5.2
  Max Verstappen  │ 235   │ 71   │ 127     │ 3309.5  │ 5.5
```

## Features

- **Natural language queries** — ask in plain English, get structured answers
- **14 data tables** — drivers, races, results, lap times, pit stops, qualifying, standings, circuits, and more
- **Full-screen TUI** — F1-themed terminal interface with scrollable history and styled tables
- **Offline by default** — deterministic rule-based parser, no external APIs required
- **Optional AI fallback** — opt into Claude-backed text-to-SQL with `--ai` for questions the rule parser can't classify
- **Fast** — loads 600K+ lap times and 27K+ results in under 2 seconds

## Supported Query Types

| Category | Example Questions |
|----------|------------------|
| Wins | "Who has the most wins?", "Ferrari wins in 2004", "First win for Verstappen" |
| Points | "How many points did Hamilton score in 2019?", "Average points per race for Hamilton since 2018" |
| Championships | "Who won the 2008 championship?" (champion only), "2021 championship standings" (full top-15) |
| Poles | "Who got the most pole positions?" |
| DNFs | "What are the most common DNF reasons?" |
| Pit Stops | "Fastest pit stops in 2023" |
| Fastest Laps | "Fastest laps at Silverstone" |
| Comparisons | "Compare Verstappen vs Hamilton", "Hamilton vs Ferrari in 2018" |
| Teammates | "Who were Hamilton's teammates?" |
| Driver/Team Info | "Tell me about Ferrari" |
| Circuits | "Tell me about Monza" |
| Seasons | "Season overview 2010" |
| Year ranges | "Wins between 2018 and 2022", "Poles since 2020", "DNF reasons before 2010" |
| Ordinals | "First win for Verstappen", "Last podium for Schumacher", "Worst finish for Hamilton" |

### Semantics
- **Sprint era**: points totals include both grand prix and sprint contributions from 2021 onward.
- **DNFs**: filtered by race status (`Finished` and `+N Lap` classifications excluded) — robust to NULL position fields.
- **Driver-vs-team comparisons**: pair a driver and a constructor in the same question to get a side-by-side breakdown (best scoped to a year).

## AI fallback (optional)

When the rule-based parser can't classify a question, you can optionally fall back to Claude to translate the question into a sandboxed `SELECT`. Disabled by default — the offline guarantee is preserved unless you explicitly opt in.

```bash
export ANTHROPIC_API_KEY=...
go run ./cmd --ai data/
```

The fallback path:
- Tries the rule parser first; AI is only invoked on `ParseError`.
- Generated SQL is validated before execution: SELECT/WITH-only, no `;` chains, allowlisted tables.
- Identical questions are cached in-process for the lifetime of the session.
- Each result is tagged with `Source: "rules"` or `Source: "ai"` so you can tell which path answered.

## Setup

### Prerequisites

- Go 1.21+
- [golangci-lint](https://golangci-lint.run/welcome/install/) (for linting)

### Get the Data

Download the F1 dataset from [Kaggle](https://www.kaggle.com/datasets/jtrotman/formula-1-race-data) and extract the CSVs into `data/`:

```bash
# With Kaggle CLI
kaggle datasets download -d jtrotman/formula-1-race-data -p data --unzip
```

### Build & Run

```bash
# Run the TUI
go run ./cmd data/

# Run in plain/pipe mode
go run ./cmd --plain data/

# Enable the AI fallback (requires ANTHROPIC_API_KEY)
ANTHROPIC_API_KEY=... go run ./cmd --ai data/

# Build binary
go build -o pitwall ./cmd
./pitwall data/
```

### Development

```bash
# Run tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Lint
golangci-lint run ./...
```

## Architecture

```
User Question → Tokenizer → Entity Extraction → Intent Detection → SQL Builder → SQLite → Table Formatter
                                                                       │
                                                       (on ParseError) ↓ (opt-in, --ai)
                                                                 Claude text-to-SQL → SQL Validator
```

The system works in three stages, with an optional fourth:

1. **NLP Layer** — Tokenizes input, removes stop words, extracts entities (drivers, teams, circuits, single year, year ranges, modifiers like "average" / "first" / "worst"), and detects query intent via keyword scoring with priority tiers.

2. **SQL Layer** — Each intent maps to a parameterized SQL template with optional filters. Templates are composed at runtime based on which entities and modifiers were found.

3. **Store Layer** — All CSV data is loaded into an in-memory SQLite database (pure Go, no CGO) with indexes on foreign keys for fast joins.

4. **AI Fallback (optional)** — When the rule parser fails and `--ai` is enabled, the question is sent to Claude. The returned SQL is validated against an allowlist of tables and a SELECT-only check before execution. Results are cached by normalized question.

## Project Structure

```
pitwall/
├── cmd/main.go           Entry point, CLI flags, REPL
├── models/models.go      F1 data structs (14 tables)
├── loader/loader.go      CSV → Go struct parser
├── store/
│   ├── schema.go         SQLite DDL + indexes
│   └── store.go          DB load, query execution
├── nlp/
│   ├── nlp.go            Parser orchestrator
│   ├── entities.go       Name-to-ID dictionaries, year-range + modifier parsing
│   ├── intents.go        Intent detection (14 types)
│   └── templates.go      SQL template builder
├── query/query.go        Engine + result types (rule + optional AI path)
├── ai/
│   ├── translator.go     Translator interface
│   ├── anthropic.go      Claude SDK implementation
│   ├── validator.go      SQL safety check (SELECT-only, allowlisted tables)
│   ├── schema.go         Schema description for the prompt
│   └── cache.go          In-process question cache
├── tui/tui.go            Terminal UI (bubbletea)
├── analysis/analysis.go  Batch analysis (standalone)
├── data/                 CSV files (not committed)
├── .golangci.yml         Linter config
└── go.mod
```

## License

Data sourced from [Ergast Motor Racing Data](http://ergast.com/mrd/) (CC0 public domain) and [Jolpi F1 API](http://api.jolpi.ca/ergast/f1/).
