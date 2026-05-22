package nlp

import (
	"fmt"
	"strings"

	"github.com/dmavrotas/pitwall/loader"
)

// EntityDict holds lookup maps from name variants to IDs.
type EntityDict struct {
	Drivers      map[string]int // "hamilton" -> 1, "lewis hamilton" -> 1, "ham" -> 1
	Constructors map[string]int // "ferrari" -> 6, "red bull" -> 9
	Circuits     map[string]int // "monza" -> 14, "silverstone" -> 9
}

// BuildEntityDict scans the dataset and builds name-to-ID lookup maps.
// For ambiguous names (e.g., "Hamilton"), the driver with the most race results wins.
func BuildEntityDict(ds *loader.Dataset) *EntityDict {
	ed := &EntityDict{
		Drivers:      make(map[string]int),
		Constructors: make(map[string]int),
		Circuits:     make(map[string]int),
	}

	// Count results per driver to prioritize popular drivers for ambiguous names
	resultCount := make(map[int]int)
	for i := range ds.Results {
		resultCount[ds.Results[i].DriverID]++
	}

	// Process in two passes: first full names (always set), then surnames (only if more popular)
	for i := range ds.Drivers {
		d := ds.Drivers[i]
		id := d.ID
		fullName := strings.ToLower(d.Forename + " " + d.Surname)
		ed.Drivers[fullName] = id
		ed.Drivers[strings.ToLower(d.Surname+" "+d.Forename)] = id
		if d.Code != "" {
			ed.Drivers[strings.ToLower(d.Code)] = id
		}
		if d.Ref != "" {
			ed.Drivers[strings.ToLower(d.Ref)] = id
		}
	}
	// For surname-only matches, the driver with most results wins
	surnameOwner := make(map[string]int) // surname -> best driver ID
	surnameCount := make(map[string]int) // surname -> best result count
	for i := range ds.Drivers {
		surname := strings.ToLower(ds.Drivers[i].Surname)
		d := ds.Drivers[i]
		count := resultCount[d.ID]
		if count > surnameCount[surname] {
			surnameOwner[surname] = d.ID
			surnameCount[surname] = count
		}
	}
	for surname, id := range surnameOwner {
		ed.Drivers[surname] = id
	}

	for _, c := range ds.Constructors {
		id := c.ID
		name := strings.ToLower(c.Name)
		ed.Constructors[name] = id
		ed.Constructors[strings.ToLower(c.Ref)] = id
		// Handle multi-word: "red bull" -> also match "redbull"
		if strings.Contains(name, " ") {
			ed.Constructors[strings.ReplaceAll(name, " ", "")] = id
		}
	}

	for _, c := range ds.Circuits {
		id := c.ID
		ed.Circuits[strings.ToLower(c.Name)] = id
		ed.Circuits[strings.ToLower(c.Ref)] = id
		ed.Circuits[strings.ToLower(c.Location)] = id
	}

	return ed
}

// ExtractEntities finds driver, constructor, circuit, and year references in the input.
func (ed *EntityDict) ExtractEntities(tokens []string) Entities {
	e := Entities{}
	input := strings.Join(tokens, " ")

	// Collect all year-looking tokens (4-digit, 1950-2030)
	var years []int
	var yearIdx []int
	for i, t := range tokens {
		if len(t) == 4 && t[0] >= '1' && t[0] <= '2' {
			var y int
			if _, err := fmt.Sscanf(t, "%d", &y); err == nil && y >= 1950 && y <= 2030 {
				years = append(years, y)
				yearIdx = append(yearIdx, i)
			}
		}
	}

	switch len(years) {
	case 0:
		// no year — leave all year fields zero
	case 1:
		y := years[0]
		prev := ""
		if yearIdx[0] > 0 {
			prev = tokens[yearIdx[0]-1]
		}
		switch prev {
		case "since":
			e.YearFrom = y
		case "after":
			e.YearFrom = y + 1
		case "before":
			e.YearTo = y - 1
		case "until", "till":
			e.YearTo = y
		default:
			e.Year = y
		}
	default:
		// Two or more years → treat as a range (sorted)
		lo, hi := years[0], years[1]
		if lo > hi {
			lo, hi = hi, lo
		}
		e.YearFrom = lo
		e.YearTo = hi
	}

	// Try longest match first for multi-word names
	// Check 3-word, then 2-word, then single-word tokens
	matched := make(map[int]bool) // token indices already consumed

	for n := 3; n >= 1; n-- {
		for i := 0; i <= len(tokens)-n; i++ {
			skip := false
			for j := i; j < i+n; j++ {
				if matched[j] {
					skip = true
					break
				}
			}
			if skip {
				continue
			}

			phrase := strings.Join(tokens[i:i+n], " ")

			if id, ok := ed.Drivers[phrase]; ok {
				if e.DriverID == 0 {
					e.DriverID = id
				} else if e.DriverID2 == 0 && id != e.DriverID {
					e.DriverID2 = id
				}
				for j := i; j < i+n; j++ {
					matched[j] = true
				}
				continue
			}

			if id, ok := ed.Constructors[phrase]; ok {
				if e.ConstructorID == 0 {
					e.ConstructorID = id
				} else if e.ConstructorID2 == 0 && id != e.ConstructorID {
					e.ConstructorID2 = id
				}
				for j := i; j < i+n; j++ {
					matched[j] = true
				}
				continue
			}

			if id, ok := ed.Circuits[phrase]; ok {
				e.CircuitID = id
				for j := i; j < i+n; j++ {
					matched[j] = true
				}
			}
		}
	}

	// Detect modifier words. These are deliberately matched on the post-stopword
	// token stream so they can sit anywhere in the query.
	for i, t := range tokens {
		switch t {
		case "average", "avg":
			e.Average = true
		case "first":
			e.Ordinal = "first"
		case "last":
			e.Ordinal = "last"
		case "worst":
			e.Ordinal = "worst"
		case "per":
			if i+1 < len(tokens) && tokens[i+1] == "race" {
				e.PerRace = true
			}
		case "standings", "rankings", "ranking":
			e.ShowFullStandings = true
		}
	}

	// Disambiguate: if a name matched both driver and constructor,
	// use surrounding context from the full input
	_ = input

	return e
}

// Entities holds the extracted entity IDs from a query.
type Entities struct {
	DriverID       int
	DriverID2      int // for comparison queries
	ConstructorID  int
	ConstructorID2 int // for comparison queries
	CircuitID      int
	Year           int    // single-year filter; mutually exclusive with YearFrom/YearTo
	YearFrom       int    // inclusive lower bound for range filters
	YearTo         int    // inclusive upper bound for range filters
	Average           bool   // "average" → use AVG instead of SUM where applicable
	PerRace           bool   // "per race" → normalize aggregate by COUNT(DISTINCT race_id)
	Ordinal           string // "first", "last", "worst" — select a single row instead of aggregate
	ShowFullStandings bool   // "standings"/"rankings" present → return full table instead of a single row
}
