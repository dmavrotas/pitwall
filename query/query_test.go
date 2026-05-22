package query

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/dmavrotas/pitwall/ai"
	"github.com/dmavrotas/pitwall/loader"
	"github.com/dmavrotas/pitwall/models"
	"github.com/dmavrotas/pitwall/nlp"
	"github.com/dmavrotas/pitwall/store"
)

func setupEngine(t *testing.T) *Engine {
	t.Helper()
	ds := &loader.Dataset{
		Drivers: []models.Driver{
			{ID: 1, Ref: "hamilton", Code: "HAM", Forename: "Lewis", Surname: "Hamilton"},
			{ID: 2, Ref: "verstappen", Code: "VER", Forename: "Max", Surname: "Verstappen"},
		},
		Constructors: []models.Constructor{
			{ID: 131, Ref: "mercedes", Name: "Mercedes", Nationality: "German"},
			{ID: 9, Ref: "red_bull", Name: "Red Bull", Nationality: "Austrian"},
		},
		Circuits: []models.Circuit{
			{ID: 9, Ref: "silverstone", Name: "Silverstone Circuit", Location: "Silverstone", Country: "UK"},
		},
		Races: []models.Race{
			{ID: 1, Year: 2020, Round: 1, CircuitID: 9, Name: "British Grand Prix"},
			{ID: 2, Year: 2020, Round: 2, CircuitID: 9, Name: "70th Anniversary GP"},
		},
		Results: []models.Result{
			{ID: 1, RaceID: 1, DriverID: 1, ConstructorID: 131, Position: 1, PositionOrder: 1, Points: 25, Laps: 52, StatusID: 1},
			{ID: 2, RaceID: 1, DriverID: 2, ConstructorID: 9, Position: 2, PositionOrder: 2, Points: 18, Laps: 52, StatusID: 1},
			{ID: 3, RaceID: 2, DriverID: 2, ConstructorID: 9, Position: 1, PositionOrder: 1, Points: 25, Laps: 52, StatusID: 1},
			{ID: 4, RaceID: 2, DriverID: 1, ConstructorID: 131, Position: 2, PositionOrder: 2, Points: 18, Laps: 52, StatusID: 1},
		},
		Statuses: []models.Status{
			{ID: 1, Status: "Finished"},
		},
		Seasons: []models.Season{
			{Year: 2020},
		},
	}

	db, err := store.Load(ds)
	if err != nil {
		t.Fatalf("store.Load error: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	parser := nlp.NewParser(ds)
	return NewEngine(db, parser)
}

func TestAsk(t *testing.T) {
	engine := setupEngine(t)

	tests := []struct {
		name        string
		question    string
		wantErr     bool
		wantRows    bool
		wantDesc    string
	}{
		{
			name:     "most wins",
			question: "Who has the most wins?",
			wantErr:  false,
			wantRows: true,
		},
		{
			name:     "wins in 2020",
			question: "Who won the most races in 2020?",
			wantErr:  false,
			wantRows: true,
		},
		{
			name:     "hamilton points",
			question: "How many points did Hamilton score?",
			wantErr:  false,
			wantRows: true,
		},
		{
			name:     "compare drivers",
			question: "Compare Hamilton vs Verstappen",
			wantErr:  false,
			wantRows: true,
		},
		{
			name:     "empty query",
			question: "",
			wantErr:  true,
			wantRows: false,
		},
		{
			name:     "nonsense query",
			question: "xyzzy plugh",
			wantErr:  true,
			wantRows: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := engine.Ask(tt.question)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantRows && len(result.Rows) == 0 {
				t.Error("expected rows, got none")
			}
			if result.Description == "" {
				t.Error("expected non-empty description")
			}
			if len(result.Columns) == 0 {
				t.Error("expected columns")
			}
		})
	}
}

type fakeTranslator struct {
	calls atomic.Int64
	res   ai.Result
	err   error
}

func (f *fakeTranslator) Translate(_ context.Context, _ string) (ai.Result, error) {
	f.calls.Add(1)
	return f.res, f.err
}

func TestAIFallbackUsedOnlyWhenParserFails(t *testing.T) {
	base := setupEngine(t)
	fake := &fakeTranslator{
		res: ai.Result{
			SQL:         "SELECT forename || ' ' || surname AS driver FROM drivers ORDER BY id LIMIT 1",
			Description: "first driver",
		},
	}
	engine := NewEngineWithAI(base.db, base.parser, fake)

	// Known intent — should hit the rules path, not the AI.
	res, err := engine.Ask("Hamilton wins")
	if err != nil {
		t.Fatalf("rules path failed: %v", err)
	}
	if res.Source != "rules" {
		t.Errorf("expected source=rules, got %q", res.Source)
	}
	if fake.calls.Load() != 0 {
		t.Errorf("AI was called for a rules-handled query")
	}

	// Gibberish — parser returns ParseError, AI takes over.
	res, err = engine.Ask("xyzzy plugh frotz")
	if err != nil {
		t.Fatalf("AI fallback failed: %v", err)
	}
	if res.Source != "ai" {
		t.Errorf("expected source=ai, got %q", res.Source)
	}
	if fake.calls.Load() != 1 {
		t.Errorf("AI was called %d times, want 1", fake.calls.Load())
	}
}

func TestAskEndToEnd(t *testing.T) {
	engine := setupEngine(t)

	tests := []struct {
		name         string
		question     string
		wantFirstCol string // expected value in first column of first row
	}{
		{
			name:         "hamilton wins count",
			question:     "Hamilton wins",
			wantFirstCol: "Lewis Hamilton",
		},
		{
			name:         "verstappen info",
			question:     "Verstappen",
			wantFirstCol: "Max Verstappen",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := engine.Ask(tt.question)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(result.Rows) == 0 {
				t.Fatal("no rows returned")
			}
			if result.Rows[0][0] != tt.wantFirstCol {
				t.Errorf("first row first col = %q, want %q", result.Rows[0][0], tt.wantFirstCol)
			}
		})
	}
}

