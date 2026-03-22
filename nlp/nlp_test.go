package nlp

import (
	"strings"
	"testing"

	"github.com/dmavrotas/pitwall/loader"
	"github.com/dmavrotas/pitwall/models"
)

func testDataset() *loader.Dataset {
	return &loader.Dataset{
		Drivers: []models.Driver{
			{ID: 1, Ref: "hamilton", Code: "HAM", Forename: "Lewis", Surname: "Hamilton"},
			{ID: 2, Ref: "max_verstappen", Code: "VER", Forename: "Max", Surname: "Verstappen"},
			{ID: 3, Ref: "prost", Code: "PRO", Forename: "Alain", Surname: "Prost"},
			{ID: 4, Ref: "senna", Code: "SEN", Forename: "Ayrton", Surname: "Senna"},
			{ID: 5, Ref: "duncan_hamilton", Code: "", Forename: "Duncan", Surname: "Hamilton"},
		},
		Constructors: []models.Constructor{
			{ID: 6, Ref: "ferrari", Name: "Ferrari", Nationality: "Italian"},
			{ID: 9, Ref: "red_bull", Name: "Red Bull", Nationality: "Austrian"},
			{ID: 131, Ref: "mercedes", Name: "Mercedes", Nationality: "German"},
			{ID: 1, Ref: "mclaren", Name: "McLaren", Nationality: "British"},
		},
		Circuits: []models.Circuit{
			{ID: 6, Ref: "monaco", Name: "Circuit de Monaco", Location: "Monte-Carlo", Country: "Monaco"},
			{ID: 9, Ref: "silverstone", Name: "Silverstone Circuit", Location: "Silverstone", Country: "UK"},
			{ID: 14, Ref: "monza", Name: "Autodromo Nazionale di Monza", Location: "Monza", Country: "Italy"},
		},
		Results: []models.Result{
			{ID: 1, RaceID: 1, DriverID: 1, ConstructorID: 131, Position: 1},
			{ID: 2, RaceID: 1, DriverID: 2, ConstructorID: 9, Position: 2},
			{ID: 3, RaceID: 2, DriverID: 1, ConstructorID: 131, Position: 1},
			{ID: 4, RaceID: 2, DriverID: 3, ConstructorID: 1, Position: 3},
		},
		Races: []models.Race{
			{ID: 1, Year: 2020, Round: 1, CircuitID: 9, Name: "British Grand Prix"},
			{ID: 2, Year: 2020, Round: 2, CircuitID: 6, Name: "Monaco Grand Prix"},
		},
	}
}

func TestTokenize(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "simple query",
			input:    "who has the most wins",
			expected: []string{"wins"},
		},
		{
			name:     "with year",
			input:    "wins in 2020",
			expected: []string{"wins", "2020"},
		},
		{
			name:     "with driver name",
			input:    "Hamilton wins",
			expected: []string{"hamilton", "wins"},
		},
		{
			name:     "with punctuation",
			input:    "How many wins does Hamilton have?",
			expected: []string{"wins", "hamilton"},
		},
		{
			name:     "preserves hyphens",
			input:    "head-to-head comparison",
			expected: []string{"head-to-head", "comparison"},
		},
		{
			name:     "empty after stop word removal",
			input:    "the a an is are",
			expected: nil,
		},
		{
			name:     "mixed case",
			input:    "Ferrari WINS 2004",
			expected: []string{"ferrari", "wins", "2004"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tokenize(tt.input)
			if len(got) != len(tt.expected) {
				t.Fatalf("tokenize(%q) = %v, want %v", tt.input, got, tt.expected)
			}
			for i := range got {
				if got[i] != tt.expected[i] {
					t.Errorf("tokenize(%q)[%d] = %q, want %q", tt.input, i, got[i], tt.expected[i])
				}
			}
		})
	}
}

