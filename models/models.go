package models

import "time"

// Circuit represents an F1 circuit/track.
type Circuit struct {
	ID       int
	Ref      string
	Name     string
	Location string
	Country  string
	Lat      float64
	Lng      float64
	Alt      int
	URL      string
}

// Constructor represents an F1 constructor/team.
type Constructor struct {
	ID          int
	Ref         string
	Name        string
	Nationality string
	URL         string
}

// Driver represents an F1 driver.
type Driver struct {
	ID          int
	Ref         string
	Number      int
	Code        string
	Forename    string
	Surname     string
	DOB         time.Time
	Nationality string
	URL         string
}

// Race represents an F1 race event.
type Race struct {
	ID        int
	Year      int
	Round     int
	CircuitID int
	Name      string
	Date      time.Time
	Time      string
	URL       string
}

// Result represents a race result for a driver.
type Result struct {
	ID            int
	RaceID        int
	DriverID      int
	ConstructorID int
	Number        int
	Grid          int
	Position      int
	PositionText  string
	PositionOrder int
	Points        float64
	Laps          int
	Time          string
	Milliseconds  int
	FastestLap    int
	Rank          int
	FastestLapTime string
	FastestLapSpeed string
	StatusID      int
}

// LapTime represents a single lap time entry.
type LapTime struct {
	RaceID       int
	DriverID     int
	Lap          int
	Position     int
	Time         string
	Milliseconds int
}

// PitStop represents a pit stop during a race.
type PitStop struct {
	RaceID       int
	DriverID     int
	Stop         int
	Lap          int
	Time         string
	Duration     string
	Milliseconds int
}

// Qualifying represents a qualifying session result.
type Qualifying struct {
	ID            int
	RaceID        int
	DriverID      int
	ConstructorID int
	Number        int
	Position      int
	Q1            string
	Q2            string
	Q3            string
}

// DriverStanding represents a driver's championship standing after a race.
type DriverStanding struct {
	ID            int
	RaceID        int
	DriverID      int
	Points        float64
	Position      int
	PositionText  string
	Wins          int
}

// ConstructorStanding represents a constructor's championship standing after a race.
type ConstructorStanding struct {
	ID            int
	RaceID        int
	ConstructorID int
	Points        float64
	Position      int
	PositionText  string
	Wins          int
}

// ConstructorResult represents constructor results per race.
type ConstructorResult struct {
	ID            int
	RaceID        int
	ConstructorID int
	Points        float64
	Status        string
}

// Status represents a race finish status (e.g., "Finished", "Retired", "Accident").
type Status struct {
	ID     int
	Status string
}

// Season represents an F1 season.
type Season struct {
	Year int
	URL  string
}

// SprintResult represents a sprint race result.
type SprintResult struct {
	ID            int
	RaceID        int
	DriverID      int
	ConstructorID int
	Number        int
	Grid          int
	Position      int
	PositionText  string
	PositionOrder int
	Points        float64
	Laps          int
	Time          string
	Milliseconds  int
	FastestLap    int
	FastestLapTime string
	StatusID      int
}
