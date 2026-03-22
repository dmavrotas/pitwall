package store

import (
	"testing"

	"github.com/dmavrotas/pitwall/loader"
	"github.com/dmavrotas/pitwall/models"
)

func testDataset() *loader.Dataset {
	return &loader.Dataset{
		Drivers: []models.Driver{
			{ID: 1, Ref: "hamilton", Code: "HAM", Forename: "Lewis", Surname: "Hamilton"},
			{ID: 2, Ref: "verstappen", Code: "VER", Forename: "Max", Surname: "Verstappen"},
		},
		Constructors: []models.Constructor{
			{ID: 131, Ref: "mercedes", Name: "Mercedes", Nationality: "German"},
			{ID: 9, Ref: "red_bull", Name: "Red Bull", Nationality: "Austrian"},
		},
		Circuits: []models.Circuit{
			{ID: 9, Ref: "silverstone", Name: "Silverstone Circuit", Location: "Silverstone", Country: "UK", Lat: 52.0786, Lng: -1.01694, Alt: 153},
		},
		Races: []models.Race{
			{ID: 1, Year: 2020, Round: 1, CircuitID: 9, Name: "British Grand Prix"},
			{ID: 2, Year: 2020, Round: 2, CircuitID: 9, Name: "70th Anniversary Grand Prix"},
			{ID: 3, Year: 2021, Round: 1, CircuitID: 9, Name: "British Grand Prix"},
		},
		Results: []models.Result{
			{ID: 1, RaceID: 1, DriverID: 1, ConstructorID: 131, Position: 1, PositionOrder: 1, Points: 25, Laps: 52, StatusID: 1},
			{ID: 2, RaceID: 1, DriverID: 2, ConstructorID: 9, Position: 2, PositionOrder: 2, Points: 18, Laps: 52, StatusID: 1},
			{ID: 3, RaceID: 2, DriverID: 2, ConstructorID: 9, Position: 1, PositionOrder: 1, Points: 25, Laps: 52, StatusID: 1},
			{ID: 4, RaceID: 2, DriverID: 1, ConstructorID: 131, Position: 2, PositionOrder: 2, Points: 18, Laps: 52, StatusID: 1},
			{ID: 5, RaceID: 3, DriverID: 1, ConstructorID: 131, Position: 1, PositionOrder: 1, Points: 25, Laps: 52, StatusID: 1},
		},
		Statuses: []models.Status{
			{ID: 1, Status: "Finished"},
		},
		Seasons: []models.Season{
			{Year: 2020},
			{Year: 2021},
		},
	}
}

func setupTestDB(t *testing.T) *DB {
	t.Helper()
	db, err := Load(testDataset())
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestLoad(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		t.Fatal("Load() returned nil")
	}
}

func TestQuerySimple(t *testing.T) {
	db := setupTestDB(t)

	tests := []struct {
		name     string
		sql      string
		args     map[string]interface{}
		wantCols int
		wantRows int
	}{
		{
			name:     "count drivers",
			sql:      "SELECT COUNT(*) AS cnt FROM drivers",
			args:     map[string]interface{}{},
			wantCols: 1,
			wantRows: 1,
		},
		{
			name:     "all races",
			sql:      "SELECT id, name FROM races ORDER BY id",
			args:     map[string]interface{}{},
			wantCols: 2,
			wantRows: 3,
		},
		{
			name:     "races in year",
			sql:      "SELECT id FROM races WHERE year = :year",
			args:     map[string]interface{}{"year": 2020},
			wantCols: 1,
			wantRows: 2,
		},
		{
			name:     "driver wins",
			sql:      "SELECT COUNT(*) FROM results WHERE driver_id = :driver_id AND position = 1",
			args:     map[string]interface{}{"driver_id": 1},
			wantCols: 1,
			wantRows: 1,
		},
		{
			name:     "no results",
			sql:      "SELECT * FROM results WHERE race_id = 999",
			args:     map[string]interface{}{},
			wantCols: 18,
			wantRows: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cols, rows, err := db.Query(tt.sql, tt.args)
			if err != nil {
				t.Fatalf("Query error: %v", err)
			}
			if len(cols) != tt.wantCols {
				t.Errorf("got %d columns, want %d", len(cols), tt.wantCols)
			}
			if len(rows) != tt.wantRows {
				t.Errorf("got %d rows, want %d", len(rows), tt.wantRows)
			}
		})
	}
}