func TestDetectIntent(t *testing.T) {
	tests := []struct {
		name   string
		tokens []string
		want   Intent
	}{
		{"wins keyword", []string{"wins", "2020"}, IntentWins},
		{"won keyword", []string{"won", "races"}, IntentWins},
		{"points keyword", []string{"points", "hamilton"}, IntentPoints},
		{"championship keyword", []string{"championship", "prost"}, IntentChampionship},
		{"championships plural", []string{"championships", "ferrari"}, IntentChampionship},
		{"title keyword", []string{"title", "2021"}, IntentChampionship},
		{"pole keyword", []string{"pole", "positions"}, IntentPoles},
		{"dnf keyword", []string{"dnf", "reasons"}, IntentDNF},
		{"pit stop keyword", []string{"pit", "stop", "fastest"}, IntentPitStops},
		{"fastest lap keyword", []string{"fastest", "lap", "monza"}, IntentFastestLap},
		{"podium keyword", []string{"podiums", "2022"}, IntentRaceResult},
		{"compare keyword", []string{"compare", "hamilton", "verstappen"}, IntentHeadToHead},
		{"vs keyword", []string{"hamilton", "vs", "verstappen"}, IntentHeadToHead},
		{"teammate keyword", []string{"teammates", "hamilton"}, IntentTeammates},
		{"season keyword", []string{"season", "overview", "2021"}, IntentSeasonOverview},
		{"standings keyword", []string{"standings", "2021"}, IntentSeasonOverview},
		{"unknown", []string{"hello", "world"}, IntentUnknown},

		// Priority tests: action intents beat info intents
		{"wins beats circuit", []string{"wins", "circuit", "monaco"}, IntentWins},
		{"wins beats driver", []string{"wins", "driver", "hamilton"}, IntentWins},
		{"championship beats team", []string{"championships", "team", "ferrari"}, IntentChampionship},
		{"points beats constructor", []string{"points", "constructor", "mercedes"}, IntentPoints},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectIntent(tt.tokens)
			if got != tt.want {
				t.Errorf("DetectIntent(%v) = %v, want %v", tt.tokens, got, tt.want)
			}
		})
	}
}

func TestExtractEntities(t *testing.T) {
	ds := testDataset()
	ed := BuildEntityDict(ds)

	tests := []struct {
		name   string
		tokens []string
		want   Entities
	}{
		{
			name:   "driver by surname",
			tokens: []string{"hamilton", "wins"},
			want:   Entities{DriverID: 1}, // Lewis Hamilton (more results) beats Duncan Hamilton
		},
		{
			name:   "driver by full name",
			tokens: []string{"lewis", "hamilton"},
			want:   Entities{DriverID: 1},
		},
		{
			name:   "driver by code",
			tokens: []string{"ver", "wins"},
			want:   Entities{DriverID: 2},
		},
		{
			name:   "constructor",
			tokens: []string{"ferrari", "wins"},
			want:   Entities{ConstructorID: 6},
		},
		{
			name:   "constructor multi-word",
			tokens: []string{"red", "bull"},
			want:   Entities{ConstructorID: 9},
		},
		{
			name:   "constructor no space",
			tokens: []string{"redbull"},
			want:   Entities{ConstructorID: 9},
		},
		{
			name:   "circuit by location",
			tokens: []string{"monza"},
			want:   Entities{CircuitID: 14},
		},
		{
			name:   "circuit by ref",
			tokens: []string{"silverstone"},
			want:   Entities{CircuitID: 9},
		},
		{
			name:   "year extraction",
			tokens: []string{"wins", "2020"},
			want:   Entities{Year: 2020},
		},
		{
			name:   "year out of range ignored",
			tokens: []string{"wins", "1800"},
			want:   Entities{},
		},
		{
			name:   "two drivers for comparison",
			tokens: []string{"hamilton", "verstappen"},
			want:   Entities{DriverID: 1, DriverID2: 2},
		},
		{
			name:   "two constructors for comparison",
			tokens: []string{"ferrari", "mercedes"},
			want:   Entities{ConstructorID: 6, ConstructorID2: 131},
		},
		{
			name:   "driver + year + circuit",
			tokens: []string{"hamilton", "2020", "silverstone"},
			want:   Entities{DriverID: 1, Year: 2020, CircuitID: 9},
		},
		{
			name:   "surname disambiguation picks popular driver",
			tokens: []string{"hamilton"},
			want:   Entities{DriverID: 1}, // Lewis, not Duncan
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ed.ExtractEntities(tt.tokens)
			if got != tt.want {
				t.Errorf("ExtractEntities(%v) = %+v, want %+v", tt.tokens, got, tt.want)
			}
		})
	}
}

