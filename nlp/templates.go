package nlp

import (
	"fmt"
	"strings"
)

// QueryTemplate defines a SQL template for an intent.
type QueryTemplate struct {
	SQL      string
	Describe string
}

// yearFilter returns SQL clauses for the year fields in e, qualified against
// the races table alias `ra`. It also populates the relevant args. Returns ""
// when no year filter applies.
func yearFilter(e *Entities, args map[string]interface{}) string {
	if e.Year > 0 {
		args["year"] = e.Year
		return "ra.year = :year"
	}
	var parts []string
	if e.YearFrom > 0 {
		args["year_from"] = e.YearFrom
		parts = append(parts, "ra.year >= :year_from")
	}
	if e.YearTo > 0 {
		args["year_to"] = e.YearTo
		parts = append(parts, "ra.year <= :year_to")
	}
	return strings.Join(parts, " AND ")
}

// yearDesc returns a human suffix like " in 2020" or " from 2018 to 2022".
func yearDesc(e *Entities) string {
	if e.Year > 0 {
		return fmt.Sprintf(" in %d", e.Year)
	}
	switch {
	case e.YearFrom > 0 && e.YearTo > 0:
		return fmt.Sprintf(" from %d to %d", e.YearFrom, e.YearTo)
	case e.YearFrom > 0:
		return fmt.Sprintf(" since %d", e.YearFrom)
	case e.YearTo > 0:
		return fmt.Sprintf(" until %d", e.YearTo)
	}
	return ""
}

// hasYear reports whether e specifies any year filter (single or range).
func hasYear(e *Entities) bool {
	return e.Year > 0 || e.YearFrom > 0 || e.YearTo > 0
}

// BuildQuery generates the SQL query and description for a given intent and entities.
func BuildQuery(intent Intent, e *Entities) (sql, desc string, args map[string]interface{}) {
	args = make(map[string]interface{})

	switch intent {
	case IntentWins:
		sql, desc = buildWinsQuery(e, args)
	case IntentPoints:
		sql, desc = buildPointsQuery(e, args)
	case IntentChampionship:
		sql, desc = buildChampionshipQuery(e, args)
	case IntentPoles:
		sql, desc = buildPolesQuery(e, args)
	case IntentDNF:
		sql, desc = buildDNFQuery(e, args)
	case IntentPitStops:
		sql, desc = buildPitStopsQuery(e, args)
	case IntentFastestLap:
		sql, desc = buildFastestLapQuery(e, args)
	case IntentRaceResult:
		sql, desc = buildRaceResultQuery(e, args)
	case IntentDriverInfo:
		sql, desc = buildDriverInfoQuery(e, args)
	case IntentConstructorInfo:
		sql, desc = buildConstructorInfoQuery(e, args)
	case IntentCircuitInfo:
		sql, desc = buildCircuitInfoQuery(e, args)
	case IntentTeammates:
		sql, desc = buildTeammatesQuery(e, args)
	case IntentHeadToHead:
		sql, desc = buildHeadToHeadQuery(e, args)
	case IntentSeasonOverview:
		sql, desc = buildSeasonOverviewQuery(e, args)
	default:
		return "", "", nil
	}

	return sql, desc, args
}

