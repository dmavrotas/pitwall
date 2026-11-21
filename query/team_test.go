package query

import (
	"testing"
	"time"

	"github.com/dmavrotas/pitwall/loader"
	"github.com/dmavrotas/pitwall/models"
	"github.com/dmavrotas/pitwall/nlp"
	"github.com/dmavrotas/pitwall/store"
)

// teamFixture has a driver whose chronologically FIRST result is for a different
// team than the one he mostly drove for, which is what exposes an arbitrary
// "team" pick in the leaderboards.
func teamFixture() *loader.Dataset {
	win := func(id, race, driver, team int) models.Result {
		return models.Result{ID: id, RaceID: race, DriverID: driver, ConstructorID: team, Position: 1, PositionOrder: 1, Points: 25, StatusID: 1}
	}
	second := func(id, race, driver int) models.Result {
		return models.Result{ID: id, RaceID: race, DriverID: driver, ConstructorID: 131, Position: 2, PositionOrder: 2, Points: 18, StatusID: 1}
	}
	return &loader.Dataset{
		Drivers: []models.Driver{
			{ID: 1, Ref: "hamilton", Forename: "Lewis", Surname: "Hamilton"},
			{ID: 2, Ref: "rosberg", Forename: "Nico", Surname: "Rosberg"},
		},
		Constructors: []models.Constructor{
			{ID: 1, Ref: "mclaren", Name: "McLaren"},
			{ID: 131, Ref: "mercedes", Name: "Mercedes"},
		},
		Circuits: []models.Circuit{{ID: 9, Ref: "silverstone", Name: "Silverstone Circuit", Location: "Silverstone"}},
		Races: []models.Race{
			{ID: 1, Year: 2007, Round: 1, CircuitID: 9, Name: "A", Date: time.Date(2007, 3, 18, 0, 0, 0, 0, time.UTC)},
			{ID: 2, Year: 2014, Round: 1, CircuitID: 9, Name: "B", Date: time.Date(2014, 3, 16, 0, 0, 0, 0, time.UTC)},
			{ID: 3, Year: 2015, Round: 1, CircuitID: 9, Name: "C", Date: time.Date(2015, 3, 15, 0, 0, 0, 0, time.UTC)},
			{ID: 4, Year: 2016, Round: 1, CircuitID: 9, Name: "D", Date: time.Date(2016, 3, 20, 0, 0, 0, 0, time.UTC)},
		},
		Results: []models.Result{
			win(1, 1, 1, 1), second(2, 1, 2), // 2007: Hamilton wins for McLaren
			win(3, 2, 1, 131), second(4, 2, 2), // 2014, 2015: Hamilton wins for Mercedes
			win(5, 3, 1, 131), second(6, 3, 2),
			win(7, 4, 2, 131), second(8, 4, 1), // 2016: Rosberg wins
		},
		Statuses: []models.Status{{ID: 1, Status: "Finished"}},
		Seasons:  []models.Season{{Year: 2007}, {Year: 2014}, {Year: 2015}, {Year: 2016}},
	}
}

