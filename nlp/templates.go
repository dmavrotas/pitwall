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

// BuildQuery generates the SQL query and description for a given intent and entities.
func BuildQuery(intent Intent, e Entities) (sql, desc string, args map[string]interface{}) {
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

func buildWinsQuery(e Entities, args map[string]interface{}) (sql, desc string) {
	var filters []string
	desc ="Top drivers by race wins"

	if e.Year > 0 {
		filters = append(filters, "ra.year = :year")
		args["year"] = e.Year
		desc = fmt.Sprintf("Top drivers by race wins in %d", e.Year)
	}
	if e.DriverID > 0 {
		filters = append(filters, "r.driver_id = :driver_id")
		args["driver_id"] = e.DriverID
		desc = "Race wins for driver"
		if e.Year > 0 {
			desc += fmt.Sprintf(" in %d", e.Year)
		}
	}
	if e.ConstructorID > 0 {
		filters = append(filters, "r.constructor_id = :constructor_id")
		args["constructor_id"] = e.ConstructorID
		desc = "Top drivers by wins with constructor"
	}
	if e.CircuitID > 0 {
		filters = append(filters, "ra.circuit_id = :circuit_id")
		args["circuit_id"] = e.CircuitID
		desc = "Top drivers by wins at circuit"
		if e.Year > 0 {
			desc += fmt.Sprintf(" in %d", e.Year)
		}
	}

	where := "r.position = 1"
	if len(filters) > 0 {
		where += " AND " + strings.Join(filters, " AND ")
	}

	sql =fmt.Sprintf(`
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

func buildPointsQuery(e Entities, args map[string]interface{}) (sql, desc string) {
	var filters []string
	desc ="Top drivers by total points"

	if e.Year > 0 {
		filters = append(filters, "ra.year = :year")
		args["year"] = e.Year
		desc = fmt.Sprintf("Top drivers by points in %d", e.Year)
	}
	if e.DriverID > 0 {
		filters = append(filters, "r.driver_id = :driver_id")
		args["driver_id"] = e.DriverID
		desc = "Points scored by driver"
		if e.Year > 0 {
			desc += fmt.Sprintf(" in %d", e.Year)
		}
	}
	if e.ConstructorID > 0 {
		filters = append(filters, "r.constructor_id = :constructor_id")
		args["constructor_id"] = e.ConstructorID
	}

	where := "1=1"
	if len(filters) > 0 {
		where = strings.Join(filters, " AND ")
	}

	sql =fmt.Sprintf(`
SELECT d.forename || ' ' || d.surname AS driver,
       c.name AS team,
       SUM(r.points) AS total_points,
       COUNT(*) AS races
FROM results r
JOIN drivers d ON d.id = r.driver_id
JOIN constructors c ON c.id = r.constructor_id
JOIN races ra ON ra.id = r.race_id
WHERE %s
GROUP BY r.driver_id
ORDER BY total_points DESC
LIMIT 15`, where)

	return sql, desc
}

func buildChampionshipQuery(e Entities, args map[string]interface{}) (sql, desc string) {
	// Specific driver's championship history
	if e.DriverID > 0 {
		args["driver_id"] = e.DriverID
		var yearFilter string
		if e.Year > 0 {
			yearFilter = "AND ra.year = :year"
			args["year"] = e.Year
		}
		sql =fmt.Sprintf(`
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
ORDER BY ra.year`, yearFilter)
		return sql, "Driver championship history"
	}

	// Constructor championships
	if e.ConstructorID > 0 {
		args["constructor_id"] = e.ConstructorID
		sql =`
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

	// Season standings
	if e.Year > 0 {
		args["year"] = e.Year
		sql =`
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
LIMIT 15`
		return sql, fmt.Sprintf("Championship standings %d", e.Year)
	}

	// All-time championship winners
	sql =`
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

func buildPolesQuery(e Entities, args map[string]interface{}) (sql, desc string) {
	var filters []string
	desc ="Top drivers by pole positions"

	if e.Year > 0 {
		filters = append(filters, "ra.year = :year")
		args["year"] = e.Year
		desc = fmt.Sprintf("Pole positions in %d", e.Year)
	}
	if e.DriverID > 0 {
		filters = append(filters, "q.driver_id = :driver_id")
		args["driver_id"] = e.DriverID
	}

	where := "q.position = 1"
	if len(filters) > 0 {
		where += " AND " + strings.Join(filters, " AND ")
	}

	sql =fmt.Sprintf(`
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

func buildDNFQuery(e Entities, args map[string]interface{}) (sql, desc string) {
	var filters []string
	desc ="Top DNF reasons"

	if e.Year > 0 {
		filters = append(filters, "ra.year = :year")
		args["year"] = e.Year
		desc = fmt.Sprintf("DNF reasons in %d", e.Year)
	}
	if e.DriverID > 0 {
		filters = append(filters, "r.driver_id = :driver_id")
		args["driver_id"] = e.DriverID
		desc = "DNF history for driver"
	}

	where := "s.status != 'Finished' AND r.position = 0"
	if len(filters) > 0 {
		where += " AND " + strings.Join(filters, " AND ")
	}

	sql =fmt.Sprintf(`
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

func buildPitStopsQuery(e Entities, args map[string]interface{}) (sql, desc string) {
	var filters []string
	desc ="Average pit stop duration by year"

	if e.Year > 0 {
		filters = append(filters, "ra.year = :year")
		args["year"] = e.Year
		desc = fmt.Sprintf("Fastest pit stops in %d", e.Year)

		where := strings.Join(filters, " AND ")
		sql =fmt.Sprintf(`
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
LIMIT 15`, where)
		return sql, desc
	}

	sql =`
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

func buildFastestLapQuery(e Entities, args map[string]interface{}) (sql, desc string) {
	var filters []string
	desc ="Fastest lap times"

	if e.Year > 0 {
		filters = append(filters, "ra.year = :year")
		args["year"] = e.Year
		desc = fmt.Sprintf("Fastest laps in %d", e.Year)
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

	sql =fmt.Sprintf(`
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

func buildRaceResultQuery(e Entities, args map[string]interface{}) (sql, desc string) {
	var filters []string
	desc ="Race results"

	if e.Year > 0 {
		filters = append(filters, "ra.year = :year")
		args["year"] = e.Year
		desc = fmt.Sprintf("Race results in %d", e.Year)
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

	// If we have a specific race (circuit+year), show full result
	if e.CircuitID > 0 || (e.DriverID > 0 && e.Year > 0) {
		sql =fmt.Sprintf(`
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
	sql =fmt.Sprintf(`
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

func buildDriverInfoQuery(e Entities, args map[string]interface{}) (sql, desc string) {
	if e.DriverID > 0 {
		args["driver_id"] = e.DriverID
		sql =`
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

	sql =`
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

func buildConstructorInfoQuery(e Entities, args map[string]interface{}) (sql, desc string) {
	if e.ConstructorID > 0 {
		args["constructor_id"] = e.ConstructorID
		sql =`
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

	sql =`
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

func buildCircuitInfoQuery(e Entities, args map[string]interface{}) (sql, desc string) {
	if e.CircuitID > 0 {
		args["circuit_id"] = e.CircuitID
		sql =`
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

	sql =`
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

func buildTeammatesQuery(e Entities, args map[string]interface{}) (sql, desc string) {
	if e.DriverID > 0 {
		args["driver_id"] = e.DriverID
		var yearFilter string
		if e.Year > 0 {
			yearFilter = "AND ra.year = :year"
			args["year"] = e.Year
		}
		sql =fmt.Sprintf(`
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
LIMIT 20`, yearFilter)
		return sql, "Teammates history"
	}

	return "", ""
}

func buildHeadToHeadQuery(e Entities, args map[string]interface{}) (sql, desc string) {
	if e.DriverID > 0 && e.DriverID2 > 0 {
		args["driver1"] = e.DriverID
		args["driver2"] = e.DriverID2
		var yearFilter string
		if e.Year > 0 {
			yearFilter = "AND ra.year = :year"
			args["year"] = e.Year
		}
		sql =fmt.Sprintf(`
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
GROUP BY r.driver_id`, yearFilter)
		return sql, "Head to head comparison"
	}

	if e.ConstructorID > 0 && e.ConstructorID2 > 0 {
		args["team1"] = e.ConstructorID
		args["team2"] = e.ConstructorID2
		var yearFilter string
		if e.Year > 0 {
			yearFilter = "AND ra.year = :year"
			args["year"] = e.Year
		}
		sql =fmt.Sprintf(`
SELECT
    c.name AS team,
    COUNT(DISTINCT r.race_id) AS races,
    SUM(CASE WHEN r.position = 1 THEN 1 ELSE 0 END) AS wins,
    SUM(r.points) AS points
FROM results r
JOIN constructors c ON c.id = r.constructor_id
JOIN races ra ON ra.id = r.race_id
WHERE r.constructor_id IN (:team1, :team2) %s
GROUP BY r.constructor_id`, yearFilter)
		return sql, "Team comparison"
	}

	return "", ""
}

func buildSeasonOverviewQuery(e Entities, args map[string]interface{}) (sql, desc string) {
	if e.Year > 0 {
		args["year"] = e.Year
		sql =`
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

	sql =`
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

