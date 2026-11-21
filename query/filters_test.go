package query

import (
	"strconv"
	"testing"
	"time"

	"github.com/dmavrotas/pitwall/loader"
	"github.com/dmavrotas/pitwall/models"
	"github.com/dmavrotas/pitwall/nlp"
	"github.com/dmavrotas/pitwall/store"
)

// filterFixture splits poles and retirements across two teams and two circuits,
// so that a missing team or circuit filter gives visibly wrong counts.
//
//	race 1 Monza    2020: pole Hamilton (Mercedes)   retirements: Hamilton (Engine), Rosberg (Crash)
//	race 2 Monza    2021: pole Hamilton (Mercedes)   retirements: Rosberg (Engine)
//	race 3 Monaco   2021: pole Rosberg  (Ferrari)    retirements: Rosberg (Engine), Hamilton (Engine)
//	race 4 Monaco   2022: pole Hamilton (Ferrari)    retirements: Hamilton (Crash)
func filterFixture() *loader.Dataset {
	date := func(y int) time.Time { return time.Date(y, 6, 1, 0, 0, 0, 0, time.UTC) }
	pole := func(id, race, driver, team int) models.Qualifying {
		return models.Qualifying{ID: id, RaceID: race, DriverID: driver, ConstructorID: team, Position: 1}
	}
	slow := func(id, race, driver, team int) models.Qualifying {
		return models.Qualifying{ID: id, RaceID: race, DriverID: driver, ConstructorID: team, Position: 2}
	}
	dnf := func(id, race, driver, team, status int) models.Result {
		return models.Result{ID: id, RaceID: race, DriverID: driver, ConstructorID: team, Position: 0, PositionOrder: 9, StatusID: status}
	}
	const mercedes, ferrari = 131, 6
	const (
		hamilton, rosberg = 1, 2
		engine, crash     = 2, 3
	)
	return &loader.Dataset{
		Drivers: []models.Driver{
			{ID: hamilton, Ref: "hamilton", Forename: "Lewis", Surname: "Hamilton"},
			{ID: rosberg, Ref: "rosberg", Forename: "Nico", Surname: "Rosberg"},
		},
		Constructors: []models.Constructor{{ID: mercedes, Ref: "mercedes", Name: "Mercedes"}, {ID: ferrari, Ref: "ferrari", Name: "Ferrari"}},
		Circuits: []models.Circuit{
			{ID: 14, Ref: "monza", Name: "Autodromo Nazionale di Monza", Location: "Monza"},
			{ID: 6, Ref: "monaco", Name: "Circuit de Monaco", Location: "Monte-Carlo"},
		},
		Races: []models.Race{
			{ID: 1, Year: 2020, Round: 1, CircuitID: 14, Name: "Italian GP", Date: date(2020)},
			{ID: 2, Year: 2021, Round: 1, CircuitID: 14, Name: "Italian GP", Date: date(2021)},
			{ID: 3, Year: 2021, Round: 2, CircuitID: 6, Name: "Monaco GP", Date: date(2021)},
			{ID: 4, Year: 2022, Round: 1, CircuitID: 6, Name: "Monaco GP", Date: date(2022)},
		},
		Qualifying: []models.Qualifying{
			pole(1, 1, hamilton, mercedes), slow(2, 1, rosberg, mercedes),
			pole(3, 2, hamilton, mercedes), slow(4, 2, rosberg, mercedes),
			pole(5, 3, rosberg, ferrari), slow(6, 3, hamilton, ferrari),
			pole(7, 4, hamilton, ferrari), slow(8, 4, rosberg, ferrari),
		},
		Results: []models.Result{
			dnf(1, 1, hamilton, mercedes, engine), dnf(2, 1, rosberg, mercedes, crash),
			dnf(3, 2, rosberg, mercedes, engine),
			dnf(4, 3, rosberg, ferrari, engine), dnf(5, 3, hamilton, ferrari, engine),
			dnf(6, 4, hamilton, ferrari, crash),
		},
		Statuses: []models.Status{{ID: 1, Status: "Finished"}, {ID: engine, Status: "Engine"}, {ID: crash, Status: "Collision"}},
		Seasons:  []models.Season{{Year: 2020}, {Year: 2021}, {Year: 2022}},
	}
}

func filterEngine(t *testing.T) *Engine {
	t.Helper()
	ds := filterFixture()
	db, err := store.Load(ds)
	if err != nil {
		t.Fatalf("store.Load: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return NewEngine(db, nlp.NewRulesOnlyParser(ds))
}

// table asks a question and returns the (first column -> last column) mapping of
// the result rows, which is "driver -> count" for poles and "reason -> count" for DNFs.
func table(t *testing.T, e *Engine, question string) map[string]int {
	t.Helper()
	res, err := e.Ask(question)
	if err != nil {
		t.Fatalf("Ask(%q): %v", question, err)
	}
	out := map[string]int{}
	for _, r := range res.Rows {
		n, convErr := strconv.Atoi(r[len(r)-1])
		if convErr != nil {
			t.Fatalf("Ask(%q): last column %q is not a count: %v", question, r[len(r)-1], convErr)
		}
		out[r[0]] = n
	}
	return out
}

func equalCounts(a, b map[string]int) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

func TestPolesRespectTeamAndCircuitFilters(t *testing.T) {
	e := filterEngine(t)
	tests := []struct {
		question string
		want     map[string]int
	}{
		{"Who got the most pole positions?", map[string]int{"Lewis Hamilton": 3, "Nico Rosberg": 1}},
		{"pole positions for ferrari", map[string]int{"Lewis Hamilton": 1, "Nico Rosberg": 1}},
		{"how many poles did mercedes take", map[string]int{"Lewis Hamilton": 2}},
		{"pole positions at monza", map[string]int{"Lewis Hamilton": 2}},
		{"pole positions at monaco", map[string]int{"Lewis Hamilton": 1, "Nico Rosberg": 1}},
		{"ferrari poles at monaco in 2021", map[string]int{"Nico Rosberg": 1}},
	}
	for _, tt := range tests {
		t.Run(tt.question, func(t *testing.T) {
			if got := table(t, e, tt.question); !equalCounts(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDNFsRespectTeamAndCircuitFilters(t *testing.T) {
	e := filterEngine(t)
	tests := []struct {
		question string
		want     map[string]int
	}{
		{"most common dnf reasons", map[string]int{"Engine": 4, "Collision": 2}},
		{"dnf for ferrari", map[string]int{"Engine": 2, "Collision": 1}},
		{"dnf for mercedes", map[string]int{"Engine": 2, "Collision": 1}},
		{"dnf reasons at monza", map[string]int{"Engine": 2, "Collision": 1}},
		{"dnf reasons at monaco in 2022", map[string]int{"Collision": 1}},
		{"ferrari dnf at monaco", map[string]int{"Engine": 2, "Collision": 1}},
	}
	for _, tt := range tests {
		t.Run(tt.question, func(t *testing.T) {
			if got := table(t, e, tt.question); !equalCounts(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}
