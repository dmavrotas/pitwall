package loader

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadCSV(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		wantRows int
		wantErr  bool
	}{
		{
			name:     "valid csv",
			content:  "id,name\n1,Alice\n2,Bob\n",
			wantRows: 2,
		},
		{
			name:     "empty csv with header",
			content:  "id,name\n",
			wantRows: 0,
		},
		{
			name:     "single row",
			content:  "id,name\n1,Alice\n",
			wantRows: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "test.csv")
			if err := os.WriteFile(path, []byte(tt.content), 0o600); err != nil {
				t.Fatal(err)
			}

			rowCount := 0
			err := loadCSV(path, func(r []string) error {
				rowCount++
				return nil
			})

			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if rowCount != tt.wantRows {
				t.Errorf("got %d rows, want %d", rowCount, tt.wantRows)
			}
		})
	}
}

func TestLoadCSVFileNotFound(t *testing.T) {
	err := loadCSV("/nonexistent/file.csv", func(r []string) error { return nil })
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestAtoi(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"42", 42},
		{"0", 0},
		{"-1", -1},
		{"", 0},
		{"\\N", 0},
		{"  7  ", 7},
		{"abc", 0},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := atoi(tt.input)
			if got != tt.want {
				t.Errorf("atoi(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestAtof(t *testing.T) {
	tests := []struct {
		input string
		want  float64
	}{
		{"3.14", 3.14},
		{"0", 0},
		{"", 0},
		{"\\N", 0},
		{" 2.5 ", 2.5},
		{"abc", 0},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := atof(tt.input)
			if got != tt.want {
				t.Errorf("atof(%q) = %f, want %f", tt.input, got, tt.want)
			}
		})
	}
}

func TestClean(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hello", "hello"},
		{"  hello  ", "hello"},
		{"\\N", ""},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := clean(tt.input)
			if got != tt.want {
				t.Errorf("clean(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseDate(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantStr string
	}{
		{"valid date", "1985-01-07", "1985-01-07"},
		{"null value", "\\N", "0001-01-01"},
		{"empty", "", "0001-01-01"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseDate(tt.input)
			gotStr := got.Format("2006-01-02")
			if gotStr != tt.wantStr {
				t.Errorf("parseDate(%q) = %s, want %s", tt.input, gotStr, tt.wantStr)
			}
		})
	}
}

func TestParseCircuit(t *testing.T) {
	tests := []struct {
		name    string
		record  []string
		wantErr bool
	}{
		{
			name:   "valid circuit",
			record: []string{"1", "albert_park", "Albert Park", "Melbourne", "Australia", "-37.8497", "144.968", "10", "http://example.com"},
		},
		{
			name:    "too few fields",
			record:  []string{"1", "albert_park"},
			wantErr: true,
		},
		{
			name:   "null values handled",
			record: []string{"1", "ref", "Name", "Loc", "Country", "\\N", "\\N", "\\N", "\\N"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ds := &Dataset{}
			err := ds.parseCircuit(tt.record)
			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if !tt.wantErr && len(ds.Circuits) != 1 {
				t.Errorf("expected 1 circuit, got %d", len(ds.Circuits))
			}
		})
	}
}

func TestParseConstructor(t *testing.T) {
	tests := []struct {
		name    string
		record  []string
		wantErr bool
	}{
		{
			name:   "valid constructor",
			record: []string{"6", "ferrari", "Ferrari", "Italian", "http://example.com"},
		},
		{
			name:    "too few fields",
			record:  []string{"6", "ferrari"},
			wantErr: true,
		},
		{
			name:   "null values",
			record: []string{"1", "ref", "Name", "\\N", "\\N"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ds := &Dataset{}
			err := ds.parseConstructor(tt.record)
			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if !tt.wantErr && len(ds.Constructors) != 1 {
				t.Errorf("expected 1 constructor, got %d", len(ds.Constructors))
			}
		})
	}
}