func buildWinsQuery(e *Entities, args map[string]interface{}) (sql, desc string) {
	var filters []string
	desc = "Top drivers by race wins"

	if yf := yearFilter(e, args); yf != "" {
		filters = append(filters, yf)
		desc = "Top drivers by race wins" + yearDesc(e)
	}
	if e.DriverID > 0 {
		filters = append(filters, "r.driver_id = :driver_id")
		args["driver_id"] = e.DriverID
		desc = "Race wins for driver" + yearDesc(e)
	}
	if e.ConstructorID > 0 {
		filters = append(filters, "r.constructor_id = :constructor_id")
		args["constructor_id"] = e.ConstructorID
		desc = "Top drivers by wins with constructor" + yearDesc(e)
	}
	if e.CircuitID > 0 {
		filters = append(filters, "ra.circuit_id = :circuit_id")
		args["circuit_id"] = e.CircuitID
		desc = "Top drivers by wins at circuit" + yearDesc(e)
	}

	// Ordinal selector: "first win" / "last win" for a specific driver returns
	// a single chronological race row instead of a win-count aggregate.
	if (e.Ordinal == "first" || e.Ordinal == "last") && e.DriverID > 0 {
		order := "ASC"
		descPrefix := "First win for driver"
		if e.Ordinal == "last" {
			order = "DESC"
			descPrefix = "Most recent win for driver"
		}
		where := "r.position = 1 AND " + strings.Join(filters, " AND ")
		sql = fmt.Sprintf(`
SELECT ra.year,
       ra.round,
       ra.name AS race,
       ra.date,
       c.name AS team,
       d.forename || ' ' || d.surname AS driver
FROM results r
JOIN drivers d ON d.id = r.driver_id
JOIN constructors c ON c.id = r.constructor_id
JOIN races ra ON ra.id = r.race_id
WHERE %s
ORDER BY ra.date %s
LIMIT 1`, where, order)
		return sql, descPrefix + yearDesc(e)
	}

	where := "r.position = 1"
	if len(filters) > 0 {
		where += " AND " + strings.Join(filters, " AND ")
	}

	sql = fmt.Sprintf(`
SELECT d.forename || ' ' || d.surname AS driver,
       c.name AS team,
       COUNT(*) AS wins
FROM results r
JOIN drivers d ON d.id = r.driver_id
JOIN constructors c ON c.id = r.constructor_id
JOIN races ra ON ra.id = r.race_id
WHERE %s
GROUP BY r.driver_id
ORDER BY wins DESC
LIMIT 15`, where)

	return sql, desc
}

func buildPointsQuery(e *Entities, args map[string]interface{}) (sql, desc string) {
	var filters []string
	desc = "Top drivers by total points"

	if yf := yearFilter(e, args); yf != "" {
		filters = append(filters, yf)
		desc = "Top drivers by points" + yearDesc(e)
	}
	if e.DriverID > 0 {
		filters = append(filters, "ap.driver_id = :driver_id")
		args["driver_id"] = e.DriverID
		desc = "Points scored by driver" + yearDesc(e)
	}
	if e.ConstructorID > 0 {
		filters = append(filters, "ap.constructor_id = :constructor_id")
		args["constructor_id"] = e.ConstructorID
	}

	where := "1=1"
	if len(filters) > 0 {
		where = strings.Join(filters, " AND ")
	}

	// Union main results with sprint results so championship-point totals
	// include sprint contributions for the 2021+ era.
	pointsExpr := "ROUND(SUM(ap.points), 1) AS total_points"
	orderCol := "total_points"
	if e.Average || e.PerRace {
		pointsExpr = "ROUND(SUM(ap.points) * 1.0 / COUNT(DISTINCT ap.race_id), 2) AS points_per_race"
		orderCol = "points_per_race"
		desc = "Points per race" + yearDesc(e)
		if e.DriverID > 0 {
			desc = "Average points per race for driver" + yearDesc(e)
		}
	}

	sql = fmt.Sprintf(`
WITH all_points AS (
    SELECT driver_id, constructor_id, race_id, points FROM results
    UNION ALL
    SELECT driver_id, constructor_id, race_id, points FROM sprint_results
)
SELECT d.forename || ' ' || d.surname AS driver,
       c.name AS team,
       %s,
       COUNT(DISTINCT ap.race_id) AS races
FROM all_points ap
JOIN drivers d ON d.id = ap.driver_id
JOIN constructors c ON c.id = ap.constructor_id
JOIN races ra ON ra.id = ap.race_id
WHERE %s
GROUP BY ap.driver_id
ORDER BY %s DESC
LIMIT 15`, pointsExpr, where, orderCol)

	return sql, desc
}

