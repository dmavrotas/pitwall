package nlp

import (
	"fmt"
	"sort"
	"strings"

	"github.com/dmavrotas/gonlu"
	"github.com/dmavrotas/gonlu/nlu"
)

// learned is a small gonlu model trained on synthetic questions built from the
// dataset's own driver, team and circuit names. It never replaces the rules: it
// only fills in what they missed (see fill).
type learned struct {
	model *gonlu.Model
	ids   map[string]map[string]int // slot type -> display name -> entity ID
	skip  map[string]bool           // ordinary words that are never entity mentions
}

// minIntentConfidence is how sure the classifier must be before its intent is
// used for a question the rules could not classify.
const minIntentConfidence = 0.6

const (
	typeDriver      = "driver"
	typeConstructor = "constructor"
	typeCircuit     = "circuit"
)

func newLearned(ed *EntityDict) *learned {
	l := &learned{
		ids:  map[string]map[string]int{typeDriver: {}, typeConstructor: {}, typeCircuit: {}},
		skip: map[string]bool{},
	}
	entries := map[string][]string{}
	add := func(typ string, names []NamedID) {
		for _, n := range names {
			if _, dup := l.ids[typ][n.Name]; dup {
				continue // the more prominent entity was listed first and keeps the name
			}
			l.ids[typ][n.Name] = n.ID
			entries[typ] = append(entries[typ], n.Name)
		}
	}
	add(typeDriver, ed.DriverNames)
	add(typeConstructor, ed.ConstructorNames)
	add(typeCircuit, ed.CircuitNames)

	for w := range stopWords {
		l.skip[w] = true
	}
	for _, def := range intentDefs {
		for _, kw := range def.keywords {
			for _, w := range strings.Fields(kw) {
				l.skip[w] = true
			}
		}
	}

	model, err := gonlu.Train(trainingData(ed), gonlu.Options{Robust: true, Entries: entries})
	if err != nil {
		return nil // no data to learn from: stay rules-only
	}
	l.model = model
	return l
}

// fill completes e from the learned model, and the intent when the rules found
// none. Entities the rules already found are never changed. The learned intent
// is used only when the model also supplied a new entity: a classifier always
// returns some confident-looking label, even for gibberish, so trusting it on
// its own would swallow questions that should fall through to the AI fallback.
// It reports whether anything was added.
func (l *learned) fill(input string, e *Entities, intent *Intent) bool {
	res := l.model.Parse(input)
	entityAdded := false
	for _, s := range res.Slots {
		if s.Entry == "" || l.skip[s.Text] {
			continue
		}
		id := l.ids[s.Label][s.Entry]
		if id == 0 {
			continue
		}
		switch s.Label {
		case typeDriver:
			entityAdded = setFirstFree(&e.DriverID, &e.DriverID2, id) || entityAdded
		case typeConstructor:
			entityAdded = setFirstFree(&e.ConstructorID, &e.ConstructorID2, id) || entityAdded
		case typeCircuit:
			if e.CircuitID == 0 {
				e.CircuitID, entityAdded = id, true
			}
		}
	}
	if entityAdded && *intent == IntentUnknown && res.Intent != "" && res.Confidence >= minIntentConfidence {
		*intent = Intent(res.Intent)
	}
	return entityAdded
}

// setFirstFree stores id in the first empty of two slots, ignoring duplicates.
func setFirstFree(first, second *int, id int) bool {
	switch {
	case *first == id || *second == id:
		return false
	case *first == 0:
		*first = id
	case *second == 0:
		*second = id
	default:
		return false
	}
	return true
}

