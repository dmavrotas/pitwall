package store

import (
	"database/sql"
	"fmt"
	"strings"

	_ "modernc.org/sqlite" // register sqlite driver

	"github.com/dmavrotas/pitwall/loader"
)

// DB wraps an in-memory SQLite database with the F1 dataset.
type DB struct {
	conn *sql.DB
}

// Load creates an in-memory SQLite database and populates it from a Dataset.
func Load(ds *loader.Dataset) (*DB, error) {
	conn, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		return nil, fmt.Errorf("opening sqlite: %w", err)
	}

	// Create schema
	for _, stmt := range strings.Split(schemaSQL, ";") {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if _, err := conn.Exec(stmt); err != nil {
			return nil, fmt.Errorf("creating schema: %w", err)
		}
	}

	db := &DB{conn: conn}
	if err := db.insertAll(ds); err != nil {
		_ = conn.Close()
		return nil, err
	}

	return db, nil
}

// Query executes a SQL query and returns column names and rows.
func (db *DB) Query(query string, args map[string]interface{}) (columns []string, results [][]string, err error) {
	// Convert named params to positional by scanning left-to-right
	var positional []interface{}
	q := query
	for {
		idx := strings.Index(q, ":")
		if idx < 0 {
			break
		}
		// Extract param name (letters, digits, underscores after the colon)
		end := idx + 1
		for end < len(q) && (q[end] == '_' || (q[end] >= 'a' && q[end] <= 'z') || (q[end] >= 'A' && q[end] <= 'Z') || (q[end] >= '0' && q[end] <= '9')) {
			end++
		}
		if end == idx+1 {
			// Not a named param, skip this colon
			q = q[:idx] + "COLON_PLACEHOLDER" + q[idx+1:]
			continue
		}
		name := q[idx+1 : end]
		if v, ok := args[name]; ok {
			positional = append(positional, v)
			q = q[:idx] + "?" + q[end:]
		} else {
			// Unknown param, skip
			q = q[:idx] + "COLON_PLACEHOLDER" + q[idx+1:]
		}
	}
	q = strings.ReplaceAll(q, "COLON_PLACEHOLDER", ":")

	rows, err := db.conn.Query(q, positional...)
	if err != nil {
		return nil, nil, fmt.Errorf("query error: %w", err)
	}
	defer func() { _ = rows.Close() }()

	cols, err := rows.Columns()
	if err != nil {
		return nil, nil, err
	}

	for rows.Next() {
		values := make([]interface{}, len(cols))
		ptrs := make([]interface{}, len(cols))
		for i := range values {
			ptrs[i] = &values[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, nil, err
		}
		row := make([]string, len(cols))
		for i, v := range values {
			if v == nil {
				row[i] = ""
			} else {
				row[i] = fmt.Sprintf("%v", v)
			}
		}
		results = append(results, row)
	}
	return cols, results, rows.Err()
}

// Close closes the database connection.
func (db *DB) Close() error {
	return db.conn.Close()
}

//nolint:errcheck // batch inserts in a transaction; committed/rolled back atomically
func (db *DB) insertAll(ds *loader.Dataset) error {
	tx, err := db.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Circuits
	stmt, _ := tx.Prepare("INSERT INTO circuits VALUES (?,?,?,?,?,?,?,?,?)")
	for _, c := range ds.Circuits {
		stmt.Exec(c.ID, c.Ref, c.Name, c.Location, c.Country, c.Lat, c.Lng, c.Alt, c.URL)
	}
	stmt.Close()

	// Constructors
	stmt, _ = tx.Prepare("INSERT INTO constructors VALUES (?,?,?,?,?)")
	for _, c := range ds.Constructors {
		stmt.Exec(c.ID, c.Ref, c.Name, c.Nationality, c.URL)
	}
	stmt.Close()

	// Drivers
	stmt, _ = tx.Prepare("INSERT INTO drivers VALUES (?,?,?,?,?,?,?,?,?)")
	for i := range ds.Drivers {
		d := &ds.Drivers[i]
		dob := ""
		if !d.DOB.IsZero() {
			dob = d.DOB.Format("2006-01-02")
		}
		stmt.Exec(d.ID, d.Ref, d.Number, d.Code, d.Forename, d.Surname, dob, d.Nationality, d.URL)
	}
	stmt.Close()

	// Races
	stmt, _ = tx.Prepare("INSERT INTO races VALUES (?,?,?,?,?,?,?,?)")
	for _, r := range ds.Races {
		date := ""
		if !r.Date.IsZero() {
			date = r.Date.Format("2006-01-02")
		}
		stmt.Exec(r.ID, r.Year, r.Round, r.CircuitID, r.Name, date, r.Time, r.URL)
	}
	stmt.Close()

	// Results
	stmt, _ = tx.Prepare("INSERT INTO results VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)")
	for i := range ds.Results {
		r := &ds.Results[i]
		stmt.Exec(r.ID, r.RaceID, r.DriverID, r.ConstructorID, r.Number, r.Grid,
			r.Position, r.PositionText, r.PositionOrder, r.Points, r.Laps,
			r.Time, r.Milliseconds, r.FastestLap, r.Rank, r.FastestLapTime,
			r.FastestLapSpeed, r.StatusID)
	}
	stmt.Close()

	// Lap times
	stmt, _ = tx.Prepare("INSERT INTO lap_times VALUES (?,?,?,?,?,?)")
	for _, l := range ds.LapTimes {
		stmt.Exec(l.RaceID, l.DriverID, l.Lap, l.Position, l.Time, l.Milliseconds)
	}
	stmt.Close()

	// Pit stops
	stmt, _ = tx.Prepare("INSERT INTO pit_stops VALUES (?,?,?,?,?,?,?)")
	for _, p := range ds.PitStops {
		stmt.Exec(p.RaceID, p.DriverID, p.Stop, p.Lap, p.Time, p.Duration, p.Milliseconds)
	}
	stmt.Close()

	// Qualifying
	stmt, _ = tx.Prepare("INSERT INTO qualifying VALUES (?,?,?,?,?,?,?,?,?)")
	for _, q := range ds.Qualifying {
		stmt.Exec(q.ID, q.RaceID, q.DriverID, q.ConstructorID, q.Number, q.Position, q.Q1, q.Q2, q.Q3)
	}
	stmt.Close()

	// Driver standings
	stmt, _ = tx.Prepare("INSERT INTO driver_standings VALUES (?,?,?,?,?,?,?)")
	for _, s := range ds.DriverStandings {
		stmt.Exec(s.ID, s.RaceID, s.DriverID, s.Points, s.Position, s.PositionText, s.Wins)
	}
	stmt.Close()

	// Constructor standings
	stmt, _ = tx.Prepare("INSERT INTO constructor_standings VALUES (?,?,?,?,?,?,?)")
	for _, s := range ds.ConstructorStandings {
		stmt.Exec(s.ID, s.RaceID, s.ConstructorID, s.Points, s.Position, s.PositionText, s.Wins)
	}
	stmt.Close()

	// Constructor results
	stmt, _ = tx.Prepare("INSERT INTO constructor_results VALUES (?,?,?,?,?)")
	for _, r := range ds.ConstructorResults {
		stmt.Exec(r.ID, r.RaceID, r.ConstructorID, r.Points, r.Status)
	}
	stmt.Close()

	// Status
	stmt, _ = tx.Prepare("INSERT INTO status VALUES (?,?)")
	for _, s := range ds.Statuses {
		stmt.Exec(s.ID, s.Status)
	}
	stmt.Close()

	// Seasons
	stmt, _ = tx.Prepare("INSERT INTO seasons VALUES (?,?)")
	for _, s := range ds.Seasons {
		stmt.Exec(s.Year, s.URL)
	}
	stmt.Close()

	// Sprint results
	stmt, _ = tx.Prepare("INSERT INTO sprint_results VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)")
	for i := range ds.SprintResults {
		r := &ds.SprintResults[i]
		stmt.Exec(r.ID, r.RaceID, r.DriverID, r.ConstructorID, r.Number, r.Grid,
			r.Position, r.PositionText, r.PositionOrder, r.Points, r.Laps,
			r.Time, r.Milliseconds, r.FastestLap, r.FastestLapTime, r.StatusID)
	}
	stmt.Close()

	return tx.Commit()
}