func buildChampionshipQuery(e *Entities, args map[string]interface{}) (sql, desc string) {
	// Specific driver's championship history
	if e.DriverID > 0 {
		args["driver_id"] = e.DriverID
		yf := yearFilter(e, args)
		if yf != "" {
			yf = "AND " + yf
		}
		sql = fmt.Sprintf(`
SELECT ra.year AS season,
       ds.position,
       ds.points,
       ds.wins,
       CASE WHEN ds.position = 1 THEN 'CHAMPION' ELSE '' END AS title
FROM driver_standings ds
JOIN drivers d ON d.id = ds.driver_id
JOIN races ra ON ra.id = ds.race_id
WHERE ds.driver_id = :driver_id
  AND ra.round = (SELECT MAX(r2.round) FROM races r2 WHERE r2.year = ra.year)
  %s
ORDER BY ra.year`, yf)
		return sql, "Driver championship history"
	}

	// Constructor championships
	if e.ConstructorID > 0 {
		args["constructor_id"] = e.ConstructorID
		sql = `
SELECT ra.year AS season,
       cs.position,
       cs.points,
       cs.wins,
       CASE WHEN cs.position = 1 THEN 'CHAMPION' ELSE '' END AS title
FROM constructor_standings cs
JOIN constructors c ON c.id = cs.constructor_id
JOIN races ra ON ra.id = cs.race_id
WHERE cs.constructor_id = :constructor_id
  AND ra.round = (SELECT MAX(r2.round) FROM races r2 WHERE r2.year = ra.year)
ORDER BY ra.year`
		return sql, "Constructor championship history"
	}

	// Year-only championship query. Default to the champion row; if the user
	// explicitly asked for "standings"/"rankings", return the top 15.
	if e.Year > 0 {
		args["year"] = e.Year
		limit := "LIMIT 1"
		desc := fmt.Sprintf("%d World Champion", e.Year)
		if e.ShowFullStandings {
			limit = "LIMIT 15"
			desc = fmt.Sprintf("Championship standings %d", e.Year)
		}
		sql = fmt.Sprintf(`
SELECT d.forename || ' ' || d.surname AS driver,
       ds.points,
       ds.wins,
       ds.position
FROM driver_standings ds
JOIN drivers d ON d.id = ds.driver_id
JOIN races ra ON ra.id = ds.race_id
WHERE ra.id = (
    SELECT id FROM races WHERE year = :year ORDER BY round DESC LIMIT 1
)
ORDER BY ds.position
%s`, limit)
		return sql, desc
	}

	// All-time championship winners
	sql = `
SELECT d.forename || ' ' || d.surname AS driver,
       COUNT(*) AS titles
FROM driver_standings ds
JOIN drivers d ON d.id = ds.driver_id
JOIN races ra ON ra.id = ds.race_id
WHERE ds.position = 1
  AND ra.round = (SELECT MAX(r2.round) FROM races r2 WHERE r2.year = ra.year)
GROUP BY ds.driver_id
ORDER BY titles DESC
LIMIT 15`
	return sql, "Drivers by World Championship titles"
}

func buildPolesQuery(e *Entities, args map[string]interface{}) (sql, desc string) {
	var filters []string
	desc = "Top drivers by pole positions"

	if yf := yearFilter(e, args); yf != "" {
		filters = append(filters, yf)
		desc = "Pole positions" + yearDesc(e)
	}
	if e.DriverID > 0 {
		filters = append(filters, "q.driver_id = :driver_id")
		args["driver_id"] = e.DriverID
	}

	where := "q.position = 1"
	if len(filters) > 0 {
		where += " AND " + strings.Join(filters, " AND ")
	}

	sql = fmt.Sprintf(`
SELECT d.forename || ' ' || d.surname AS driver,
       COUNT(*) AS poles
FROM qualifying q
JOIN drivers d ON d.id = q.driver_id
JOIN races ra ON ra.id = q.race_id
WHERE %s
GROUP BY q.driver_id
ORDER BY poles DESC
LIMIT 15`, where)

	return sql, desc
}