func TestQueryNamedParams(t *testing.T) {
	db := setupTestDB(t)

	tests := []struct {
		name     string
		sql      string
		args     map[string]interface{}
		wantRows int
	}{
		{
			name:     "single param",
			sql:      "SELECT * FROM results WHERE driver_id = :driver_id",
			args:     map[string]interface{}{"driver_id": 1},
			wantRows: 3,
		},
		{
			name:     "multiple params",
			sql:      "SELECT * FROM results WHERE driver_id = :driver_id AND race_id = :race_id",
			args:     map[string]interface{}{"driver_id": 1, "race_id": 1},
			wantRows: 1,
		},
		{
			name:     "same param twice",
			sql:      "SELECT * FROM results WHERE driver_id = :did OR constructor_id = :did",
			args:     map[string]interface{}{"did": 131},
			wantRows: 3,
		},
		{
			name:     "IN clause with two params",
			sql:      "SELECT * FROM results WHERE driver_id IN (:d1, :d2)",
			args:     map[string]interface{}{"d1": 1, "d2": 2},
			wantRows: 5,
		},
		{
			name:     "no params",
			sql:      "SELECT COUNT(*) FROM seasons",
			args:     map[string]interface{}{},
			wantRows: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, rows, err := db.Query(tt.sql, tt.args)
			if err != nil {
				t.Fatalf("Query error: %v", err)
			}
			if len(rows) != tt.wantRows {
				t.Errorf("got %d rows, want %d", len(rows), tt.wantRows)
			}
		})
	}
}

func TestQueryJoins(t *testing.T) {
	db := setupTestDB(t)

	tests := []struct {
		name      string
		sql       string
		args      map[string]interface{}
		wantRows  int
		checkFunc func([][]string) error
	}{
		{
			name: "driver wins count",
			sql: `SELECT d.forename || ' ' || d.surname AS driver, COUNT(*) AS wins
				   FROM results r JOIN drivers d ON d.id = r.driver_id
				   WHERE r.position = 1 GROUP BY r.driver_id ORDER BY wins DESC`,
			args:     map[string]interface{}{},
			wantRows: 2,
		},
		{
			name: "wins by year",
			sql: `SELECT d.surname, COUNT(*) AS wins
				   FROM results r JOIN drivers d ON d.id = r.driver_id
				   JOIN races ra ON ra.id = r.race_id
				   WHERE r.position = 1 AND ra.year = :year
				   GROUP BY r.driver_id ORDER BY wins DESC`,
			args:     map[string]interface{}{"year": 2020},
			wantRows: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, rows, err := db.Query(tt.sql, tt.args)
			if err != nil {
				t.Fatalf("Query error: %v", err)
			}
			if len(rows) != tt.wantRows {
				t.Errorf("got %d rows, want %d", len(rows), tt.wantRows)
			}
		})
	}
}

func TestLoadEmptyDataset(t *testing.T) {
	db, err := Load(&loader.Dataset{})
	if err != nil {
		t.Fatalf("Load empty dataset error: %v", err)
	}
	defer func() { _ = db.Close() }()

	_, rows, err := db.Query("SELECT COUNT(*) FROM drivers", map[string]interface{}{})
	if err != nil {
		t.Fatal(err)
	}
	if rows[0][0] != "0" {
		t.Errorf("expected 0 drivers, got %s", rows[0][0])
	}
}

func TestQueryInvalidSQL(t *testing.T) {
	db := setupTestDB(t)

	_, _, err := db.Query("SELECT * FROM nonexistent_table", map[string]interface{}{})
	if err == nil {
		t.Error("expected error for invalid SQL")
	}
}

func TestQueryNullValues(t *testing.T) {
	db := setupTestDB(t)

	cols, rows, err := db.Query("SELECT NULL AS val", map[string]interface{}{})
	if err != nil {
		t.Fatal(err)
	}
	if len(cols) != 1 || len(rows) != 1 {
		t.Fatal("expected 1 col, 1 row")
	}
	if rows[0][0] != "" {
		t.Errorf("expected empty string for NULL, got %q", rows[0][0])
	}
}

func TestLoadAllDataInserted(t *testing.T) {
	ds := testDataset()
	db, err := Load(ds)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	checks := []struct {
		table string
		want  string
	}{
		{"drivers", "2"},
		{"constructors", "2"},
		{"circuits", "1"},
		{"races", "3"},
		{"results", "5"},
		{"status", "1"},
		{"seasons", "2"},
	}

	for _, c := range checks {
		t.Run(c.table, func(t *testing.T) {
			_, rows, err := db.Query("SELECT COUNT(*) FROM "+c.table, map[string]interface{}{})
			if err != nil {
				t.Fatal(err)
			}
			if rows[0][0] != c.want {
				t.Errorf("table %s: got %s rows, want %s", c.table, rows[0][0], c.want)
			}
		})
	}
}
