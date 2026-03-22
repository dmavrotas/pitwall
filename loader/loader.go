package loader

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/dmavrotas/pitwall/models"
)

// Dataset holds all loaded F1 data tables in memory.
type Dataset struct {
	Circuits              []models.Circuit
	Constructors          []models.Constructor
	Drivers               []models.Driver
	Races                 []models.Race
	Results               []models.Result
	LapTimes              []models.LapTime
	PitStops              []models.PitStop
	Qualifying            []models.Qualifying
	DriverStandings       []models.DriverStanding
	ConstructorStandings  []models.ConstructorStanding
	ConstructorResults    []models.ConstructorResult
	Statuses              []models.Status
	Seasons               []models.Season
	SprintResults         []models.SprintResult
}

// LoadAll loads all CSV files from the given directory into a Dataset.
func LoadAll(dir string) (*Dataset, error) {
	ds := &Dataset{}

	loaders := []struct {
		file string
		fn   func(string) error
	}{
		{"circuits.csv", func(p string) error { return loadCSV(p, func(r []string) error { return ds.parseCircuit(r) }) }},
		{"constructors.csv", func(p string) error { return loadCSV(p, func(r []string) error { return ds.parseConstructor(r) }) }},
		{"drivers.csv", func(p string) error { return loadCSV(p, func(r []string) error { return ds.parseDriver(r) }) }},
		{"races.csv", func(p string) error { return loadCSV(p, func(r []string) error { return ds.parseRace(r) }) }},
		{"results.csv", func(p string) error { return loadCSV(p, func(r []string) error { return ds.parseResult(r) }) }},
		{"lap_times.csv", func(p string) error { return loadCSV(p, func(r []string) error { return ds.parseLapTime(r) }) }},
		{"pit_stops.csv", func(p string) error { return loadCSV(p, func(r []string) error { return ds.parsePitStop(r) }) }},
		{"qualifying.csv", func(p string) error { return loadCSV(p, func(r []string) error { return ds.parseQualifying(r) }) }},
		{"driver_standings.csv", func(p string) error { return loadCSV(p, func(r []string) error { return ds.parseDriverStanding(r) }) }},
		{"constructor_standings.csv", func(p string) error { return loadCSV(p, func(r []string) error { return ds.parseConstructorStanding(r) }) }},
		{"constructor_results.csv", func(p string) error { return loadCSV(p, func(r []string) error { return ds.parseConstructorResult(r) }) }},
		{"status.csv", func(p string) error { return loadCSV(p, func(r []string) error { return ds.parseStatus(r) }) }},
		{"seasons.csv", func(p string) error { return loadCSV(p, func(r []string) error { return ds.parseSeason(r) }) }},
		{"sprint_results.csv", func(p string) error { return loadCSV(p, func(r []string) error { return ds.parseSprintResult(r) }) }},
	}

	for _, l := range loaders {
		path := dir + "/" + l.file
		if _, err := os.Stat(path); os.IsNotExist(err) {
			fmt.Printf("  [skip] %s not found\n", l.file)
			continue
		}
		fmt.Printf("  [load] %s\n", l.file)
		if err := l.fn(path); err != nil {
			return nil, fmt.Errorf("loading %s: %w", l.file, err)
		}
	}

	return ds, nil
}

// Summary prints a summary of loaded data counts.
func (ds *Dataset) Summary() {
	fmt.Println("\n=== Pitwall Dataset Summary ===")
	fmt.Printf("  Circuits:              %d\n", len(ds.Circuits))
	fmt.Printf("  Constructors:          %d\n", len(ds.Constructors))
	fmt.Printf("  Drivers:               %d\n", len(ds.Drivers))
	fmt.Printf("  Races:                 %d\n", len(ds.Races))
	fmt.Printf("  Results:               %d\n", len(ds.Results))
	fmt.Printf("  Lap Times:             %d\n", len(ds.LapTimes))
	fmt.Printf("  Pit Stops:             %d\n", len(ds.PitStops))
	fmt.Printf("  Qualifying:            %d\n", len(ds.Qualifying))
	fmt.Printf("  Driver Standings:      %d\n", len(ds.DriverStandings))
	fmt.Printf("  Constructor Standings: %d\n", len(ds.ConstructorStandings))
	fmt.Printf("  Constructor Results:   %d\n", len(ds.ConstructorResults))
	fmt.Printf("  Statuses:              %d\n", len(ds.Statuses))
	fmt.Printf("  Seasons:               %d\n", len(ds.Seasons))
	fmt.Printf("  Sprint Results:        %d\n", len(ds.SprintResults))
	fmt.Println("===============================")
}