func buildDNFQuery(e *Entities, args map[string]interface{}) (sql, desc string) {
	var filters []string
	desc = "Top DNF reasons"

	if yf := yearFilter(e, args); yf != "" {
		filters = append(filters, yf)
		desc = "DNF reasons" + yearDesc(e)
	}
	if e.DriverID > 0 {
		filters = append(filters, "r.driver_id = :driver_id")
		args["driver_id"] = e.DriverID
		desc = "DNF history for driver" + yearDesc(e)
	}

	// A DNF = not classified at race end. The semantic signal in Ergast is the
	// status: anything other than "Finished" or a lap-down classification
	// ("+1 Lap", "+2 Laps", ...) means the driver didn't reach the flag.
	where := "s.status != 'Finished' AND s.status NOT LIKE '+%Lap%'"
	if len(filters) > 0 {
		where += " AND " + strings.Join(filters, " AND ")
	}

	sql = fmt.Sprintf(`
SELECT s.status AS reason,
       COUNT(*) AS count
FROM results r
JOIN status s ON s.id = r.status_id
JOIN races ra ON ra.id = r.race_id
WHERE %s
GROUP BY s.status
ORDER BY count DESC
LIMIT 15`, where)

	return sql, desc
}

func buildPitStopsQuery(e *Entities, args map[string]interface{}) (sql, desc string) {
	desc = "Average pit stop duration by year"

	if yf := yearFilter(e, args); yf != "" {
		desc = "Fastest pit stops" + yearDesc(e)
		sql = fmt.Sprintf(`
SELECT d.forename || ' ' || d.surname AS driver,
       ra.name AS race,
       p.lap,
       p.duration,
       p.milliseconds
FROM pit_stops p
JOIN drivers d ON d.id = p.driver_id
JOIN races ra ON ra.id = p.race_id
WHERE %s AND p.milliseconds > 0
ORDER BY p.milliseconds ASC
LIMIT 15`, yf)
		return sql, desc
	}

	sql = `
SELECT ra.year,
       ROUND(AVG(p.milliseconds) / 1000.0, 2) AS avg_seconds,
       MIN(p.milliseconds) AS fastest_ms,
       COUNT(*) AS total_stops
FROM pit_stops p
JOIN races ra ON ra.id = p.race_id
WHERE p.milliseconds > 0
GROUP BY ra.year
ORDER BY ra.year`

	return sql, desc
}

func buildFastestLapQuery(e *Entities, args map[string]interface{}) (sql, desc string) {
	var filters []string
	desc = "Fastest lap times"

	if yf := yearFilter(e, args); yf != "" {
		filters = append(filters, yf)
		desc = "Fastest laps" + yearDesc(e)
	}
	if e.DriverID > 0 {
		filters = append(filters, "r.driver_id = :driver_id")
		args["driver_id"] = e.DriverID
	}
	if e.CircuitID > 0 {
		filters = append(filters, "ra.circuit_id = :circuit_id")
		args["circuit_id"] = e.CircuitID
	}

	where := "r.fastest_lap_time != '' AND r.rank = 1"
	if len(filters) > 0 {
		where += " AND " + strings.Join(filters, " AND ")
	}

	sql = fmt.Sprintf(`
SELECT d.forename || ' ' || d.surname AS driver,
       ra.name AS race,
       ra.year,
       r.fastest_lap_time AS lap_time,
       r.fastest_lap_speed AS speed_kph
FROM results r
JOIN drivers d ON d.id = r.driver_id
JOIN races ra ON ra.id = r.race_id
WHERE %s
ORDER BY r.fastest_lap_time ASC
LIMIT 15`, where)

	return sql, desc
}

