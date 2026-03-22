package nlp

import "strings"

// Intent represents the type of question being asked.
type Intent string

// Intent constants for all supported query types.
const (
	IntentWins            Intent = "wins"
	IntentPoints          Intent = "points"
	IntentChampionship    Intent = "championship"
	IntentPoles           Intent = "poles"
	IntentDNF             Intent = "dnf"
	IntentPitStops        Intent = "pitstops"
	IntentFastestLap      Intent = "fastest_lap"
	IntentRaceResult      Intent = "race_result"
	IntentDriverInfo      Intent = "driver_info"
	IntentConstructorInfo Intent = "constructor_info"
	IntentCircuitInfo     Intent = "circuit_info"
	IntentTeammates       Intent = "teammates"
	IntentHeadToHead      Intent = "head_to_head"
	IntentSeasonOverview  Intent = "season_overview"
	IntentUnknown         Intent = "unknown"
)

// intentDef groups keywords with a priority tier.
// Higher priority intents (action intents) beat lower priority (info intents)
// when both match. Within the same priority, highest keyword score wins.
type intentDef struct {
	intent   Intent
	keywords []string
	priority int // higher = stronger. Action intents = 2, info intents = 1
}

var intentDefs = []intentDef{
	// Action intents — priority 2
	{IntentWins, []string{"win", "wins", "won", "victories", "victory", "most wins", "first place"}, 2},
	{IntentPoints, []string{"points", "scored", "total points", "score"}, 2},
	{IntentChampionship, []string{"championship", "championships", "champion", "title", "titles", "world champion", "wdc", "wcc"}, 2},
	{IntentPoles, []string{"pole", "poles", "pole position", "pole positions", "qualifying first"}, 2},
	{IntentDNF, []string{"dnf", "retired", "did not finish", "retirement", "crash", "accident", "mechanical"}, 2},
	{IntentPitStops, []string{"pit stop", "pit stops", "pitstop", "pitstops", "pit time", "fastest pit"}, 2},
	{IntentFastestLap, []string{"fastest lap", "best lap", "lap time", "lap times", "fastest time", "lap record"}, 2},
	{IntentRaceResult, []string{"result", "results", "finish", "finished", "race result", "podium", "podiums"}, 2},
	{IntentHeadToHead, []string{"compare", "vs", "versus", "against", "head to head", "compared to", "comparison"}, 2},
	{IntentTeammates, []string{"teammate", "teammates", "team mate", "team mates", "drove for", "drove with"}, 2},
	{IntentSeasonOverview, []string{"season", "overview", "summary", "standings", "ranking", "rankings"}, 2},

	// Info intents — priority 1 (only win if no action intent matches)
	{IntentDriverInfo, []string{"driver", "drivers", "who is", "born", "nationality", "age"}, 1},
	{IntentConstructorInfo, []string{"team", "teams", "constructor", "constructors"}, 1},
	{IntentCircuitInfo, []string{"circuit", "circuits", "track", "tracks", "grand prix", "gp"}, 1},
}

// DetectIntent finds the best matching intent for the given tokens.
// Action intents (wins, points, championships) always beat info intents (driver/circuit info)
// when both match.
func DetectIntent(tokens []string) Intent {
	input := strings.Join(tokens, " ")

	bestIntent := IntentUnknown
	bestScore := 0
	bestPriority := 0

	for _, def := range intentDefs {
		score := 0
		for _, kw := range def.keywords {
			if strings.Contains(input, kw) {
				score += len(kw)
			}
		}
		if score == 0 {
			continue
		}
		// Higher priority always wins. Within same priority, higher score wins.
		if def.priority > bestPriority || (def.priority == bestPriority && score > bestScore) {
			bestPriority = def.priority
			bestScore = score
			bestIntent = def.intent
		}
	}

	return bestIntent
}
