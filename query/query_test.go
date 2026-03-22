package query

import (
	"testing"

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