func buildRaceResultQuery(e *Entities, args map[string]interface{}) (sql, desc string) {
	var filters []string
	desc = "Race results"

	if yf := yearFilter(e, args); yf != "" {
		filters = append(filters, yf)
		desc = "Race results" + yearDesc(e)
	}
	if e.DriverID > 0 {
		filters = append(filters, "r.driver_id = :driver_id")
		args["driver_id"] = e.DriverID
	}
	if e.CircuitID > 0 {
		filters = append(filters, "ra.circuit_id = :circuit_id")
		args["circuit_id"] = e.CircuitID
	}
	if e.ConstructorID > 0 {
		filters = append(filters, "r.constructor_id = :constructor_id")
		args["constructor_id"] = e.ConstructorID
	}

	where := "1=1"
	if len(filters) > 0 {
		where = strings.Join(filters, " AND ")
	}

	// "Worst finish" for a driver: return their lowest classified position
	// (DNFs ordered last via position_order). Single-row result.
	if e.Ordinal == "worst" && e.DriverID > 0 {
		sql = fmt.Sprintf(`
SELECT ra.year,
       ra.name AS race,
       r.position_order AS finish_pos,
       COALESCE(NULLIF(r.position_text, ''), 'R') AS classified,
       c.name AS team,
       d.forename || ' ' || d.surname AS driver,
       s.status
FROM results r
JOIN drivers d ON d.id = r.driver_id
JOIN constructors c ON c.id = r.constructor_id
JOIN races ra ON ra.id = r.race_id
JOIN status s ON s.id = r.status_id
WHERE %s
ORDER BY r.position_order DESC, ra.date DESC
LIMIT 1`, where)
		return sql, "Worst finish for driver" + yearDesc(e)
	}

	// If we have a specific race (circuit+year), show full result
	if e.CircuitID > 0 || (e.DriverID > 0 && hasYear(e)) {
		sql = fmt.Sprintf(`
SELECT r.position_order AS pos,
       d.forename || ' ' || d.surname AS driver,
       c.name AS team,
       r.points,
       r.laps,
       COALESCE(NULLIF(r.time,''), s.status) AS time_status,
       ra.name AS race
FROM results r
JOIN drivers d ON d.id = r.driver_id
JOIN constructors c ON c.id = r.constructor_id
JOIN races ra ON ra.id = r.race_id
JOIN status s ON s.id = r.status_id
WHERE %s
ORDER BY ra.date, r.position_order
LIMIT 30`, where)
		return sql, desc
	}

	// Podium finishes
	sql = fmt.Sprintf(`
SELECT d.forename || ' ' || d.surname AS driver,
       COUNT(*) AS podiums,
       SUM(CASE WHEN r.position = 1 THEN 1 ELSE 0 END) AS wins,
       SUM(CASE WHEN r.position = 2 THEN 1 ELSE 0 END) AS p2,
       SUM(CASE WHEN r.position = 3 THEN 1 ELSE 0 END) AS p3
FROM results r
JOIN drivers d ON d.id = r.driver_id
JOIN races ra ON ra.id = r.race_id
WHERE r.position BETWEEN 1 AND 3 AND %s
GROUP BY r.driver_id
ORDER BY podiums DESC
LIMIT 15`, where)

	return sql, desc
}

func buildDriverInfoQuery(e *Entities, args map[string]interface{}) (sql, desc string) {
	if e.DriverID > 0 {
		args["driver_id"] = e.DriverID
		sql = `
SELECT d.forename || ' ' || d.surname AS name,
       d.code,
       d.dob AS date_of_birth,
       d.nationality,
       COUNT(DISTINCT r.race_id) AS races,
       SUM(CASE WHEN r.position = 1 THEN 1 ELSE 0 END) AS wins,
       SUM(r.points) AS total_points
FROM drivers d
LEFT JOIN results r ON r.driver_id = d.id
WHERE d.id = :driver_id
GROUP BY d.id`
		return sql, "Driver profile"
	}

	sql = `
SELECT d.forename || ' ' || d.surname AS name,
       d.nationality,
       COUNT(DISTINCT r.race_id) AS races,
       SUM(CASE WHEN r.position = 1 THEN 1 ELSE 0 END) AS wins
FROM drivers d
LEFT JOIN results r ON r.driver_id = d.id
GROUP BY d.id
ORDER BY wins DESC
LIMIT 15`
	return sql, "Top drivers overview"
}