func TestParserParse(t *testing.T) {
	ds := testDataset()
	parser := NewParser(ds)

	tests := []struct {
		name       string
		input      string
		wantIntent Intent
		wantErr    bool
	}{
		{"most wins", "Who has the most wins?", IntentWins, false},
		{"wins in year", "Who won the most races in 2020?", IntentWins, false},
		{"driver points", "How many points did Hamilton score in 2019?", IntentPoints, false},
		{"championship standings", "Show me the 2021 championship standings", IntentChampionship, false},
		{"compare drivers", "Compare Verstappen vs Hamilton", IntentHeadToHead, false},
		{"dnf reasons", "What are the most common DNF reasons?", IntentDNF, false},
		{"pit stops", "Fastest pit stops in 2023", IntentPitStops, false},
		{"poles", "Who got the most pole positions?", IntentPoles, false},
		{"circuit info", "Tell me about Monza", IntentCircuitInfo, false},
		{"teammates", "Who were Hamilton's teammates?", IntentTeammates, false},
		{"constructor wins", "Ferrari wins in 2004", IntentWins, false},
		{"season overview", "Season overview 2010", IntentSeasonOverview, false},
		{"fastest laps", "Fastest laps at Silverstone", IntentFastestLap, false},
		{"podiums", "Podiums in 2022", IntentRaceResult, false},
		{"driver championship", "How many championships Alain Prost has?", IntentChampionship, false},
		{"wins at circuit", "Who has the most wins in Monaco Circuit?", IntentWins, false},
		{"empty query", "", IntentUnknown, true},

		// Entity-only fallback intent inference
		{"just a driver name", "Hamilton", IntentDriverInfo, false},
		{"just a constructor", "Ferrari", IntentConstructorInfo, false},
		{"just a year", "2020", IntentSeasonOverview, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan, err := parser.Parse(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("Parse(%q) expected error, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("Parse(%q) unexpected error: %v", tt.input, err)
			}
			if plan.Intent != tt.wantIntent {
				t.Errorf("Parse(%q).Intent = %v, want %v", tt.input, plan.Intent, tt.wantIntent)
			}
			if plan.SQL == "" {
				t.Errorf("Parse(%q).SQL is empty", tt.input)
			}
		})
	}
}

func TestInferIntent(t *testing.T) {
	tests := []struct {
		name     string
		entities Entities
		want     Intent
	}{
		{"two drivers", Entities{DriverID: 1, DriverID2: 2}, IntentHeadToHead},
		{"two constructors", Entities{ConstructorID: 1, ConstructorID2: 2}, IntentHeadToHead},
		{"single driver", Entities{DriverID: 1}, IntentDriverInfo},
		{"single constructor", Entities{ConstructorID: 1}, IntentConstructorInfo},
		{"single circuit", Entities{CircuitID: 1}, IntentCircuitInfo},
		{"year only", Entities{Year: 2020}, IntentSeasonOverview},
		{"empty", Entities{}, IntentUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := inferIntent(tt.entities)
			if got != tt.want {
				t.Errorf("inferIntent(%+v) = %v, want %v", tt.entities, got, tt.want)
			}
		})
	}
}

