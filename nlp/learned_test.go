package nlp

import (
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
