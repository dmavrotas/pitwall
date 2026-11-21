package nlp

import (
	"strings"
	"testing"

	"github.com/dmavrotas/pitwall/loader"
	"github.com/dmavrotas/pitwall/models"
)

// monthDataset has the two real collisions with month names: the March
// constructor and the driver Michael May.
func monthDataset() *loader.Dataset {
	return &loader.Dataset{
		Drivers: []models.Driver{
			{ID: 1, Ref: "hamilton", Code: "HAM", Forename: "Lewis", Surname: "Hamilton"},
			{ID: 2, Ref: "may", Forename: "Michael", Surname: "May"},
		},
		Constructors: []models.Constructor{
			{ID: 6, Ref: "ferrari", Name: "Ferrari"},
			{ID: 77, Ref: "march", Name: "March"},
		},
		Circuits: []models.Circuit{{ID: 14, Ref: "monza", Name: "Autodromo Nazionale di Monza", Location: "Monza"}},
		Results: []models.Result{
			{ID: 1, RaceID: 1, DriverID: 1, ConstructorID: 6, Position: 1},
			{ID: 2, RaceID: 1, DriverID: 2, ConstructorID: 77, Position: 2},
		},
		Races: []models.Race{{ID: 1, Year: 1982, Round: 1, CircuitID: 14, Name: "Italian Grand Prix"}},
	}
}

func TestMonthAfterAPrepositionIsAMonthNotAnEntity(t *testing.T) {
	ds := monthDataset()
	for name, p := range map[string]*Parser{"rules": NewRulesOnlyParser(ds), "hybrid": NewParser(ds)} {
		for _, q := range []string{"who won in march 1982", "wins during march", "most wins in May", "poles in march 1982"} {
			t.Run(name+"/"+q, func(t *testing.T) {
				plan, err := p.Parse(q)
				if err != nil {
					t.Fatal(err)
				}
				for _, k := range []string{"constructor_id", "driver_id"} {
					if v, ok := plan.Args[k]; ok {
						t.Errorf("%q read a month as an entity: %s=%v", q, k, v)
					}
				}
				if !strings.Contains(plan.Description, "month") {
					t.Errorf("description %q should say that the month was not applied", plan.Description)
				}
			})
		}
	}
}

func TestMonthNamedEntitiesStillResolveWithoutAPreposition(t *testing.T) {
	p := NewRulesOnlyParser(monthDataset())
	plan, err := p.Parse("march wins in 1982") // the constructor, no preposition before the word
	if err != nil {
		t.Fatal(err)
	}
	if plan.Args["constructor_id"] != 77 {
		t.Errorf("constructor_id = %v, want 77 (args %v)", plan.Args["constructor_id"], plan.Args)
	}
	if strings.Contains(plan.Description, "month") {
		t.Errorf("a constructor mention must not be reported as a month: %q", plan.Description)
	}
}

func TestMonthWordsNeverChangeTheRestOfTheQuery(t *testing.T) {
	p := NewRulesOnlyParser(monthDataset())
	with, err := p.Parse("hamilton wins in june 1982")
	if err != nil {
		t.Fatal(err)
	}
	without, err := p.Parse("hamilton wins in 1982")
	if err != nil {
		t.Fatal(err)
	}
	if with.SQL != without.SQL || with.Args["driver_id"] != without.Args["driver_id"] || with.Args["year"] != without.Args["year"] {
		t.Errorf("the month changed the query\nwith   : %v\nwithout: %v", with.Args, without.Args)
	}
	if !strings.Contains(with.Description, "June") {
		t.Errorf("description %q should name the ignored month", with.Description)
	}
}