func buildConstructorInfoQuery(e *Entities, args map[string]interface{}) (sql, desc string) {
	if e.ConstructorID > 0 {
		args["constructor_id"] = e.ConstructorID
		sql = `
SELECT c.name,
       c.nationality,
       COUNT(DISTINCT r.race_id) AS races,
       SUM(CASE WHEN r.position = 1 THEN 1 ELSE 0 END) AS wins,
       SUM(r.points) AS total_points
FROM constructors c
LEFT JOIN results r ON r.constructor_id = c.id
WHERE c.id = :constructor_id
GROUP BY c.id`
		return sql, "Constructor profile"
	}

	sql = `
SELECT c.name,
       c.nationality,
       COUNT(DISTINCT r.race_id) AS races,
       SUM(CASE WHEN r.position = 1 THEN 1 ELSE 0 END) AS wins,
       SUM(r.points) AS total_points
FROM constructors c
LEFT JOIN results r ON r.constructor_id = c.id
GROUP BY c.id
ORDER BY total_points DESC
LIMIT 15`
	return sql, "Top constructors overview"
}

func buildCircuitInfoQuery(e *Entities, args map[string]interface{}) (sql, desc string) {
	if e.CircuitID > 0 {
		args["circuit_id"] = e.CircuitID
		sql = `
SELECT ci.name,
       ci.location,
       ci.country,
       COUNT(DISTINCT ra.id) AS races_held,
       MIN(ra.year) AS first_race,
       MAX(ra.year) AS last_race
FROM circuits ci
LEFT JOIN races ra ON ra.circuit_id = ci.id
WHERE ci.id = :circuit_id
GROUP BY ci.id`
		return sql, "Circuit profile"
	}

	sql = `
SELECT ci.name,
       ci.country,
       COUNT(DISTINCT ra.id) AS races_held,
       MIN(ra.year) AS first_year,
       MAX(ra.year) AS last_year
FROM circuits ci
LEFT JOIN races ra ON ra.circuit_id = ci.id
GROUP BY ci.id
ORDER BY races_held DESC
LIMIT 15`
	return sql, "Most used circuits"
}

func buildTeammatesQuery(e *Entities, args map[string]interface{}) (sql, desc string) {
	if e.DriverID > 0 {
		args["driver_id"] = e.DriverID
		yf := yearFilter(e, args)
		if yf != "" {
			yf = "AND " + yf
		}
		sql = fmt.Sprintf(`
SELECT d2.forename || ' ' || d2.surname AS teammate,
       c.name AS team,
       ra.year,
       COUNT(*) AS races_together
FROM results r1
JOIN results r2 ON r1.race_id = r2.race_id
    AND r1.constructor_id = r2.constructor_id
    AND r1.driver_id != r2.driver_id
JOIN drivers d2 ON d2.id = r2.driver_id
JOIN constructors c ON c.id = r1.constructor_id
JOIN races ra ON ra.id = r1.race_id
WHERE r1.driver_id = :driver_id %s
GROUP BY r2.driver_id, r1.constructor_id, ra.year
ORDER BY ra.year DESC, races_together DESC
LIMIT 20`, yf)
		return sql, "Teammates history"
	}

	return "", ""
}

