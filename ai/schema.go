package ai

// SchemaDescription is the SQLite schema the LLM is constrained to. Kept compact
// because the small system prompt will not benefit from prompt caching (well
// below the per-model minimum), so verbosity buys nothing here.
const SchemaDescription = `Tables (SQLite, all populated from Ergast F1 CSV data):

circuits(id, ref, name, location, country, lat, lng, alt, url)
constructors(id, ref, name, nationality, url)
drivers(id, ref, number, code, forename, surname, dob, nationality, url)
races(id, year, round, circuit_id, name, date, time, url)
results(id, race_id, driver_id, constructor_id, number, grid, position, position_text, position_order, points, laps, time, milliseconds, fastest_lap, rank, fastest_lap_time, fastest_lap_speed, status_id)
sprint_results(id, race_id, driver_id, constructor_id, number, grid, position, position_text, position_order, points, laps, time, milliseconds, fastest_lap, fastest_lap_time, status_id)
lap_times(race_id, driver_id, lap, position, time, milliseconds)
pit_stops(race_id, driver_id, stop, lap, time, duration, milliseconds)
qualifying(id, race_id, driver_id, constructor_id, number, position, q1, q2, q3)
driver_standings(id, race_id, driver_id, points, position, position_text, wins)
constructor_standings(id, race_id, constructor_id, points, position, position_text, wins)
constructor_results(id, race_id, constructor_id, points, status)
status(id, status)  -- e.g. 'Finished', 'Accident', 'Engine', '+1 Lap'
seasons(year, url)

Conventions:
- position = 0 in results/sprint_results means the driver did not classify (DNF).
- position_text uses 'R', 'D', 'W', 'F', 'N', 'E' for retired/disqualified/etc.
- A DNF filter should be: s.status != 'Finished' AND s.status NOT LIKE '+%Lap%'.
- Championship points include sprint points; SUM both results.points and sprint_results.points.
- Grand prix wins are results.position = 1 (sprint wins are tracked separately).
- driver_standings.position = 1 at the last race of a year means that year's champion.`