func engineFor(t *testing.T, ds *loader.Dataset) *Engine {
	t.Helper()
	db, err := store.Load(ds)
	if err != nil {
		t.Fatalf("store.Load: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return NewEngine(db, nlp.NewRulesOnlyParser(ds))
}

// row returns the named columns of the first result row.
func row(t *testing.T, e *Engine, question string, cols ...string) map[string]string {
	t.Helper()
	res, err := e.Ask(question)
	if err != nil {
		t.Fatalf("Ask(%q): %v", question, err)
	}
	if len(res.Rows) == 0 {
		t.Fatalf("Ask(%q): no rows", question)
	}
	out := map[string]string{}
	for _, want := range cols {
		found := false
		for i, c := range res.Columns {
			if c == want {
				out[want], found = res.Rows[0][i], true
			}
		}
		if !found {
			t.Fatalf("Ask(%q): column %q not in %v", question, want, res.Columns)
		}
	}
	return out
}

func TestWinsLeaderboardShowsTheTeamWonWithMost(t *testing.T) {
	e := engineFor(t, teamFixture())
	got := row(t, e, "Who has the most wins?", "driver", "team", "wins")
	if got["driver"] != "Lewis Hamilton" || got["wins"] != "3" {
		t.Fatalf("unexpected leader: %v", got)
	}
	// 2 of his 3 wins were for Mercedes; his first (and the first row scanned) was McLaren.
	if got["team"] != "Mercedes" {
		t.Errorf("team = %q, want Mercedes (his team with most wins), not an arbitrary row's team", got["team"])
	}
}

func TestWinsLeaderboardTeamRespectsTheFilter(t *testing.T) {
	e := engineFor(t, teamFixture())
	got := row(t, e, "Who won the most races in 2007?", "driver", "team", "wins")
	if got["team"] != "McLaren" || got["wins"] != "1" {
		t.Errorf("2007 leader = %v, want McLaren / 1", got)
	}
}

func TestPointsLeaderboardShowsTheTeamScoredMostFor(t *testing.T) {
	e := engineFor(t, teamFixture())
	got := row(t, e, "Who has the most points?", "driver", "team", "total_points")
	// Hamilton scored 93 points in total (25 for McLaren, 68 for Mercedes); Rosberg scored 79.
	if got["driver"] != "Lewis Hamilton" || got["total_points"] != "93" {
		t.Fatalf("unexpected leader: %v", got)
	}
	if got["team"] != "Mercedes" {
		t.Errorf("team = %q, want Mercedes", got["team"])
	}
}

func TestPointsPerRaceKeepsTotalsAcrossTeams(t *testing.T) {
	e := engineFor(t, teamFixture())
	got := row(t, e, "Average points per race", "driver", "team", "points_per_race")
	// 93 points over 4 races: the per-race figure must use the driver's total, not one team's share.
	if got["driver"] != "Lewis Hamilton" || got["points_per_race"] != "23.25" {
		t.Errorf("got %v, want Lewis Hamilton at 23.25", got)
	}
	if got["team"] != "Mercedes" {
		t.Errorf("team = %q, want Mercedes", got["team"])
	}
}

func TestTeamTieGoesToTheMostRecentTeam(t *testing.T) {
	win := func(id, race, team int) models.Result {
		return models.Result{ID: id, RaceID: race, DriverID: 1, ConstructorID: team, Position: 1, PositionOrder: 1, Points: 25, StatusID: 1}
	}
	ds := &loader.Dataset{
		Drivers:      []models.Driver{{ID: 1, Ref: "hamilton", Forename: "Lewis", Surname: "Hamilton"}},
		Constructors: []models.Constructor{{ID: 1, Ref: "mclaren", Name: "McLaren"}, {ID: 131, Ref: "mercedes", Name: "Mercedes"}},
		Circuits:     []models.Circuit{{ID: 9, Ref: "silverstone", Name: "Silverstone Circuit", Location: "Silverstone"}},
		Races: []models.Race{
			{ID: 1, Year: 2007, Round: 1, CircuitID: 9, Name: "A", Date: time.Date(2007, 3, 18, 0, 0, 0, 0, time.UTC)},
			{ID: 2, Year: 2008, Round: 1, CircuitID: 9, Name: "B", Date: time.Date(2008, 3, 16, 0, 0, 0, 0, time.UTC)},
			{ID: 3, Year: 2014, Round: 1, CircuitID: 9, Name: "C", Date: time.Date(2014, 3, 16, 0, 0, 0, 0, time.UTC)},
			{ID: 4, Year: 2015, Round: 1, CircuitID: 9, Name: "D", Date: time.Date(2015, 3, 15, 0, 0, 0, 0, time.UTC)},
		},
		// Two wins for each team; the first row scanned is McLaren, the most recent win is Mercedes.
		Results:  []models.Result{win(1, 1, 1), win(2, 2, 1), win(3, 3, 131), win(4, 4, 131)},
		Statuses: []models.Status{{ID: 1, Status: "Finished"}},
		Seasons:  []models.Season{{Year: 2007}, {Year: 2008}, {Year: 2014}, {Year: 2015}},
	}
	got := row(t, engineFor(t, ds), "Who has the most wins?", "driver", "team", "wins")
	if got["wins"] != "4" {
		t.Fatalf("wins = %q, want 4 (totals must span both teams)", got["wins"])
	}
	if got["team"] != "Mercedes" {
		t.Errorf("tie: team = %q, want Mercedes (the most recent of the tied teams)", got["team"])
	}
}