func setupSprintEngine(t *testing.T) *Engine {
	t.Helper()
	ds := &loader.Dataset{
		Drivers: []models.Driver{
			{ID: 1, Ref: "verstappen", Code: "VER", Forename: "Max", Surname: "Verstappen"},
			{ID: 2, Ref: "hamilton", Code: "HAM", Forename: "Lewis", Surname: "Hamilton"},
		},
		Constructors: []models.Constructor{
			{ID: 9, Ref: "red_bull", Name: "Red Bull", Nationality: "Austrian"},
			{ID: 131, Ref: "mercedes", Name: "Mercedes", Nationality: "German"},
		},
		Circuits: []models.Circuit{
			{ID: 1, Ref: "spa", Name: "Spa", Location: "Spa", Country: "Belgium"},
		},
		Races: []models.Race{
			{ID: 1, Year: 2023, Round: 1, CircuitID: 1, Name: "Belgian GP"},
		},
		Results: []models.Result{
			{ID: 1, RaceID: 1, DriverID: 1, ConstructorID: 9, Position: 1, PositionOrder: 1, Points: 25, StatusID: 1},
			{ID: 2, RaceID: 1, DriverID: 2, ConstructorID: 131, Position: 2, PositionOrder: 2, Points: 18, StatusID: 1},
		},
		SprintResults: []models.SprintResult{
			{ID: 1, RaceID: 1, DriverID: 1, ConstructorID: 9, Position: 1, PositionOrder: 1, Points: 8, StatusID: 1},
			{ID: 2, RaceID: 1, DriverID: 2, ConstructorID: 131, Position: 2, PositionOrder: 2, Points: 7, StatusID: 1},
		},
		Statuses: []models.Status{{ID: 1, Status: "Finished"}},
		Seasons:  []models.Season{{Year: 2023}},
	}

	db, err := store.Load(ds)
	if err != nil {
		t.Fatalf("store.Load error: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	return NewEngine(db, nlp.NewParser(ds))
}

func setupDNFEngine(t *testing.T) *Engine {
	t.Helper()
	ds := &loader.Dataset{
		Drivers: []models.Driver{
			{ID: 1, Ref: "hamilton", Code: "HAM", Forename: "Lewis", Surname: "Hamilton"},
			{ID: 2, Ref: "verstappen", Code: "VER", Forename: "Max", Surname: "Verstappen"},
			{ID: 3, Ref: "leclerc", Code: "LEC", Forename: "Charles", Surname: "Leclerc"},
		},
		Constructors: []models.Constructor{
			{ID: 1, Ref: "mercedes", Name: "Mercedes", Nationality: "German"},
		},
		Circuits: []models.Circuit{{ID: 1, Ref: "spa", Name: "Spa"}},
		Races: []models.Race{
			{ID: 1, Year: 2023, Round: 1, CircuitID: 1, Name: "Belgian GP"},
			{ID: 2, Year: 2023, Round: 2, CircuitID: 1, Name: "Italian GP"},
		},
		Statuses: []models.Status{
			{ID: 1, Status: "Finished"},
			{ID: 2, Status: "Accident"},
			{ID: 3, Status: "Engine"},
			{ID: 11, Status: "+1 Lap"},
			{ID: 12, Status: "+2 Laps"},
		},
		Results: []models.Result{
			// Race 1: Hamilton finished, Verstappen accident, Leclerc +1 Lap (classified)
			{ID: 1, RaceID: 1, DriverID: 1, ConstructorID: 1, Position: 1, PositionOrder: 1, StatusID: 1},
			{ID: 2, RaceID: 1, DriverID: 2, ConstructorID: 1, Position: 0, PositionOrder: 20, StatusID: 2},
			{ID: 3, RaceID: 1, DriverID: 3, ConstructorID: 1, Position: 18, PositionOrder: 18, StatusID: 11},
			// Race 2: Hamilton engine failure, Verstappen finished, Leclerc +2 Laps
			{ID: 4, RaceID: 2, DriverID: 1, ConstructorID: 1, Position: 0, PositionOrder: 20, StatusID: 3},
			{ID: 5, RaceID: 2, DriverID: 2, ConstructorID: 1, Position: 1, PositionOrder: 1, StatusID: 1},
			{ID: 6, RaceID: 2, DriverID: 3, ConstructorID: 1, Position: 17, PositionOrder: 17, StatusID: 12},
		},
		Seasons: []models.Season{{Year: 2023}},
	}
	db, err := store.Load(ds)
	if err != nil {
		t.Fatalf("store.Load error: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return NewEngine(db, nlp.NewParser(ds))
}

func TestDNFExcludesFinishedAndLappedClassified(t *testing.T) {
	engine := setupDNFEngine(t)
	result, err := engine.Ask("What are the most common DNF reasons?")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Only "Accident" and "Engine" should show — "Finished", "+1 Lap", "+2 Laps" excluded.
	gotReasons := map[string]string{}
	for _, row := range result.Rows {
		gotReasons[row[0]] = row[1]
	}
	if _, ok := gotReasons["Finished"]; ok {
		t.Error("DNF query incorrectly included Finished")
	}
	if _, ok := gotReasons["+1 Lap"]; ok {
		t.Error("DNF query incorrectly included +1 Lap (classified finisher)")
	}
	if _, ok := gotReasons["+2 Laps"]; ok {
		t.Error("DNF query incorrectly included +2 Laps (classified finisher)")
	}
	if gotReasons["Accident"] != "1" {
		t.Errorf("Accident count = %q, want %q", gotReasons["Accident"], "1")
	}
	if gotReasons["Engine"] != "1" {
		t.Errorf("Engine count = %q, want %q", gotReasons["Engine"], "1")
	}
}

func TestFirstWinReturnsEarliestRace(t *testing.T) {
	engine := setupEngine(t)
	// setupEngine: Hamilton wins race 1 (round 1, 2020), Verstappen wins race 2.
	result, err := engine.Ask("First win for Hamilton")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(result.Rows))
	}
	// Race column should be "British Grand Prix" (round 1)
	raceCol := -1
	for i, c := range result.Columns {
		if c == "race" {
			raceCol = i
			break
		}
	}
	if raceCol < 0 {
		t.Fatalf("race column not found in %v", result.Columns)
	}
	if result.Rows[0][raceCol] != "British Grand Prix" {
		t.Errorf("first win race = %q, want British Grand Prix", result.Rows[0][raceCol])
	}
}

func TestAveragePointsPerRace(t *testing.T) {
	engine := setupSprintEngine(t)
	// Verstappen: 25 (GP) + 8 (sprint) = 33 over 1 race → 33 per race.
	result, err := engine.Ask("Average points per race for Verstappen")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Rows) == 0 {
		t.Fatal("no rows")
	}
	ppr := -1
	for i, c := range result.Columns {
		if c == "points_per_race" {
			ppr = i
			break
		}
	}
	if ppr < 0 {
		t.Fatalf("points_per_race column not found in %v", result.Columns)
	}
	if result.Rows[0][ppr] != "33" {
		t.Errorf("Verstappen points per race = %q, want 33", result.Rows[0][ppr])
	}
}

func TestDriverVsTeamHeadToHead(t *testing.T) {
	engine := setupEngine(t)
	result, err := engine.Ask("Hamilton vs Mercedes")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Rows) != 2 {
		t.Fatalf("expected 2 rows (Driver + Team), got %d: %v", len(result.Rows), result.Rows)
	}
	kinds := []string{result.Rows[0][1], result.Rows[1][1]}
	driverThenTeam := kinds[0] == "Driver" && kinds[1] == "Team"
	teamThenDriver := kinds[0] == "Team" && kinds[1] == "Driver"
	if !driverThenTeam && !teamThenDriver {
		t.Errorf("expected one Driver and one Team row, got kinds=%v", kinds)
	}
}

func TestPointsIncludesSprintResults(t *testing.T) {
	engine := setupSprintEngine(t)

	result, err := engine.Ask("How many points did Verstappen score in 2023?")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Rows) == 0 {
		t.Fatal("no rows returned")
	}
	// Verstappen scored 25 GP points + 8 sprint points = 33 total.
	wantPoints := "33"
	pointsCol := -1
	for i, c := range result.Columns {
		if c == "total_points" {
			pointsCol = i
			break
		}
	}
	if pointsCol < 0 {
		t.Fatalf("total_points column not found in %v", result.Columns)
	}
	if result.Rows[0][pointsCol] != wantPoints {
		t.Errorf("Verstappen 2023 total_points = %q, want %q (GP+sprint sum)", result.Rows[0][pointsCol], wantPoints)
	}
}