func buildHeadToHeadQuery(e *Entities, args map[string]interface{}) (sql, desc string) {
	if e.DriverID > 0 && e.DriverID2 > 0 {
		args["driver1"] = e.DriverID
		args["driver2"] = e.DriverID2
		yf := yearFilter(e, args)
		if yf != "" {
			yf = "AND " + yf
		}
		sql = fmt.Sprintf(`
SELECT
    d.forename || ' ' || d.surname AS driver,
    COUNT(*) AS races,
    SUM(CASE WHEN r.position = 1 THEN 1 ELSE 0 END) AS wins,
    SUM(CASE WHEN r.position BETWEEN 1 AND 3 THEN 1 ELSE 0 END) AS podiums,
    SUM(r.points) AS points,
    ROUND(AVG(r.position_order), 1) AS avg_finish
FROM results r
JOIN drivers d ON d.id = r.driver_id
JOIN races ra ON ra.id = r.race_id
WHERE r.driver_id IN (:driver1, :driver2) %s
GROUP BY r.driver_id`, yf)
		return sql, "Head to head comparison"
	}

	if e.ConstructorID > 0 && e.ConstructorID2 > 0 {
		args["team1"] = e.ConstructorID
		args["team2"] = e.ConstructorID2
		yf := yearFilter(e, args)
		if yf != "" {
			yf = "AND " + yf
		}
		sql = fmt.Sprintf(`
SELECT
    c.name AS team,
    COUNT(DISTINCT r.race_id) AS races,
    SUM(CASE WHEN r.position = 1 THEN 1 ELSE 0 END) AS wins,
    SUM(r.points) AS points
FROM results r
JOIN constructors c ON c.id = r.constructor_id
JOIN races ra ON ra.id = r.race_id
WHERE r.constructor_id IN (:team1, :team2) %s
GROUP BY r.constructor_id`, yf)
		return sql, "Team comparison"
	}

	// Driver vs team: compare one driver's stats against one team's stats.
	// Only makes sense scoped to a shared period — require a year, otherwise
	// the comparison is dominated by sample-size differences.
	if e.DriverID > 0 && e.ConstructorID > 0 {
		args["driver_id"] = e.DriverID
		args["constructor_id"] = e.ConstructorID
		yf := yearFilter(e, args)
		driverYearFilter, teamYearFilter := "", ""
		if yf != "" {
			driverYearFilter = "AND " + yf
			teamYearFilter = "AND " + yf
		}
		sql = fmt.Sprintf(`
SELECT
    d.forename || ' ' || d.surname AS subject,
    'Driver' AS kind,
    COUNT(DISTINCT r.race_id) AS races,
    SUM(CASE WHEN r.position = 1 THEN 1 ELSE 0 END) AS wins,
    SUM(CASE WHEN r.position BETWEEN 1 AND 3 THEN 1 ELSE 0 END) AS podiums,
    ROUND(SUM(r.points), 1) AS points
FROM results r
JOIN drivers d ON d.id = r.driver_id
JOIN races ra ON ra.id = r.race_id
WHERE r.driver_id = :driver_id %s
GROUP BY r.driver_id
UNION ALL
SELECT
    c.name AS subject,
    'Team' AS kind,
    COUNT(DISTINCT r.race_id) AS races,
    SUM(CASE WHEN r.position = 1 THEN 1 ELSE 0 END) AS wins,
    SUM(CASE WHEN r.position BETWEEN 1 AND 3 THEN 1 ELSE 0 END) AS podiums,
    ROUND(SUM(r.points), 1) AS points
FROM results r
JOIN constructors c ON c.id = r.constructor_id
JOIN races ra ON ra.id = r.race_id
WHERE r.constructor_id = :constructor_id %s
GROUP BY r.constructor_id`, driverYearFilter, teamYearFilter)
		return sql, "Driver vs team comparison"
	}

	return "", ""
}

func buildSeasonOverviewQuery(e *Entities, args map[string]interface{}) (sql, desc string) {
	if e.Year > 0 {
		args["year"] = e.Year
		sql = `
SELECT ds.position AS pos,
       d.forename || ' ' || d.surname AS driver,
       ds.points,
       ds.wins
FROM driver_standings ds
JOIN drivers d ON d.id = ds.driver_id
JOIN races ra ON ra.id = ds.race_id
WHERE ra.id = (
    SELECT id FROM races WHERE year = :year ORDER BY round DESC LIMIT 1
)
ORDER BY ds.position
LIMIT 20`
		return sql, fmt.Sprintf("%d Season final standings", e.Year)
	}

	sql = `
SELECT ra.year AS season,
       COUNT(DISTINCT ra.id) AS races,
       d.forename || ' ' || d.surname AS champion
FROM races ra
JOIN driver_standings ds ON ds.race_id = ra.id
JOIN drivers d ON d.id = ds.driver_id
WHERE ds.position = 1
  AND ra.round = (SELECT MAX(r2.round) FROM races r2 WHERE r2.year = ra.year)
GROUP BY ra.year
ORDER BY ra.year DESC
LIMIT 20`
	return sql, "Recent seasons and champions"
}