func loadCSV(path string, parseFn func([]string) error) error {
	f, err := os.Open(path) //nolint:gosec // path comes from trusted internal config, not user input
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	reader := csv.NewReader(f)
	// Skip header row
	if _, err := reader.Read(); err != nil {
		return fmt.Errorf("reading header: %w", err)
	}

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("reading row: %w", err)
		}
		if err := parseFn(record); err != nil {
			return fmt.Errorf("parsing row %v: %w", record, err)
		}
	}
	return nil
}

// Helper parsers

func atoi(s string) int {
	s = strings.TrimSpace(s)
	if s == "" || s == "\\N" {
		return 0
	}
	v, _ := strconv.Atoi(s)
	return v
}

func atof(s string) float64 {
	s = strings.TrimSpace(s)
	if s == "" || s == "\\N" {
		return 0
	}
	v, _ := strconv.ParseFloat(s, 64)
	return v
}

func parseDate(s string) time.Time {
	s = strings.TrimSpace(s)
	if s == "" || s == "\\N" {
		return time.Time{}
	}
	t, _ := time.Parse("2006-01-02", s)
	return t
}

func clean(s string) string {
	s = strings.TrimSpace(s)
	if s == "\\N" {
		return ""
	}
	return s
}

func (ds *Dataset) parseCircuit(r []string) error {
	if len(r) < 9 {
		return fmt.Errorf("expected 9 fields, got %d", len(r))
	}
	ds.Circuits = append(ds.Circuits, models.Circuit{
		ID:       atoi(r[0]),
		Ref:      clean(r[1]),
		Name:     clean(r[2]),
		Location: clean(r[3]),
		Country:  clean(r[4]),
		Lat:      atof(r[5]),
		Lng:      atof(r[6]),
		Alt:      atoi(r[7]),
		URL:      clean(r[8]),
	})
	return nil
}

func (ds *Dataset) parseConstructor(r []string) error {
	if len(r) < 5 {
		return fmt.Errorf("expected 5 fields, got %d", len(r))
	}
	ds.Constructors = append(ds.Constructors, models.Constructor{
		ID:          atoi(r[0]),
		Ref:         clean(r[1]),
		Name:        clean(r[2]),
		Nationality: clean(r[3]),
		URL:         clean(r[4]),
	})
	return nil
}

func (ds *Dataset) parseDriver(r []string) error {
	if len(r) < 9 {
		return fmt.Errorf("expected 9 fields, got %d", len(r))
	}
	ds.Drivers = append(ds.Drivers, models.Driver{
		ID:          atoi(r[0]),
		Ref:         clean(r[1]),
		Number:      atoi(r[2]),
		Code:        clean(r[3]),
		Forename:    clean(r[4]),
		Surname:     clean(r[5]),
		DOB:         parseDate(r[6]),
		Nationality: clean(r[7]),
		URL:         clean(r[8]),
	})
	return nil
}

func (ds *Dataset) parseRace(r []string) error {
	if len(r) < 8 {
		return fmt.Errorf("expected at least 8 fields, got %d", len(r))
	}
	ds.Races = append(ds.Races, models.Race{
		ID:        atoi(r[0]),
		Year:      atoi(r[1]),
		Round:     atoi(r[2]),
		CircuitID: atoi(r[3]),
		Name:      clean(r[4]),
		Date:      parseDate(r[5]),
		Time:      clean(r[6]),
		URL:       clean(r[7]),
	})
	return nil
}

func (ds *Dataset) parseResult(r []string) error {
	if len(r) < 18 {
		return fmt.Errorf("expected 18 fields, got %d", len(r))
	}
	ds.Results = append(ds.Results, models.Result{
		ID:              atoi(r[0]),
		RaceID:          atoi(r[1]),
		DriverID:        atoi(r[2]),
		ConstructorID:   atoi(r[3]),
		Number:          atoi(r[4]),
		Grid:            atoi(r[5]),
		Position:        atoi(r[6]),
		PositionText:    clean(r[7]),
		PositionOrder:   atoi(r[8]),
		Points:          atof(r[9]),
		Laps:            atoi(r[10]),
		Time:            clean(r[11]),
		Milliseconds:    atoi(r[12]),
		FastestLap:      atoi(r[13]),
		Rank:            atoi(r[14]),
		FastestLapTime:  clean(r[15]),
		FastestLapSpeed: clean(r[16]),
		StatusID:        atoi(r[17]),
	})
	return nil
}

