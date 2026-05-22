package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/dmavrotas/pitwall/ai"
	"github.com/dmavrotas/pitwall/loader"
	"github.com/dmavrotas/pitwall/nlp"
	"github.com/dmavrotas/pitwall/query"
	"github.com/dmavrotas/pitwall/store"
	"github.com/dmavrotas/pitwall/tui"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	dataDir := "data"
	plain := false
	useAI := false

	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--plain":
			plain = true
		case "--ai":
			useAI = true
		default:
			dataDir = args[i]
		}
	}

	fmt.Println("\n  Loading F1 dataset...")

	start := time.Now()
	ds, err := loader.LoadAll(dataDir)
	if err != nil {
		return fmt.Errorf("loading data: %w", err)
	}
	loadTime := time.Since(start)

	fmt.Println("  Indexing into query engine...")
	start = time.Now()
	db, err := store.Load(ds)
	if err != nil {
		return fmt.Errorf("creating store: %w", err)
	}
	defer func() { _ = db.Close() }()
	indexTime := time.Since(start)

	parser := nlp.NewParser(ds)

	var engine *query.Engine
	if useAI {
		if os.Getenv("ANTHROPIC_API_KEY") == "" {
			return fmt.Errorf("--ai requires ANTHROPIC_API_KEY to be set")
		}
		translator := ai.NewCache(ai.NewAnthropicTranslator("", ""))
		engine = query.NewEngineWithAI(db, parser, translator)
		fmt.Println("  AI fallback enabled.")
	} else {
		engine = query.NewEngine(db, parser)
	}

	if plain {
		runPlain(engine)
		return nil
	}

	statsInfo := fmt.Sprintf(
		"%d drivers · %d races · %d results · loaded in %v",
		len(ds.Drivers), len(ds.Races), len(ds.Results),
		(loadTime + indexTime).Round(time.Millisecond),
	)

	p := tui.New(engine, statsInfo)
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("running TUI: %w", err)
	}
	return nil
}

func runPlain(engine *query.Engine) {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("\npitwall (plain mode) — type 'quit' to exit")
	for {
		fmt.Print("pitwall> ")
		if !scanner.Scan() {
			break
		}
		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}
		if input == "quit" || input == "q" {
			break
		}

		result, err := engine.Ask(input)
		if err != nil {
			fmt.Printf("  Error: %v\n\n", err)
			continue
		}
		printPlainResult(result)
	}
}

func printPlainResult(r *query.Result) {
	fmt.Printf("\n  %s\n", r.Description)
	if len(r.Rows) == 0 {
		fmt.Println("  No results found.")
		return
	}

	widths := make([]int, len(r.Columns))
	for i, c := range r.Columns {
		widths[i] = len(c)
	}
	for _, row := range r.Rows {
		for i, cell := range row {
			if i < len(widths) && len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}
	for i := range widths {
		if widths[i] > 35 {
			widths[i] = 35
		}
	}

	totalW := 0
	for _, w := range widths {
		totalW += w + 3
	}

	sep := "  " + strings.Repeat("-", totalW)
	fmt.Println(sep)
	fmt.Print("  ")
	for i, c := range r.Columns {
		fmt.Printf("%-*s   ", widths[i], strings.ReplaceAll(strings.ToUpper(c), "_", " "))
	}
	fmt.Println()
	fmt.Println(sep)
	for _, row := range r.Rows {
		fmt.Print("  ")
		for i, cell := range row {
			if i >= len(widths) {
				break
			}
			if len(cell) > widths[i] {
				cell = cell[:widths[i]-1] + "…"
			}
			fmt.Printf("%-*s   ", widths[i], cell)
		}
		fmt.Println()
	}
	fmt.Println(sep)
	fmt.Printf("  %d rows\n\n", len(r.Rows))
}
