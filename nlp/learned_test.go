package nlp

import (
	"fmt"
	"reflect"
	"testing"
)

// The hybrid must never change an answer the rules already got right.
func TestHybridMatchesRulesOnExampleQuestions(t *testing.T) {
	ds := testDataset()
	rules, hybrid := NewRulesOnlyParser(ds), NewParser(ds)
	for _, q := range ExampleQuestions() {
		t.Run(q, func(t *testing.T) {
			want, wantErr := rules.Parse(q)
			got, gotErr := hybrid.Parse(q)
			if (wantErr == nil) != (gotErr == nil) {
				t.Fatalf("error mismatch: rules=%v hybrid=%v", wantErr, gotErr)
			}
			if wantErr != nil {
				return
			}
			if got.SQL != want.SQL || !reflect.DeepEqual(got.Args, want.Args) || got.Intent != want.Intent {
				t.Errorf("hybrid changed the plan\nrules : %v %v\nhybrid: %v %v", want.Intent, want.Args, got.Intent, got.Args)
			}
			if got.Learned {
				t.Error("plan marked Learned although the rules already had everything")
			}
		})
	}
}

func TestHybridRepairsMisspelledDriver(t *testing.T) {
	ds := testDataset()
	// The rule engine does not know the misspelled name: it silently answers the generic question.
	rulesPlan, err := NewRulesOnlyParser(ds).Parse("how many wins did hamliton have") //nolint:misspell // intentional typo
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := rulesPlan.Args["driver_id"]; ok {
		t.Fatal("test premise broken: rules already resolved the typo")
	}
	got, err := NewParser(ds).Parse("how many wins did hamliton have") //nolint:misspell // intentional typo
	if err != nil {
		t.Fatal(err)
	}
	if got.Args["driver_id"] != 1 { // Lewis Hamilton (2 results) outranks Duncan Hamilton (0)
		t.Errorf("driver_id = %v, want 1; plan %+v", got.Args["driver_id"], got)
	}
	if !got.Learned {
		t.Error("plan should be marked Learned")
	}
}

func TestHybridRepairsMisspelledConstructorAndCircuit(t *testing.T) {
	ds := testDataset()
	got, err := NewParser(ds).Parse("ferari wins at silverstoen") //nolint:misspell // intentional typos
	if err != nil {
		t.Fatal(err)
	}
	if got.Args["constructor_id"] != 6 {
		t.Errorf("constructor_id = %v, want 6 (args %v)", got.Args["constructor_id"], got.Args)
	}
	if got.Args["circuit_id"] != 9 {
		t.Errorf("circuit_id = %v, want 9 (args %v)", got.Args["circuit_id"], got.Args)
	}
}

func TestHybridFillsSecondDriverInComparison(t *testing.T) {
	got, err := NewParser(testDataset()).Parse("compare verstappen vs hamliton") //nolint:misspell // intentional typo
	if err != nil {
		t.Fatal(err)
	}
	if got.Intent != IntentHeadToHead {
		t.Errorf("intent = %v, want head_to_head", got.Intent)
	}
	ids := map[interface{}]bool{got.Args["driver1"]: true, got.Args["driver2"]: true}
	if !ids[2] || !ids[1] {
		t.Errorf("expected drivers 2 and 1 in args, got %v", got.Args)
	}
}

func TestHybridIgnoresIntentWords(t *testing.T) {
	// Ordinary words must never be linked to a driver, however close a surname is.
	for _, q := range []string{"who has the most wins", "fastest pit stops in 2023", "season overview 2010", "most common dnf reasons"} {
		got, err := NewParser(testDataset()).Parse(q)
		if err != nil {
			t.Fatalf("%q: %v", q, err)
		}
		if got.Learned {
			t.Errorf("%q was altered by the learned layer: %+v", q, got.Args)
		}
	}
}

func TestEntityDictOrdersNamesByPopularity(t *testing.T) {
	ed := BuildEntityDict(testDataset())
	if len(ed.DriverNames) == 0 || ed.DriverNames[0].ID != 1 {
		t.Fatalf("most popular driver should come first, got %+v", ed.DriverNames)
	}
}

// Constructor mentions must resolve in the same sentence frames that people use
// for drivers; the tagger must rely on the name, not only on the frame.
func TestHybridRepairsMisspelledConstructorInDriverStyleFrames(t *testing.T) {
	p := NewParser(testDataset())
	typos := map[string]int{"ferari": 6, "mclaern": 1, "mercedse": 131, "red bul": 9} //nolint:misspell // intentional typos
	// Only frames whose query builders filter by team: the poles and DNF
	// builders ignore constructors entirely (even for a correct spelling), so
	// the plan cannot carry a team for them.
	frames := []string{
		"how many wins did %s have",
		"how many races did %s win",
		"how many points did %s score in 2019",
		"%s wins in 2004",
		"tell me about %s",
	}
	for typo, id := range typos {
		for _, f := range frames {
			q := fmt.Sprintf(f, typo)
			t.Run(q, func(t *testing.T) {
				plan, err := p.Parse(q)
				if err != nil {
					t.Fatalf("parse error: %v", err)
				}
				if plan.Args["constructor_id"] != id {
					t.Errorf("constructor_id = %v, want %d (args %v, %s)", plan.Args["constructor_id"], id, plan.Args, plan.Description)
				}
				if _, bogus := plan.Args["driver_id"]; bogus {
					t.Errorf("a driver was invented for a team question: %v", plan.Args)
				}
			})
		}
	}
}

func TestResolveLetsTheNameDecideTheType(t *testing.T) {
	l := newLearned(BuildEntityDict(testDataset()))
	tests := []struct {
		name, taggedAs, text, wantType string
		wantID                         int
		wantOK                         bool
	}{
		{"tagged right", typeDriver, "hamliton", typeDriver, 1, true},                              //nolint:misspell // intentional typo
		{"tagger says driver, name is a team", typeDriver, "ferari", typeConstructor, 6, true},     //nolint:misspell // intentional typo
		{"tagger says team, name is a driver", typeConstructor, "hamliton", typeDriver, 1, true},   //nolint:misspell // intentional typo
		{"tagger says driver, name is a circuit", typeDriver, "silverstoen", typeCircuit, 9, true}, //nolint:misspell // intentional typo
		{"nothing matches", typeDriver, "zzzzzz", "", 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			typ, entry, ok := l.resolve(tt.taggedAs, tt.text)
			id := l.ids[typ][entry]
			if ok != tt.wantOK || typ != tt.wantType || id != tt.wantID {
				t.Errorf("resolve(%q, %q) = (%q, %q -> id %d, %v), want (%q, id %d, %v)", tt.taggedAs, tt.text, typ, entry, id, ok, tt.wantType, tt.wantID, tt.wantOK)
			}
		})
	}
}

// A one-word team name must not steal a typo from a multi-word driver entry
// just because gonlu scores a single-word match on a longer entry one step worse.
func TestResolveTaggedTypeKeepsItsHeadStart(t *testing.T) {
	l := newLearned(BuildEntityDict(testDataset()))
	// "prost" is exact for a driver surname; it must stay a driver even if tagged as a team.
	if typ, entry, ok := l.resolve(typeDriver, "prost"); !ok || typ != typeDriver || entry != "Alain Prost" {
		t.Errorf("resolve(driver, prost) = (%q, %q, %v)", typ, entry, ok)
	}
}