// questionTemplates are phrasings per intent. {d} {e} are drivers, {c} {b}
// teams, {k} a circuit, {y} a year.
var questionTemplates = map[Intent][]string{
	IntentWins: {
		"who has the most wins", "how many wins does {d} have", "how many races did {d} win",
		"{c} wins in {y}", "most wins at {k}", "wins by {d} in {y}", "who won the most races at {k}",
		"{d} wins at {k}", "how many times has {c} won",
	},
	IntentPoints: {
		"how many points did {d} score in {y}", "total points for {c}", "points scored by {d}",
		"{d} points in {y}", "{c} points in {y}",
	},
	IntentChampionship: {
		"how many championships does {d} have", "{c} world championships", "who won the championship in {y}",
		"how many titles has {d} won", "{c} titles",
	},
	IntentPoles: {
		"who got the most pole positions", "poles for {d}", "how many poles did {d} take",
		"pole positions at {k}", "{c} poles in {y}",
	},
	IntentDNF: {
		"what are the most common dnf reasons", "how many times did {d} retire", "dnf for {c}",
		"most common retirements", "{d} dnf in {y}",
	},
	IntentPitStops: {
		"fastest pit stops in {y}", "pit stops for {c}", "fastest pit stop by {c}", "{c} pit stop times in {y}",
	},
	IntentFastestLap: {
		"fastest laps at {k}", "fastest lap by {d}", "lap record at {k}", "{d} fastest laps in {y}",
	},
	IntentRaceResult: {
		"results of the {k} grand prix in {y}", "podiums in {y}", "podiums for {d}",
		"where did {d} finish at {k}", "{c} results in {y}",
	},
	IntentDriverInfo: {
		"who is {d}", "tell me about {d}", "when was {d} born", "{d} nationality",
	},
	IntentConstructorInfo: {
		"tell me about {c}", "what is {c}", "{c} team info",
	},
	IntentCircuitInfo: {
		"tell me about {k}", "where is {k}", "{k} circuit info",
	},
	IntentTeammates: {
		"who were the teammates of {d}", "who did {d} drive with", "teammates of {d}",
	},
	IntentHeadToHead: {
		"compare {d} vs {e}", "{d} versus {e}", "{d} against {e}", "{c} vs {b}", "compare {c} and {b}",
	},
	IntentSeasonOverview: {
		"season overview {y}", "{y} championship standings", "summary of the {y} season",
	},
}

var years = []string{"1988", "1994", "2004", "2008", "2012", "2019", "2021", "2023"}

// trainingData instantiates every template several times with prominent
// entities. Generation is deterministic: no randomness is involved.
func trainingData(ed *EntityDict) []nlu.Utterance {
	const copies = 6
	drivers := surnameAndFull(ed.DriverNames, 60)
	teams := names(ed.ConstructorNames, 24)
	circuits := names(ed.CircuitNames, 24)
	if len(drivers) == 0 || len(teams) < 2 || len(circuits) == 0 {
		// Tiny datasets (tests): fall back to whatever exists so every slot type is seen.
		drivers, teams, circuits = nonEmpty(drivers), nonEmpty2(teams), nonEmpty(circuits)
	}
	var out []nlu.Utterance
	for intent, templates := range questionTemplates {
		for ti, t := range templates {
			n := copies
			if !strings.ContainsAny(t, "{") {
				n = 1
			}
			for i := 0; i < n; i++ {
				k := ti*7 + i*3
				text := t
				text = strings.ReplaceAll(text, "{d}", mark(typeDriver, pick(drivers, k)))
				text = strings.ReplaceAll(text, "{e}", mark(typeDriver, pick(drivers, k+5)))
				text = strings.ReplaceAll(text, "{c}", mark(typeConstructor, pick(teams, k)))
				text = strings.ReplaceAll(text, "{b}", mark(typeConstructor, pick(teams, k+3)))
				text = strings.ReplaceAll(text, "{k}", mark(typeCircuit, pick(circuits, k)))
				text = strings.ReplaceAll(text, "{y}", pick(years, k))
				out = append(out, nlu.ParseAnnotated(text, string(intent)))
			}
		}
	}
	return sortedForDeterminism(out)
}

func mark(label, value string) string { return fmt.Sprintf("[%s : %s]", label, value) }

func pick(list []string, i int) string {
	if len(list) == 0 {
		return "unknown"
	}
	return list[i%len(list)]
}

func names(list []NamedID, limit int) []string {
	var out []string
	for i := 0; i < len(list) && i < limit; i++ {
		out = append(out, list[i].Name)
	}
	return out
}

// surnameAndFull returns, for each prominent driver, alternating the surname and
// the full name, because people type both.
func surnameAndFull(list []NamedID, limit int) []string {
	var out []string
	for i := 0; i < len(list) && i < limit; i++ {
		full := list[i].Name
		if i%2 == 0 {
			if f := strings.Fields(full); len(f) > 1 {
				full = f[len(f)-1]
			}
		}
		out = append(out, full)
	}
	return out
}

func nonEmpty(l []string) []string {
	if len(l) == 0 {
		return []string{"unknown"}
	}
	return l
}

func nonEmpty2(l []string) []string {
	for len(l) < 2 {
		l = append(l, "unknown")
	}
	return l
}

// sortedForDeterminism orders utterances by their text: map iteration above is
// random, and the learner is order-sensitive, so a stable order keeps models
// reproducible.
func sortedForDeterminism(us []nlu.Utterance) []nlu.Utterance {
	sortUtterances(us)
	return us
}

func sortUtterances(us []nlu.Utterance) {
	sort.Slice(us, func(i, j int) bool {
		a, b := strings.Join(us[i].Tokens, " "), strings.Join(us[j].Tokens, " ")
		if a != b {
			return a < b
		}
		return us[i].Intent < us[j].Intent
	})
}