func TestBuildQueryReturnsSQL(t *testing.T) {
	tests := []struct {
		name     string
		intent   Intent
		entities Entities
		wantSQL  bool
	}{
		{"wins no filter", IntentWins, Entities{}, true},
		{"wins with year", IntentWins, Entities{Year: 2020}, true},
		{"wins with driver", IntentWins, Entities{DriverID: 1}, true},
		{"wins with circuit", IntentWins, Entities{CircuitID: 6}, true},
		{"points no filter", IntentPoints, Entities{}, true},
		{"points with driver and year", IntentPoints, Entities{DriverID: 1, Year: 2019}, true},
		{"championship all time", IntentChampionship, Entities{}, true},
		{"championship for driver", IntentChampionship, Entities{DriverID: 3}, true},
		{"championship for year", IntentChampionship, Entities{Year: 2021}, true},
		{"championship for constructor", IntentChampionship, Entities{ConstructorID: 6}, true},
		{"poles", IntentPoles, Entities{}, true},
		{"dnf", IntentDNF, Entities{}, true},
		{"pit stops by year", IntentPitStops, Entities{}, true},
		{"pit stops in year", IntentPitStops, Entities{Year: 2023}, true},
		{"fastest lap", IntentFastestLap, Entities{}, true},
		{"race result with circuit", IntentRaceResult, Entities{CircuitID: 9, Year: 2020}, true},
		{"driver info specific", IntentDriverInfo, Entities{DriverID: 1}, true},
		{"driver info all", IntentDriverInfo, Entities{}, true},
		{"constructor info specific", IntentConstructorInfo, Entities{ConstructorID: 6}, true},
		{"circuit info specific", IntentCircuitInfo, Entities{CircuitID: 14}, true},
		{"teammates", IntentTeammates, Entities{DriverID: 1}, true},
		{"head to head drivers", IntentHeadToHead, Entities{DriverID: 1, DriverID2: 2}, true},
		{"head to head constructors", IntentHeadToHead, Entities{ConstructorID: 6, ConstructorID2: 9}, true},
		{"season overview year", IntentSeasonOverview, Entities{Year: 2021}, true},
		{"season overview all", IntentSeasonOverview, Entities{}, true},
		{"unknown intent", IntentUnknown, Entities{}, false},
		{"teammates no driver", IntentTeammates, Entities{}, false},
		{"head to head one driver", IntentHeadToHead, Entities{DriverID: 1}, false},

		// Additional coverage for filter combinations
		{"wins with constructor", IntentWins, Entities{ConstructorID: 6}, true},
		{"wins driver + year", IntentWins, Entities{DriverID: 1, Year: 2020}, true},
		{"wins circuit + year", IntentWins, Entities{CircuitID: 6, Year: 2020}, true},
		{"points with constructor", IntentPoints, Entities{ConstructorID: 6}, true},
		{"points driver only", IntentPoints, Entities{DriverID: 1}, true},
		{"poles with driver", IntentPoles, Entities{DriverID: 1}, true},
		{"poles with year", IntentPoles, Entities{Year: 2020}, true},
		{"poles driver + year", IntentPoles, Entities{DriverID: 1, Year: 2020}, true},
		{"dnf with year", IntentDNF, Entities{Year: 2020}, true},
		{"dnf with driver", IntentDNF, Entities{DriverID: 1}, true},
		{"dnf driver + year", IntentDNF, Entities{DriverID: 1, Year: 2020}, true},
		{"fastest lap with year", IntentFastestLap, Entities{Year: 2020}, true},
		{"fastest lap with driver", IntentFastestLap, Entities{DriverID: 1}, true},
		{"fastest lap with circuit", IntentFastestLap, Entities{CircuitID: 9}, true},
		{"fastest lap all filters", IntentFastestLap, Entities{DriverID: 1, Year: 2020, CircuitID: 9}, true},
		{"race result no filter", IntentRaceResult, Entities{}, true},
		{"race result with year", IntentRaceResult, Entities{Year: 2020}, true},
		{"race result driver + year", IntentRaceResult, Entities{DriverID: 1, Year: 2020}, true},
		{"race result with constructor", IntentRaceResult, Entities{ConstructorID: 6}, true},
		{"constructor info all", IntentConstructorInfo, Entities{}, true},
		{"circuit info all", IntentCircuitInfo, Entities{}, true},
		{"teammates with year", IntentTeammates, Entities{DriverID: 1, Year: 2020}, true},
		{"head to head drivers + year", IntentHeadToHead, Entities{DriverID: 1, DriverID2: 2, Year: 2020}, true},
		{"head to head constructors + year", IntentHeadToHead, Entities{ConstructorID: 6, ConstructorID2: 9, Year: 2020}, true},
		{"championship driver + year", IntentChampionship, Entities{DriverID: 3, Year: 1985}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql, desc, args := BuildQuery(tt.intent, tt.entities)
			if tt.wantSQL {
				if sql == "" {
					t.Errorf("BuildQuery(%v, %+v) returned empty SQL", tt.intent, tt.entities)
				}
				if desc == "" {
					t.Errorf("BuildQuery(%v, %+v) returned empty description", tt.intent, tt.entities)
				}
				if args == nil {
					t.Errorf("BuildQuery(%v, %+v) returned nil args", tt.intent, tt.entities)
				}
			} else if sql != "" {
				t.Errorf("BuildQuery(%v, %+v) expected empty SQL, got %q", tt.intent, tt.entities, sql)
			}
		})
	}
}