func (ds *Dataset) parseLapTime(r []string) error {
	if len(r) < 6 {
		return fmt.Errorf("expected 6 fields, got %d", len(r))
	}
	ds.LapTimes = append(ds.LapTimes, models.LapTime{
		RaceID:       atoi(r[0]),
		DriverID:     atoi(r[1]),
		Lap:          atoi(r[2]),
		Position:     atoi(r[3]),
		Time:         clean(r[4]),
		Milliseconds: atoi(r[5]),
	})
	return nil
}

func (ds *Dataset) parsePitStop(r []string) error {
	if len(r) < 7 {
		return fmt.Errorf("expected 7 fields, got %d", len(r))
	}
	ds.PitStops = append(ds.PitStops, models.PitStop{
		RaceID:       atoi(r[0]),
		DriverID:     atoi(r[1]),
		Stop:         atoi(r[2]),
		Lap:          atoi(r[3]),
		Time:         clean(r[4]),
		Duration:     clean(r[5]),
		Milliseconds: atoi(r[6]),
	})
	return nil
}

func (ds *Dataset) parseQualifying(r []string) error {
	if len(r) < 9 {
		return fmt.Errorf("expected 9 fields, got %d", len(r))
	}
	ds.Qualifying = append(ds.Qualifying, models.Qualifying{
		ID:            atoi(r[0]),
		RaceID:        atoi(r[1]),
		DriverID:      atoi(r[2]),
		ConstructorID: atoi(r[3]),
		Number:        atoi(r[4]),
		Position:      atoi(r[5]),
		Q1:            clean(r[6]),
		Q2:            clean(r[7]),
		Q3:            clean(r[8]),
	})
	return nil
}

func (ds *Dataset) parseDriverStanding(r []string) error {
	if len(r) < 7 {
		return fmt.Errorf("expected 7 fields, got %d", len(r))
	}
	ds.DriverStandings = append(ds.DriverStandings, models.DriverStanding{
		ID:           atoi(r[0]),
		RaceID:       atoi(r[1]),
		DriverID:     atoi(r[2]),
		Points:       atof(r[3]),
		Position:     atoi(r[4]),
		PositionText: clean(r[5]),
		Wins:         atoi(r[6]),
	})
	return nil
}

func (ds *Dataset) parseConstructorStanding(r []string) error {
	if len(r) < 7 {
		return fmt.Errorf("expected 7 fields, got %d", len(r))
	}
	ds.ConstructorStandings = append(ds.ConstructorStandings, models.ConstructorStanding{
		ID:            atoi(r[0]),
		RaceID:        atoi(r[1]),
		ConstructorID: atoi(r[2]),
		Points:        atof(r[3]),
		Position:      atoi(r[4]),
		PositionText:  clean(r[5]),
		Wins:          atoi(r[6]),
	})
	return nil
}

func (ds *Dataset) parseConstructorResult(r []string) error {
	if len(r) < 5 {
		return fmt.Errorf("expected 5 fields, got %d", len(r))
	}
	ds.ConstructorResults = append(ds.ConstructorResults, models.ConstructorResult{
		ID:            atoi(r[0]),
		RaceID:        atoi(r[1]),
		ConstructorID: atoi(r[2]),
		Points:        atof(r[3]),
		Status:        clean(r[4]),
	})
	return nil
}

func (ds *Dataset) parseStatus(r []string) error {
	if len(r) < 2 {
		return fmt.Errorf("expected 2 fields, got %d", len(r))
	}
	ds.Statuses = append(ds.Statuses, models.Status{
		ID:     atoi(r[0]),
		Status: clean(r[1]),
	})
	return nil
}

func (ds *Dataset) parseSeason(r []string) error {
	if len(r) < 2 {
		return fmt.Errorf("expected 2 fields, got %d", len(r))
	}
	ds.Seasons = append(ds.Seasons, models.Season{
		Year: atoi(r[0]),
		URL:  clean(r[1]),
	})
	return nil
}

func (ds *Dataset) parseSprintResult(r []string) error {
	if len(r) < 16 {
		return fmt.Errorf("expected 16 fields, got %d", len(r))
	}
	ds.SprintResults = append(ds.SprintResults, models.SprintResult{
		ID:             atoi(r[0]),
		RaceID:         atoi(r[1]),
		DriverID:       atoi(r[2]),
		ConstructorID:  atoi(r[3]),
		Number:         atoi(r[4]),
		Grid:           atoi(r[5]),
		Position:       atoi(r[6]),
		PositionText:   clean(r[7]),
		PositionOrder:  atoi(r[8]),
		Points:         atof(r[9]),
		Laps:           atoi(r[10]),
		Time:           clean(r[11]),
		Milliseconds:   atoi(r[12]),
		FastestLap:     atoi(r[13]),
		FastestLapTime: clean(r[14]),
		StatusID:       atoi(r[15]),
	})
	return nil
}
