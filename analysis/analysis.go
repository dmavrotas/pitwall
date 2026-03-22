package analysis

import (
	"fmt"
	"sort"

	"github.com/dmavrotas/pitwall/loader"
)

// Analyzer provides analytical queries over the F1 dataset.
type Analyzer struct {
	ds *loader.Dataset
}

// New creates a new Analyzer for the given dataset.
func New(ds *loader.Dataset) *Analyzer {
	return &Analyzer{ds: ds}
}

// DriverWins returns a sorted list of drivers by total race wins (descending).
func (a *Analyzer) DriverWins(top int) {
	wins := make(map[int]int)
	for i := range a.ds.Results {
		if a.ds.Results[i].Position == 1 {
			wins[a.ds.Results[i].DriverID]++
		}
	}

	driverIndex := a.driverIndex()

	type entry struct {
		Name string
		Wins int
	}

	entries := make([]entry, 0, len(wins))
	for driverID, w := range wins {
		d := driverIndex[driverID]
		entries = append(entries, entry{
			Name: fmt.Sprintf("%s %s", d.Forename, d.Surname),
			Wins: w,
		})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Wins > entries[j].Wins })

	fmt.Printf("\n--- Top %d Drivers by Race Wins ---\n", top)
	for i, e := range entries {
		if i >= top {
			break
		}
		fmt.Printf("  %2d. %-25s %d wins\n", i+1, e.Name, e.Wins)
	}
}

// ConstructorChampionships returns constructors sorted by number of championship titles.
func (a *Analyzer) ConstructorChampionships(top int) {
	// Find the last race of each season
	lastRace := make(map[int]int) // year -> raceID
	for _, race := range a.ds.Races {
		if existing, ok := lastRace[race.Year]; !ok || race.Round > a.raceRound(existing) {
			lastRace[race.Year] = race.ID
		}
	}

	// Count constructor championships (position 1 at season's last race)
	titles := make(map[int]int)
	for _, cs := range a.ds.ConstructorStandings {
		for _, raceID := range lastRace {
			if cs.RaceID == raceID && cs.Position == 1 {
				titles[cs.ConstructorID]++
			}
		}
	}

	constructorIndex := a.constructorIndex()

	type entry struct {
		Name   string
		Titles int
	}

	entries := make([]entry, 0, len(titles))
	for cID, t := range titles {
		c := constructorIndex[cID]
		entries = append(entries, entry{Name: c.Name, Titles: t})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Titles > entries[j].Titles })

	fmt.Printf("\n--- Top %d Constructors by Championship Titles ---\n", top)
	for i, e := range entries {
		if i >= top {
			break
		}
		fmt.Printf("  %2d. %-25s %d titles\n", i+1, e.Name, e.Titles)
	}
}

// AveragePitStopByYear calculates the average pit stop duration per season.
func (a *Analyzer) AveragePitStopByYear() {
	type accumulator struct {
		totalMs int
		count   int
	}

	raceYear := make(map[int]int)
	for _, race := range a.ds.Races {
		raceYear[race.ID] = race.Year
	}

	yearly := make(map[int]*accumulator)
	for _, ps := range a.ds.PitStops {
		year := raceYear[ps.RaceID]
		if year == 0 || ps.Milliseconds == 0 {
			continue
		}
		if _, ok := yearly[year]; !ok {
			yearly[year] = &accumulator{}
		}
		yearly[year].totalMs += ps.Milliseconds
		yearly[year].count++
	}

	type entry struct {
		Year int
		Avg  float64
	}
	entries := make([]entry, 0, len(yearly))
	for year, acc := range yearly {
		entries = append(entries, entry{Year: year, Avg: float64(acc.totalMs) / float64(acc.count) / 1000.0})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Year < entries[j].Year })

	fmt.Println("\n--- Average Pit Stop Duration by Year (seconds) ---")
	for _, e := range entries {
		bar := ""
		for i := 0; i < int(e.Avg); i++ {
			bar += "#"
		}
		fmt.Printf("  %d  %5.2fs  %s\n", e.Year, e.Avg, bar)
	}
}

// DNFStats shows the most common reasons for not finishing a race.
func (a *Analyzer) DNFStats(top int) {
	statusIndex := make(map[int]string)
	for _, s := range a.ds.Statuses {
		statusIndex[s.ID] = s.Status
	}

	counts := make(map[string]int)
	for i := range a.ds.Results {
		r := a.ds.Results[i]
		status := statusIndex[r.StatusID]
		if status != "Finished" && status != "" && r.Position == 0 {
			counts[status]++
		}
	}

	type entry struct {
		Status string
		Count  int
	}
	entries := make([]entry, 0, len(counts))
	for s, c := range counts {
		entries = append(entries, entry{Status: s, Count: c})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Count > entries[j].Count })

	fmt.Printf("\n--- Top %d DNF Reasons ---\n", top)
	for i, e := range entries {
		if i >= top {
			break
		}
		fmt.Printf("  %2d. %-25s %d occurrences\n", i+1, e.Status, e.Count)
	}
}

// helpers

func (a *Analyzer) driverIndex() map[int]struct{ Forename, Surname string } {
	idx := make(map[int]struct{ Forename, Surname string })
	for i := range a.ds.Drivers {
		idx[a.ds.Drivers[i].ID] = struct{ Forename, Surname string }{a.ds.Drivers[i].Forename, a.ds.Drivers[i].Surname}
	}
	return idx
}

func (a *Analyzer) constructorIndex() map[int]struct{ Name string } {
	idx := make(map[int]struct{ Name string })
	for _, c := range a.ds.Constructors {
		idx[c.ID] = struct{ Name string }{c.Name}
	}
	return idx
}

func (a *Analyzer) raceRound(raceID int) int {
	for _, r := range a.ds.Races {
		if r.ID == raceID {
			return r.Round
		}
	}
	return 0
}