func TestParseError(t *testing.T) {
	err := &ParseError{Input: "hello world", Reason: "test reason"}
	got := err.Error()
	if !strings.Contains(got, "test reason") {
		t.Errorf("ParseError.Error() = %q, want it to contain 'test reason'", got)
	}
}

func TestExampleQuestions(t *testing.T) {
	examples := ExampleQuestions()
	if len(examples) == 0 {
		t.Error("ExampleQuestions() returned empty slice")
	}
	for i, q := range examples {
		if q == "" {
			t.Errorf("ExampleQuestions()[%d] is empty", i)
		}
	}
}

func TestParseErrorFromParser(t *testing.T) {
	ds := testDataset()
	parser := NewParser(ds)

	_, err := parser.Parse("")
	if err == nil {
		t.Fatal("expected error for empty input")
	}
	pe, ok := err.(*ParseError)
	if !ok {
		t.Fatalf("expected *ParseError, got %T", err)
	}
	if pe.Reason != "empty query" {
		t.Errorf("expected reason 'empty query', got %q", pe.Reason)
	}
}

func TestBuildQueryArgsPopulated(t *testing.T) {
	tests := []struct {
		name     string
		intent   Intent
		entities Entities
		wantArgs []string
	}{
		{"wins year populates year arg", IntentWins, Entities{Year: 2020}, []string{"year"}},
		{"wins driver populates driver_id", IntentWins, Entities{DriverID: 1}, []string{"driver_id"}},
		{"wins circuit populates circuit_id", IntentWins, Entities{CircuitID: 6}, []string{"circuit_id"}},
		{"points driver+year", IntentPoints, Entities{DriverID: 1, Year: 2019}, []string{"driver_id", "year"}},
		{"championship driver", IntentChampionship, Entities{DriverID: 3}, []string{"driver_id"}},
		{"championship constructor", IntentChampionship, Entities{ConstructorID: 6}, []string{"constructor_id"}},
		{"head to head", IntentHeadToHead, Entities{DriverID: 1, DriverID2: 2}, []string{"driver1", "driver2"}},
		{"teammates year", IntentTeammates, Entities{DriverID: 1, Year: 2020}, []string{"driver_id", "year"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, args := BuildQuery(tt.intent, tt.entities)
			for _, key := range tt.wantArgs {
				if _, ok := args[key]; !ok {
					t.Errorf("expected arg %q in args map, got %v", key, args)
				}
			}
		})
	}
}
