package store

const schemaSQL = `
CREATE TABLE circuits (
	id       INTEGER PRIMARY KEY,
	ref      TEXT,
	name     TEXT,
	location TEXT,
	country  TEXT,
	lat      REAL,
	lng      REAL,
	alt      INTEGER,
	url      TEXT
);

CREATE TABLE constructors (
	id          INTEGER PRIMARY KEY,
	ref         TEXT,
	name        TEXT,
	nationality TEXT,
	url         TEXT
);

CREATE TABLE drivers (
	id          INTEGER PRIMARY KEY,
	ref         TEXT,
	number      INTEGER,
	code        TEXT,
	forename    TEXT,
	surname     TEXT,
	dob         TEXT,
	nationality TEXT,
	url         TEXT
);

CREATE TABLE races (
	id         INTEGER PRIMARY KEY,
	year       INTEGER,
	round      INTEGER,
	circuit_id INTEGER,
	name       TEXT,
	date       TEXT,
	time       TEXT,
	url        TEXT
);

CREATE TABLE results (
	id               INTEGER PRIMARY KEY,
	race_id          INTEGER,
	driver_id        INTEGER,
	constructor_id   INTEGER,
	number           INTEGER,
	grid             INTEGER,
	position         INTEGER,
	position_text    TEXT,
	position_order   INTEGER,
	points           REAL,
	laps             INTEGER,
	time             TEXT,
	milliseconds     INTEGER,
	fastest_lap      INTEGER,
	rank             INTEGER,
	fastest_lap_time TEXT,
	fastest_lap_speed TEXT,
	status_id        INTEGER
);

CREATE TABLE lap_times (
	race_id      INTEGER,
	driver_id    INTEGER,
	lap          INTEGER,
	position     INTEGER,
	time         TEXT,
	milliseconds INTEGER
);

CREATE TABLE pit_stops (
	race_id      INTEGER,
	driver_id    INTEGER,
	stop         INTEGER,
	lap          INTEGER,
	time         TEXT,
	duration     TEXT,
	milliseconds INTEGER
);

CREATE TABLE qualifying (
	id             INTEGER PRIMARY KEY,
	race_id        INTEGER,
	driver_id      INTEGER,
	constructor_id INTEGER,
	number         INTEGER,
	position       INTEGER,
	q1             TEXT,
	q2             TEXT,
	q3             TEXT
);

CREATE TABLE driver_standings (
	id            INTEGER PRIMARY KEY,
	race_id       INTEGER,
	driver_id     INTEGER,
	points        REAL,
	position      INTEGER,
	position_text TEXT,
	wins          INTEGER
);

CREATE TABLE constructor_standings (
	id              INTEGER PRIMARY KEY,
	race_id         INTEGER,
	constructor_id  INTEGER,
	points          REAL,
	position        INTEGER,
	position_text   TEXT,
	wins            INTEGER
);

CREATE TABLE constructor_results (
	id              INTEGER PRIMARY KEY,
	race_id         INTEGER,
	constructor_id  INTEGER,
	points          REAL,
	status          TEXT
);

CREATE TABLE status (
	id     INTEGER PRIMARY KEY,
	status TEXT
);

CREATE TABLE seasons (
	year INTEGER PRIMARY KEY,
	url  TEXT
);

CREATE TABLE sprint_results (
	id               INTEGER PRIMARY KEY,
	race_id          INTEGER,
	driver_id        INTEGER,
	constructor_id   INTEGER,
	number           INTEGER,
	grid             INTEGER,
	position         INTEGER,
	position_text    TEXT,
	position_order   INTEGER,
	points           REAL,
	laps             INTEGER,
	time             TEXT,
	milliseconds     INTEGER,
	fastest_lap      INTEGER,
	fastest_lap_time TEXT,
	status_id        INTEGER
);

CREATE INDEX idx_races_year ON races(year);
CREATE INDEX idx_results_race ON results(race_id);
CREATE INDEX idx_results_driver ON results(driver_id);
CREATE INDEX idx_results_constructor ON results(constructor_id);
CREATE INDEX idx_results_position ON results(position);
CREATE INDEX idx_lap_times_race ON lap_times(race_id);
CREATE INDEX idx_lap_times_driver ON lap_times(driver_id);
CREATE INDEX idx_pit_stops_race ON pit_stops(race_id);
CREATE INDEX idx_qualifying_race ON qualifying(race_id);
CREATE INDEX idx_driver_standings_race ON driver_standings(race_id);
CREATE INDEX idx_constructor_standings_race ON constructor_standings(race_id);
`
