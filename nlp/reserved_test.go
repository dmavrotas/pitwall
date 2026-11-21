package nlp

import (
	"fmt"
	"testing"

	"github.com/dmavrotas/pitwall/loader"
	"github.com/dmavrotas/pitwall/models"
)

// collisionDataset contains real drivers whose names or codes are ordinary
// question words: Paddy Driver (ref "driver"), Markus Winkelhock (code "WIN")
// and Sergio Perez (code "PER").
func collisionDataset() *loader.Dataset {
	return &loader.Dataset{
		Drivers: []models.Driver{
			{ID: 1, Ref: "hamilton", Code: "HAM", Forename: "Lewis", Surname: "Hamilton"},
			{ID: 2, Ref: "driver", Forename: "Paddy", Surname: "Driver"},
			{ID: 3, Ref: "markus_winkelhock", Code: "WIN", Forename: "Markus", Surname: "Winkelhock"},
			{ID: 4, Ref: "perez", Code: "PER", Forename: "Sergio", Surname: "Perez"},
			{ID: 5, Ref: "stevens", Code: "STE", Forename: "Will", Surname: "Stevens"},
			{ID: 6, Ref: "may", Forename: "Michael", Surname: "May"},
			{ID: 7, Ref: "suzuki", Code: "SUZ", Forename: "Aguri", Surname: "Suzuki"},
		},
		Constructors: []models.Constructor{{ID: 6, Ref: "ferrari", Name: "Ferrari"}},
		Circuits: []models.Circuit{
			{ID: 14, Ref: "monza", Name: "Autodromo Nazionale di Monza", Location: "Monza"},
			{ID: 22, Ref: "suzuka", Name: "Suzuka Circuit", Location: "Suzuka"},
		},
		Results: []models.Result{
			{ID: 1, RaceID: 1, DriverID: 1, ConstructorID: 6, Position: 1},
			{ID: 2, RaceID: 1, DriverID: 2, ConstructorID: 6, Position: 2},
			{ID: 3, RaceID: 1, DriverID: 3, ConstructorID: 6, Position: 3},
			{ID: 4, RaceID: 1, DriverID: 4, ConstructorID: 6, Position: 4},
			{ID: 5, RaceID: 1, DriverID: 5, ConstructorID: 6, Position: 5},
			{ID: 6, RaceID: 1, DriverID: 6, ConstructorID: 6, Position: 6},
			{ID: 7, RaceID: 1, DriverID: 7, ConstructorID: 6, Position: 7},
		},
		Races: []models.Race{{ID: 1, Year: 2021, Round: 1, CircuitID: 14, Name: "Italian Grand Prix"}},
	}
}

func TestQuestionWordsAreNotMistakenForDrivers(t *testing.T) {
	ds := collisionDataset()
	for name, p := range map[string]*Parser{"rules": NewRulesOnlyParser(ds), "hybrid": NewParser(ds)} {
		t.Run(name, func(t *testing.T) { questionWordsAreNotDrivers(t, p) })
	}
}

func questionWordsAreNotDrivers(t *testing.T, p *Parser) {
	t.Helper()
	for _, q := range []string{
		"who is the best driver",
		"which driver has the most wins",
		"drivers with the most podiums",
		"average points per race",
		"points per race in 2021",
		"who will win the championship in 2021",
		"which team will win",
		"who may win the title",
		"how many wins can a driver get",
		"who would win more races",
	} {
		t.Run(q, func(t *testing.T) {
			plan, err := p.Parse(q)
			if err != nil {
				return // not understood at all is acceptable, a wrong driver is not
			}
			for _, k := range []string{"driver_id", "driver1", "driver2", "constructor_id"} {
				if v, ok := plan.Args[k]; ok {
					t.Errorf("%q resolved a bogus %s=%v (%s)", q, k, v, plan.Description)
				}
			}
		})
	}
}

func TestNamesStillResolveWhenSpelledOut(t *testing.T) {
	ds := collisionDataset()
	for name, p := range map[string]*Parser{"rules": NewRulesOnlyParser(ds), "hybrid": NewParser(ds)} {
		t.Run(name, func(t *testing.T) { namesStillResolve(t, p) })
	}
}

func namesStillResolve(t *testing.T, p *Parser) {
	t.Helper()
	tests := []struct {
		q    string
		want int
	}{
		{"paddy driver wins", 2},           // full name
		{"driver paddy wins", 2},           // reversed full name
		{"winkelhock wins", 3},             // surname
		{"markus winkelhock points", 3},    // full name
		{"perez points", 4},                // surname
		{"how many wins does ham have", 1}, // three-letter code that is not a question word
		{"will stevens wins", 5},           // the full name still works
		{"stevens wins", 5},                // and so does the surname
		{"michael may points", 6},          // "may" alone is a modal verb, the full name is not
	}
	for _, tt := range tests {
		t.Run(tt.q, func(t *testing.T) {
			plan, err := p.Parse(tt.q)
			if err != nil {
				t.Fatal(err)
			}
			if plan.Args["driver_id"] != tt.want {
				t.Errorf("driver_id = %v, want %d (args %v)", plan.Args["driver_id"], tt.want, plan.Args)
			}
		})
	}
}

// A word the rule dictionaries already know exactly is the rules' business. The
// learned layer must not reinterpret it as a different entity type, even when
// the tagger is unsure and a near-spelled name of another type exists ("suzuka"
// the circuit versus "Suzuki" the driver).
func TestLearnedLayerLeavesKnownWordsAlone(t *testing.T) {
	ds := collisionDataset()
	rules, hybrid := NewRulesOnlyParser(ds), NewParser(ds)
	for _, q := range []string{"suzuka", "tell me about suzuka", "suzuka wins", "who won at suzuka in 2021", "monza"} {
		t.Run(q, func(t *testing.T) {
			want, werr := rules.Parse(q)
			got, gerr := hybrid.Parse(q)
			if (werr == nil) != (gerr == nil) {
				t.Fatalf("error mismatch: rules=%v hybrid=%v", werr, gerr)
			}
			if werr != nil {
				return
			}
			if fmt.Sprint(got.Args) != fmt.Sprint(want.Args) || got.Learned {
				t.Errorf("hybrid changed a rules answer\nrules : %v\nhybrid: %v learned=%v", want.Args, got.Args, got.Learned)
			}
		})
	}
}

func TestLearnedKnownWords(t *testing.T) {
	l := newLearned(BuildEntityDict(collisionDataset()))
	for _, w := range []string{"suzuka", "monza", "hamilton", "suzuki", "ham"} {
		if !l.known(w) {
			t.Errorf("known(%q) = false, want true: the rule dictionaries have it", w)
		}
	}
	for _, w := range []string{"suzukaa", "hamliton", "monzza", "zzzzzz", ""} { //nolint:misspell // intentional typos
		if l.known(w) {
			t.Errorf("known(%q) = true, want false", w)
		}
	}
}