func TestParseDriver(t *testing.T) {
	tests := []struct {
		name    string
		record  []string
		wantErr bool
	}{
		{
			name:   "valid driver",
			record: []string{"1", "hamilton", "44", "HAM", "Lewis", "Hamilton", "1985-01-07", "British", "http://example.com"},
		},
		{
			name:    "too few fields",
			record:  []string{"1"},
			wantErr: true,
		},
		{
			name:   "null number and code",
			record: []string{"99", "ref", "\\N", "\\N", "First", "Last", "\\N", "Nationality", "url"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ds := &Dataset{}
			err := ds.parseDriver(tt.record)
			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestParseRace(t *testing.T) {
	tests := []struct {
		name    string
		record  []string
		wantErr bool
	}{
		{
			name:   "valid race",
			record: []string{"1", "2020", "1", "9", "British Grand Prix", "2020-08-02", "14:10:00", "http://example.com"},
		},
		{
			name:    "too few fields",
			record:  []string{"1", "2020"},
			wantErr: true,
		},
		{
			name:   "null time and url",
			record: []string{"1", "2020", "1", "9", "Race", "\\N", "\\N", "\\N"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ds := &Dataset{}
			err := ds.parseRace(tt.record)
			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if !tt.wantErr && len(ds.Races) != 1 {
				t.Errorf("expected 1 race, got %d", len(ds.Races))
			}
		})
	}
}

func TestParseResult(t *testing.T) {
	tests := []struct {
		name    string
		record  []string
		wantErr bool
	}{
		{
			name:   "valid result",
			record: []string{"1", "1", "1", "1", "44", "1", "1", "1", "1", "25", "58", "1:30.000", "5400000", "42", "1", "1:28.000", "220.5", "1"},
		},
		{
			name:    "too few fields",
			record:  []string{"1", "2", "3"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ds := &Dataset{}
			err := ds.parseResult(tt.record)
			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestParseLapTime(t *testing.T) {
	tests := []struct {
		name    string
		record  []string
		wantErr bool
	}{
		{
			name:   "valid lap time",
			record: []string{"1", "1", "5", "1", "1:32.456", "92456"},
		},
		{
			name:    "too few fields",
			record:  []string{"1", "1"},
			wantErr: true,
		},
		{
			name:   "null time",
			record: []string{"1", "1", "5", "1", "\\N", "\\N"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ds := &Dataset{}
			err := ds.parseLapTime(tt.record)
			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if !tt.wantErr && len(ds.LapTimes) != 1 {
				t.Errorf("expected 1 lap time, got %d", len(ds.LapTimes))
			}
		})
	}
}

func TestParsePitStop(t *testing.T) {
	tests := []struct {
		name    string
		record  []string
		wantErr bool
	}{
		{
			name:   "valid pit stop",
			record: []string{"1", "1", "1", "15", "14:32:00", "23.456", "23456"},
		},
		{
			name:    "too few fields",
			record:  []string{"1"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ds := &Dataset{}
			err := ds.parsePitStop(tt.record)
			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if !tt.wantErr && len(ds.PitStops) != 1 {
				t.Errorf("expected 1 pit stop, got %d", len(ds.PitStops))
			}
		})
	}
}

func TestParseQualifying(t *testing.T) {
	tests := []struct {
		name    string
		record  []string
		wantErr bool
	}{
		{
			name:   "valid qualifying",
			record: []string{"1", "1", "1", "6", "44", "1", "1:28.000", "1:27.500", "1:26.000"},
		},
		{
			name:    "too few fields",
			record:  []string{"1", "1", "1"},
			wantErr: true,
		},
		{
			name:   "null Q2 Q3",
			record: []string{"1", "1", "1", "6", "44", "20", "1:30.000", "\\N", "\\N"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ds := &Dataset{}
			err := ds.parseQualifying(tt.record)
			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if !tt.wantErr && len(ds.Qualifying) != 1 {
				t.Errorf("expected 1 qualifying, got %d", len(ds.Qualifying))
			}
		})
	}
}

func TestParseDriverStanding(t *testing.T) {
	tests := []struct {
		name    string
		record  []string
		wantErr bool
	}{
		{
			name:   "valid standing",
			record: []string{"1", "1", "1", "25", "1", "1", "1"},
		},
		{
			name:    "too few fields",
			record:  []string{"1", "1"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ds := &Dataset{}
			err := ds.parseDriverStanding(tt.record)
			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if !tt.wantErr && len(ds.DriverStandings) != 1 {
				t.Errorf("expected 1 standing, got %d", len(ds.DriverStandings))
			}
		})
	}
}

func TestParseConstructorStanding(t *testing.T) {
	tests := []struct {
		name    string
		record  []string
		wantErr bool
	}{
		{
			name:   "valid standing",
			record: []string{"1", "1", "6", "200", "1", "1", "10"},
		},
		{
			name:    "too few fields",
			record:  []string{"1"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ds := &Dataset{}
			err := ds.parseConstructorStanding(tt.record)
			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if !tt.wantErr && len(ds.ConstructorStandings) != 1 {
				t.Errorf("expected 1 standing, got %d", len(ds.ConstructorStandings))
			}
		})
	}
}

func TestParseConstructorResult(t *testing.T) {
	tests := []struct {
		name    string
		record  []string
		wantErr bool
	}{
		{
			name:   "valid result",
			record: []string{"1", "1", "6", "25", "Finished"},
		},
		{
			name:    "too few fields",
			record:  []string{"1", "1"},
			wantErr: true,
		},
		{
			name:   "null status",
			record: []string{"1", "1", "6", "0", "\\N"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ds := &Dataset{}
			err := ds.parseConstructorResult(tt.record)
			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if !tt.wantErr && len(ds.ConstructorResults) != 1 {
				t.Errorf("expected 1 result, got %d", len(ds.ConstructorResults))
			}
		})
	}
}

func TestParseStatus(t *testing.T) {
	tests := []struct {
		name    string
		record  []string
		wantErr bool
	}{
		{
			name:   "valid status",
			record: []string{"1", "Finished"},
		},
		{
			name:    "too few fields",
			record:  []string{"1"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ds := &Dataset{}
			err := ds.parseStatus(tt.record)
			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if !tt.wantErr && len(ds.Statuses) != 1 {
				t.Errorf("expected 1 status, got %d", len(ds.Statuses))
			}
		})
	}
}

func TestParseSeason(t *testing.T) {
	tests := []struct {
		name    string
		record  []string
		wantErr bool
	}{
		{
			name:   "valid season",
			record: []string{"2020", "http://example.com"},
		},
		{
			name:    "too few fields",
			record:  []string{"2020"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ds := &Dataset{}
			err := ds.parseSeason(tt.record)
			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if !tt.wantErr && len(ds.Seasons) != 1 {
				t.Errorf("expected 1 season, got %d", len(ds.Seasons))
			}
		})
	}
}

func TestParseSprintResult(t *testing.T) {
	tests := []struct {
		name    string
		record  []string
		wantErr bool
	}{
		{
			name:   "valid sprint result",
			record: []string{"1", "1", "1", "6", "44", "1", "1", "1", "1", "8", "17", "30:00.000", "1800000", "5", "1:28.000", "1"},
		},
		{
			name:    "too few fields",
			record:  []string{"1", "1", "1"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ds := &Dataset{}
			err := ds.parseSprintResult(tt.record)
			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if !tt.wantErr && len(ds.SprintResults) != 1 {
				t.Errorf("expected 1 sprint result, got %d", len(ds.SprintResults))
			}
		})
	}
}

func TestLoadAllMissingDir(t *testing.T) {
	ds, err := LoadAll(t.TempDir())
	if err != nil {
		t.Fatalf("LoadAll on empty dir should not error, got: %v", err)
	}
	if len(ds.Drivers) != 0 {
		t.Error("expected empty dataset for missing CSVs")
	}
}

func TestLoadAllWithCSVFiles(t *testing.T) {
	dir := t.TempDir()

	files := map[string]string{
		"circuits.csv":              "circuitId,circuitRef,name,location,country,lat,lng,alt,url\n1,monza,Monza,Monza,Italy,45.6,9.28,162,http://x\n",
		"constructors.csv":         "constructorId,constructorRef,name,nationality,url\n6,ferrari,Ferrari,Italian,http://x\n",
		"drivers.csv":              "driverId,driverRef,number,code,forename,surname,dob,nationality,url\n1,hamilton,44,HAM,Lewis,Hamilton,1985-01-07,British,http://x\n",
		"races.csv":                "raceId,year,round,circuitId,name,date,time,url\n1,2020,1,1,Italian GP,2020-09-06,14:10:00,http://x\n",
		"results.csv":              "resultId,raceId,driverId,constructorId,number,grid,position,positionText,positionOrder,points,laps,time,milliseconds,fastestLap,rank,fastestLapTime,fastestLapSpeed,statusId\n1,1,1,6,44,1,1,1,1,25,53,1:30,5400000,42,1,1:28,220,1\n",
		"lap_times.csv":            "raceId,driverId,lap,position,time,milliseconds\n1,1,1,1,1:32.000,92000\n",
		"pit_stops.csv":            "raceId,driverId,stop,lap,time,duration,milliseconds\n1,1,1,15,14:32:00,23.456,23456\n",
		"qualifying.csv":           "qualifyId,raceId,driverId,constructorId,number,position,q1,q2,q3\n1,1,1,6,44,1,1:28,1:27,1:26\n",
		"driver_standings.csv":     "driverStandingsId,raceId,driverId,points,position,positionText,wins\n1,1,1,25,1,1,1\n",
		"constructor_standings.csv": "constructorStandingsId,raceId,constructorId,points,position,positionText,wins\n1,1,6,25,1,1,1\n",
		"constructor_results.csv":  "constructorResultsId,raceId,constructorId,points,status\n1,1,6,25,Finished\n",
		"status.csv":               "statusId,status\n1,Finished\n",
		"seasons.csv":              "year,url\n2020,http://x\n",
		"sprint_results.csv":       "sprintResultId,raceId,driverId,constructorId,number,grid,position,positionText,positionOrder,points,laps,time,milliseconds,fastestLap,fastestLapTime,statusId\n1,1,1,6,44,1,1,1,1,8,17,30:00,1800000,5,1:28,1\n",
	}

	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	ds, err := LoadAll(dir)
	if err != nil {
		t.Fatalf("LoadAll error: %v", err)
	}

	checks := []struct {
		name  string
		count int
	}{
		{"Circuits", len(ds.Circuits)},
		{"Constructors", len(ds.Constructors)},
		{"Drivers", len(ds.Drivers)},
		{"Races", len(ds.Races)},
		{"Results", len(ds.Results)},
		{"LapTimes", len(ds.LapTimes)},
		{"PitStops", len(ds.PitStops)},
		{"Qualifying", len(ds.Qualifying)},
		{"DriverStandings", len(ds.DriverStandings)},
		{"ConstructorStandings", len(ds.ConstructorStandings)},
		{"ConstructorResults", len(ds.ConstructorResults)},
		{"Statuses", len(ds.Statuses)},
		{"Seasons", len(ds.Seasons)},
		{"SprintResults", len(ds.SprintResults)},
	}

	for _, c := range checks {
		if c.count != 1 {
			t.Errorf("%s: expected 1, got %d", c.name, c.count)
		}
	}
}
