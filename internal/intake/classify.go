package intake

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
	"unicode"

	"pitchboard/internal/model"
	"pitchboard/internal/schedule"
)

// Field is one column role a timetable can use.
type Field struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

// Fields are the dropdown choices for each spreadsheet column.
func Fields() []Field {
	return []Field{
		{ID: "date", Label: "Date"},
		{ID: "day", Label: "Weekday"},
		{ID: "start", Label: "Start time"},
		{ID: "end", Label: "End time"},
		{ID: "duration", Label: "Duration"},
		{ID: "place", Label: "Place"},
		{ID: "use", Label: "Practice, game, or available"},
		{ID: "zone", Label: "Zone"},
		{ID: "home", Label: "Home team"},
		{ID: "away", Label: "Away team"},
		{ID: "age", Label: "Age group"},
		{ID: "notes", Label: "Notes"},
		{ID: "ignore", Label: "Ignore"},
	}
}

// Row is one interpreted line from a timetable.
type Row struct {
	Line     int      `json:"line"`
	Kind     string   `json:"kind"`
	Use      string   `json:"use"`
	Start    string   `json:"start"`
	End      string   `json:"end"`
	Place    string   `json:"place"`
	VenueID  string   `json:"venueId"`
	PitchID  string   `json:"pitchId"`
	Zone     string   `json:"zone"`
	Holder   string   `json:"holder"`
	Age      string   `json:"age"`
	Notes    string   `json:"notes"`
	TeamIDs  []string `json:"teamIds"`
	Opponent string   `json:"opponent"`
	HomeAway string   `json:"homeAway"`
	Summary  string   `json:"summary"`
	Problem  string   `json:"problem"`
	Key      string   `json:"key"`
}

// Result is a preview or the outcome of adding those rows.
type Result struct {
	Headers  []string          `json:"headers"`
	Mapping  map[string]string `json:"mapping"`
	Fields   []Field           `json:"fields"`
	Bookings int               `json:"bookings"`
	Open     int               `json:"open"`
	Swaps    int               `json:"swaps"`
	Skipped  int               `json:"skipped"`
	Rows     []Row             `json:"rows"`
}

// Options describe how to read one upload.
type Options struct {
	Source  string
	WeekOf  time.Time
	Mapping map[string]string
}

// Classify reads a grid into bookings, open times, and swaps. It does not change the database.
func Classify(db *model.Database, grid [][]string, opt Options) (Result, error) {
	headers, body, err := splitHeader(grid)
	if err != nil {
		return Result{}, err
	}
	mapping := mergeMapping(headers, opt.Mapping)
	rows := make([]Row, 0, len(body))
	res := Result{Headers: headers, Mapping: mapping, Fields: Fields()}
	for i, raw := range body {
		row := interpret(db, headers, mapping, raw, i+2, opt)
		switch row.Kind {
		case "booking":
			res.Bookings++
		case "open":
			res.Open++
		case "swap":
			res.Swaps++
		default:
			res.Skipped++
			row.Kind = "skip"
		}
		rows = append(rows, row)
	}
	res.Rows = rows
	return res, nil
}

// BookChoice is the team and place picked for an open time.
type BookChoice struct {
	TeamID   string
	Use      string
	PitchID  string
	HomeAway string
	Opponent string
}

// EventForOpening builds a booking from an imported time. The caller checks conflicts and saves it.
func EventForOpening(db *model.Database, op model.Opening, choice BookChoice) (model.Event, error) {
	if choice.TeamID == "" {
		choice.TeamID = op.TeamID
	}
	if _, ok := db.FindTeam(choice.TeamID); !ok || choice.TeamID == "" {
		return model.Event{}, fmt.Errorf("Choose a team.")
	}
	if choice.PitchID == "" {
		choice.PitchID = op.PitchID
	}
	if _, ok := db.FindPitch(choice.PitchID); !ok {
		return model.Event{}, fmt.Errorf("Choose a place.")
	}
	use := choice.Use
	if use == "" || use == "either" {
		use = op.Use
	}
	if use != "game" {
		use = "practice"
	}
	home := choice.HomeAway
	if home != "away" {
		home = "home"
	}
	opponent := strings.TrimSpace(choice.Opponent)
	if opponent == "" && use == "game" {
		opponent = op.Holder
	}
	row := Row{
		Use: use, Start: op.Start, End: op.End, PitchID: choice.PitchID,
		TeamIDs: []string{choice.TeamID}, Opponent: opponent, HomeAway: home, Notes: op.Notes,
	}
	return bookingEvent(db, row, op.Source), nil
}

// Commit adds the classified rows. A booking that does not fit becomes an open time instead.
func Commit(db *model.Database, grid [][]string, opt Options) (Result, error) {
	res, err := Classify(db, grid, opt)
	if err != nil {
		return Result{}, err
	}
	if opt.Source == "" {
		opt.Source = "Saskatoon Youth Soccer"
	}
	kept := make([]Row, 0, len(res.Rows))
	res.Bookings, res.Open, res.Swaps, res.Skipped = 0, 0, 0, 0
	for _, row := range res.Rows {
		if row.Kind == "skip" {
			res.Skipped++
			kept = append(kept, row)
			continue
		}
		if row.Kind == "booking" {
			if seenBooking(db, row) {
				row.Kind = "skip"
				row.Problem = "Already on the calendar."
				row.Summary = "Skipped · " + row.Problem
				res.Skipped++
				kept = append(kept, row)
				continue
			}
			ev := bookingEvent(db, row, opt.Source)
			checked := schedule.Check(db, ev, schedule.WriteOptions{AcknowledgeLate: true})
			if checked.OK {
				ev, _ = schedule.Normalize(db, ev)
				ev.ID = db.NewID("evt")
				now := time.Now().Format(time.RFC3339)
				ev.CreatedAt, ev.UpdatedAt = now, now
				db.Events = append(db.Events, ev)
				res.Bookings++
				kept = append(kept, row)
				continue
			}
			row.Kind = "open"
			if len(checked.Errors()) > 0 {
				row.Problem = checked.Errors()[0].Message
			} else {
				row.Problem = "That time could not be booked."
			}
			row.Summary = "Open time · " + row.Problem
		}
		if seenOpening(db, row) {
			row.Kind = "skip"
			row.Problem = "Already imported."
			row.Summary = "Skipped · " + row.Problem
			res.Skipped++
			kept = append(kept, row)
			continue
		}
		db.Openings = append(db.Openings, openingFrom(db, row, opt.Source))
		if row.Kind == "swap" {
			res.Swaps++
		} else {
			row.Kind = "open"
			res.Open++
		}
		kept = append(kept, row)
	}
	res.Rows = kept
	return res, nil
}

func splitHeader(grid [][]string) (headers []string, body [][]string, err error) {
	start := -1
	for i, row := range grid {
		filled := 0
		for _, cell := range row {
			if strings.TrimSpace(cell) != "" {
				filled++
			}
		}
		if filled >= 2 {
			start = i
			break
		}
	}
	if start < 0 {
		return nil, nil, fmt.Errorf("the sheet needs a header row and at least one time")
	}
	headers = uniqueHeaders(grid[start])
	body = grid[start+1:]
	if len(body) == 0 {
		return nil, nil, fmt.Errorf("the sheet has headers but no times")
	}
	if len(body) > 500 {
		return nil, nil, fmt.Errorf("a timetable can have at most 500 rows")
	}
	return headers, body, nil
}

func uniqueHeaders(row []string) []string {
	seen := map[string]int{}
	out := make([]string, len(row))
	for i, cell := range row {
		h := strings.TrimSpace(cell)
		if h == "" {
			h = fmt.Sprintf("Column %d", i+1)
		}
		seen[h]++
		if seen[h] > 1 {
			h = fmt.Sprintf("%s (%d)", h, seen[h])
		}
		out[i] = h
	}
	return out
}

func mergeMapping(headers []string, given map[string]string) map[string]string {
	guessed := guessMapping(headers)
	out := map[string]string{}
	for _, h := range headers {
		if field, ok := given[h]; ok && knownField(field) {
			out[h] = field
			continue
		}
		out[h] = guessed[h]
	}
	return out
}

func knownField(field string) bool {
	for _, item := range Fields() {
		if item.ID == field {
			return true
		}
	}
	return false
}

func guessMapping(headers []string) map[string]string {
	out := map[string]string{}
	used := map[string]bool{}
	for _, h := range headers {
		field := alias(h)
		if field == "" || used[field] {
			out[h] = "ignore"
			continue
		}
		used[field] = true
		out[h] = field
	}
	return out
}

func alias(header string) string {
	h := norm(header)
	switch h {
	case "date", "start date", "game date", "session date":
		return "date"
	case "day", "weekday", "day of week":
		return "day"
	case "start", "start time", "time", "from", "begin":
		return "start"
	case "end", "end time", "to", "finish", "until":
		return "end"
	case "duration", "length", "minutes", "mins":
		return "duration"
	case "location", "venue", "field", "facility", "pitch", "arena", "site", "park", "place":
		return "place"
	case "type", "event type", "activity", "event", "use", "category", "session":
		return "use"
	case "zone", "club", "association", "district":
		return "zone"
	case "home", "home team", "team", "teams", "our team":
		return "home"
	case "away", "away team", "opponent", "visitor", "visiting team":
		return "away"
	case "age", "age group", "division", "group":
		return "age"
	case "notes", "note", "comments", "comment", "details":
		return "notes"
	default:
		return ""
	}
}

func interpret(db *model.Database, headers []string, mapping map[string]string, raw []string, line int, opt Options) Row {
	got := map[string]string{}
	for i, header := range headers {
		field := mapping[header]
		if field == "" || field == "ignore" || i >= len(raw) {
			continue
		}
		if strings.TrimSpace(got[field]) == "" {
			got[field] = strings.TrimSpace(raw[i])
		}
	}
	row := Row{Line: line, HomeAway: "home", Use: "either"}
	start, end, problem := when(got, opt.WeekOf)
	row.Place = got["place"]
	row.Zone = canonicalZone(got["zone"])
	row.Age = got["age"]
	row.Notes = got["notes"]
	row.Use = parseUse(got["use"])
	if problem != "" {
		row.Problem = problem
		row.Summary = fmt.Sprintf("Skipped · row %d %s", line, problem)
		return row
	}
	row.Start = model.FormatWall(start)
	row.End = model.FormatWall(end)
	if row.Use == "either" && got["use"] == "" && (got["home"] != "" || got["away"] != "") {
		if got["home"] != "" && got["away"] != "" {
			row.Use = "game"
		} else {
			row.Use = "practice"
		}
	}
	venueID, pitchID := matchPlace(db, row.Place)
	row.VenueID, row.PitchID = venueID, pitchID
	homeTeams := matchTeams(db, got["home"])
	awayTeams := matchTeams(db, got["away"])
	row.TeamIDs = append(uniqueIDs(homeTeams), uniqueIDs(awayTeams)...)
	row.TeamIDs = uniqueStrings(row.TeamIDs)
	if len(homeTeams) > 1 || len(awayTeams) > 1 {
		row.Problem = "More than one club team matches that name."
		row.TeamIDs = nil
	}
	foreign := row.Zone != "" && !zoneIsOurs(row.Zone, db.Settings.ZoneName, db.Settings.ClubName)
	holder := holderText(got["home"], got["away"], row.Zone, len(homeTeams) == 1, len(awayTeams) == 1)
	named := strings.TrimSpace(got["home"]) != "" || strings.TrimSpace(got["away"]) != ""
	switch {
	case len(row.TeamIDs) > 0:
		row.Kind = "booking"
		if len(awayTeams) == 1 && len(homeTeams) == 0 {
			row.HomeAway = "away"
			row.Opponent = strings.TrimSpace(got["home"])
		} else if strings.TrimSpace(got["away"]) != "" && len(awayTeams) == 0 {
			row.Opponent = strings.TrimSpace(got["away"])
		}
		if row.PitchID == "" {
			row.Kind = "open"
			row.Problem = placeProblem(row.Place)
		}
	case foreign || named:
		row.Kind = "swap"
		row.Holder = holder
		if row.Use == "either" {
			row.Use = "game"
		}
	default:
		row.Kind = "open"
	}
	if row.Kind == "open" && row.Holder == "" && holder != "" && len(row.TeamIDs) == 0 {
		row.Holder = holder
	}
	row.Key = slotKey(row.Start, row.End, row.Place, row.Holder)
	row.Summary = summarize(db, row)
	return row
}

func when(got map[string]string, weekOf time.Time) (time.Time, time.Time, string) {
	startText := got["start"]
	if startText == "" {
		return time.Time{}, time.Time{}, "has no start time"
	}
	if a, b, ok := splitRange(startText); ok && got["end"] == "" {
		startText, got = a, clone(got)
		got["end"] = b
	}
	day := time.Time{}
	var err error
	if got["date"] != "" {
		if _, ok := weekday(got["date"]); ok {
			day, err = weekdayDate(got["date"], weekOf)
		} else {
			day, err = parseDate(got["date"])
		}
		if err != nil {
			return time.Time{}, time.Time{}, "has a date Pitchboard cannot read"
		}
	} else if got["day"] != "" {
		day, err = weekdayDate(got["day"], weekOf)
		if err != nil {
			return time.Time{}, time.Time{}, err.Error()
		}
	} else if whole, ok := parseDateTime(startText); ok {
		end := whole.Add(time.Hour)
		if got["end"] != "" {
			if clock, ok := parseClock(got["end"]); ok {
				end = time.Date(whole.Year(), whole.Month(), whole.Day(), clock/60, clock%60, 0, 0, time.Local)
			}
		}
		if !end.After(whole) {
			return time.Time{}, time.Time{}, "ends before it starts"
		}
		return whole, end, ""
	} else {
		return time.Time{}, time.Time{}, "has no date"
	}
	clock, ok := parseClock(startText)
	if !ok {
		if whole, ok := parseDateTime(startText); ok {
			clock = whole.Hour()*60 + whole.Minute()
		} else {
			return time.Time{}, time.Time{}, "has a start time Pitchboard cannot read"
		}
	}
	start := time.Date(day.Year(), day.Month(), day.Day(), clock/60, clock%60, 0, 0, time.Local)
	end := start
	if got["end"] != "" {
		if endClock, ok := parseClock(got["end"]); ok {
			end = time.Date(day.Year(), day.Month(), day.Day(), endClock/60, endClock%60, 0, 0, time.Local)
		} else {
			return time.Time{}, time.Time{}, "has an end time Pitchboard cannot read"
		}
	} else if mins, ok := parseDuration(got["duration"]); ok {
		end = start.Add(time.Duration(mins) * time.Minute)
	} else {
		end = start.Add(time.Hour)
	}
	if !end.After(start) {
		return time.Time{}, time.Time{}, "ends before it starts"
	}
	return start, end, ""
}

func clone(in map[string]string) map[string]string {
	out := map[string]string{}
	for k, v := range in {
		out[k] = v
	}
	return out
}

func splitRange(value string) (string, string, bool) {
	for _, sep := range []string{"–", "—", " - ", "-"} {
		parts := strings.Split(value, sep)
		if len(parts) != 2 {
			continue
		}
		a, b := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
		if _, ok := parseClock(a); !ok {
			continue
		}
		if _, ok := parseClock(b); !ok {
			continue
		}
		if !hasMeridiem(a) && hasMeridiem(b) {
			if strings.Contains(strings.ToUpper(b), "PM") {
				a += " PM"
			} else {
				a += " AM"
			}
		}
		return a, b, true
	}
	return "", "", false
}

func parseUse(value string) string {
	v := norm(value)
	switch {
	case v == "":
		return "either"
	case strings.Contains(v, "available") || strings.Contains(v, "open") || v == "free" || strings.Contains(v, "spare") || strings.Contains(v, "unused"):
		return "either"
	case strings.Contains(v, "game") || strings.Contains(v, "match") || strings.Contains(v, "fixture"):
		return "game"
	case strings.Contains(v, "camp"):
		return "practice"
	default:
		return "practice"
	}
}

func canonicalZone(value string) string {
	v := norm(value)
	switch {
	case v == "":
		return ""
	case strings.Contains(v, "aurora") || strings.Contains(v, "zone 3") || strings.Contains(v, "zone3"):
		return "Aurora"
	case strings.Contains(v, "eastside") || strings.Contains(v, "zone 5") || strings.Contains(v, "zone5"):
		return "Eastside"
	case strings.Contains(v, "lakewood") || strings.Contains(v, "zone 4") || strings.Contains(v, "zone4"):
		return "Lakewood"
	case strings.Contains(v, "hollandia") || strings.Contains(v, "zone 2") || strings.Contains(v, "zone2"):
		return "Hollandia"
	case strings.Contains(v, "susc") || strings.Contains(v, "saskatoon united") || strings.Contains(v, "zone 1") || strings.Contains(v, "zone1"):
		return "SUSC"
	default:
		return strings.TrimSpace(value)
	}
}

func zoneIsOurs(zone, ours, club string) bool {
	if zone == "" {
		return true
	}
	zone = canonicalZone(zone)
	ours = canonicalZone(ours)
	if ours == "" {
		ours = "Aurora"
	}
	if strings.EqualFold(zone, ours) {
		return true
	}
	club = canonicalZone(club)
	return club != "" && strings.EqualFold(zone, club)
}

func matchTeams(db *model.Database, name string) []string {
	name = norm(name)
	if name == "" || db == nil {
		return nil
	}
	var ids []string
	seen := map[string]bool{}
	for _, team := range db.Teams {
		if !team.Active {
			continue
		}
		for _, label := range []string{team.Name, team.ShortName} {
			if containsName(name, norm(label)) && !seen[team.ID] {
				seen[team.ID] = true
				ids = append(ids, team.ID)
			}
		}
	}
	return ids
}

func matchPlace(db *model.Database, text string) (string, string) {
	text = norm(text)
	if text == "" || db == nil {
		return "", ""
	}
	bestScore := 0
	bestPitch := ""
	bestVenue := ""
	tied := false
	for _, pitch := range db.Pitches {
		if !pitch.Active {
			continue
		}
		venue, _ := db.FindVenue(pitch.VenueID)
		score := 0
		if containsPhrase(text, norm(pitch.Name)) && len(norm(pitch.Name)) >= 3 {
			score += 2
		}
		if containsPhrase(text, norm(venue.Name)) && len(norm(venue.Name)) >= 4 {
			score += 3
		}
		if score == 0 || score < bestScore {
			continue
		}
		if score == bestScore {
			tied = true
			continue
		}
		tied = false
		bestScore = score
		bestPitch = pitch.ID
		bestVenue = pitch.VenueID
	}
	if bestScore >= 2 && !tied {
		return bestVenue, bestPitch
	}
	if bestScore >= 3 && tied {
		return bestVenue, ""
	}
	return "", ""
}

func containsName(hay, needle string) bool {
	if needle == "" {
		return false
	}
	if hay == needle {
		return true
	}
	// A short label such as U12 must match the whole cell, so "Eastside U12" is not this club's U12.
	if !strings.Contains(needle, " ") && len([]rune(needle)) <= 4 {
		return false
	}
	return containsPhrase(hay, needle)
}

func containsPhrase(hay, needle string) bool {
	if needle == "" {
		return false
	}
	if hay == needle {
		return true
	}
	i := strings.Index(hay, needle)
	if i < 0 {
		return false
	}
	if i > 0 && isNameChar(rune(hay[i-1])) {
		return false
	}
	after := i + len(needle)
	if after < len(hay) && isNameChar(rune(hay[after])) {
		return false
	}
	return true
}

func hasMeridiem(value string) bool {
	upper := strings.ToUpper(value)
	return strings.Contains(upper, "AM") || strings.Contains(upper, "PM")
}

func isNameChar(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r)
}

func holderText(home, away, zone string, homeOurs, awayOurs bool) string {
	home, away = strings.TrimSpace(home), strings.TrimSpace(away)
	var parts []string
	if home != "" && !homeOurs {
		parts = append(parts, home)
	}
	if away != "" && !awayOurs {
		if len(parts) == 1 {
			parts = append(parts, away)
			return parts[0] + " vs " + parts[1]
		}
		parts = append(parts, away)
	}
	if len(parts) == 2 {
		return parts[0] + " vs " + parts[1]
	}
	if len(parts) == 1 {
		if zone != "" && !strings.Contains(norm(parts[0]), norm(zone)) {
			return zone + " · " + parts[0]
		}
		return parts[0]
	}
	return zone
}

func placeProblem(place string) string {
	if strings.TrimSpace(place) == "" {
		return "The sheet has no place. Pick one when you book it."
	}
	return "No pitch matches " + place + ". Pick a place when you book it."
}

func summarize(db *model.Database, row Row) string {
	when := ""
	if start, err := model.ParseFlexible(row.Start); err == nil {
		end, _ := model.ParseFlexible(row.End)
		when = start.Format("Mon Jan 2") + ", " + model.FormatWhen(start, "12") + "–" + model.FormatWhen(end, "12")
	}
	where := row.Place
	if row.PitchID != "" && db != nil {
		if pitch, ok := db.FindPitch(row.PitchID); ok {
			venue, _ := db.FindVenue(pitch.VenueID)
			where = strings.TrimSpace(venue.Name + " " + pitch.Name)
		}
	}
	switch row.Kind {
	case "booking":
		return joinBits("Booking", teamLabel(db, row.TeamIDs), where, when)
	case "swap":
		return joinBits("Swap with "+blank(row.Zone, "another zone"), row.Holder, where, when)
	case "open":
		label := "Open time"
		if row.Use == "practice" {
			label = "Open practice"
		} else if row.Use == "game" {
			label = "Open game"
		}
		if row.Problem != "" {
			return joinBits(label, where, when) + " · " + row.Problem
		}
		return joinBits(label, where, when)
	default:
		if row.Summary != "" {
			return row.Summary
		}
		return "Skipped"
	}
}

func teamLabel(db *model.Database, ids []string) string {
	if db == nil {
		return ""
	}
	var names []string
	for _, id := range ids {
		if team, ok := db.FindTeam(id); ok {
			names = append(names, team.Name)
		}
	}
	return strings.Join(names, ", ")
}

func joinBits(parts ...string) string {
	var out []string
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return strings.Join(out, " · ")
}

func blank(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func bookingEvent(db *model.Database, row Row, source string) model.Event {
	use := row.Use
	if use != "game" {
		use = "practice"
	}
	ev := model.Event{
		EventTypeID: typeID(db, use),
		TeamIDs:     append([]string{}, row.TeamIDs...),
		VenueID:     row.VenueID,
		PitchID:     row.PitchID,
		HomeAway:    row.HomeAway,
		Opponent:    row.Opponent,
		Start:       row.Start,
		End:         row.End,
		Status:      "scheduled",
		Notes:       withSource(row.Notes, source),
	}
	if pitch, ok := db.FindPitch(row.PitchID); ok {
		ev.SurfaceID = pitch.SurfaceID
		ev.Environment = pitch.Environment
		ev.FormatID = pitch.FormatID
		ev.VenueID = pitch.VenueID
	}
	return ev
}

func openingFrom(db *model.Database, row Row, source string) model.Opening {
	notes := withSource(row.Notes, source)
	if row.Problem != "" {
		if notes != "" {
			notes += " "
		}
		notes += row.Problem
	}
	op := model.Opening{
		ID:      db.NewID("opn"),
		Source:  source,
		Kind:    row.Kind,
		Use:     row.Use,
		Start:   row.Start,
		End:     row.End,
		Place:   row.Place,
		VenueID: row.VenueID,
		PitchID: row.PitchID,
		Zone:    row.Zone,
		Holder:  row.Holder,
		Age:     row.Age,
		Notes:   strings.TrimSpace(notes),
		Status:  "open",
		Key:     row.Key,
	}
	if len(row.TeamIDs) == 1 {
		op.TeamID = row.TeamIDs[0]
	}
	if pitch, ok := db.FindPitch(op.PitchID); ok {
		op.SurfaceID = pitch.SurfaceID
		op.Environment = pitch.Environment
		op.VenueID = pitch.VenueID
	}
	if op.Kind != "swap" {
		op.Kind = "open"
	}
	return op
}

func withSource(notes, source string) string {
	notes = strings.TrimSpace(notes)
	source = strings.TrimSpace(source)
	if source == "" {
		return notes
	}
	if notes == "" {
		return "From " + source
	}
	if strings.Contains(strings.ToLower(notes), strings.ToLower(source)) {
		return notes
	}
	return notes + " · From " + source
}

func typeID(db *model.Database, category string) string {
	for _, item := range db.EventTypes {
		if item.Category == category {
			return item.ID
		}
	}
	if len(db.EventTypes) > 0 {
		return db.EventTypes[0].ID
	}
	return ""
}

func seenBooking(db *model.Database, row Row) bool {
	for _, ev := range db.Events {
		if ev.Status == "cancelled" || ev.Start != row.Start {
			continue
		}
		for _, id := range row.TeamIDs {
			for _, have := range ev.TeamIDs {
				if id == have {
					return true
				}
			}
		}
	}
	return false
}

func seenOpening(db *model.Database, row Row) bool {
	for _, op := range db.Openings {
		if op.Key != "" && op.Key == row.Key {
			return true
		}
	}
	return false
}

func slotKey(start, end, place, holder string) string {
	return norm(start) + "|" + norm(end) + "|" + norm(place) + "|" + norm(holder)
}

func firstID(groups ...[]string) []string {
	for _, group := range groups {
		if len(group) == 1 {
			return []string{group[0]}
		}
	}
	return nil
}

func uniqueIDs(ids []string) []string { return ids }

func uniqueStrings(ids []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, id := range ids {
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out
}

func norm(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var b strings.Builder
	space := false
	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			space = false
			continue
		}
		if !space && b.Len() > 0 {
			b.WriteByte(' ')
			space = true
		}
	}
	return strings.TrimSpace(b.String())
}

func parseDate(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if n, ok := numeric(value); ok && n > 20000 && n < 80000 {
		return excelDay(n), nil
	}
	layouts := []string{
		"2006-01-02", "2006/01/02", "02-Jan-2006", "2-Jan-2006",
		"Jan 2, 2006", "January 2, 2006", "Jan 2 2006", "January 2 2006",
		"2 Jan 2006", "2 January 2006",
	}
	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, value, time.Local); err == nil {
			return t, nil
		}
	}
	if t, ok := slashDate(value); ok {
		return t, nil
	}
	return time.Time{}, fmt.Errorf("bad date")
}

func slashDate(value string) (time.Time, bool) {
	parts := strings.FieldsFunc(value, func(r rune) bool { return r == '/' || r == '-' || r == '.' })
	if len(parts) != 3 {
		return time.Time{}, false
	}
	a, err1 := strconv.Atoi(parts[0])
	b, err2 := strconv.Atoi(parts[1])
	y, err3 := strconv.Atoi(parts[2])
	if err1 != nil || err2 != nil || err3 != nil {
		return time.Time{}, false
	}
	if y < 100 {
		y += 2000
	}
	day, month := a, b
	if a <= 12 && b > 12 {
		month, day = a, b
	}
	t := time.Date(y, time.Month(month), day, 0, 0, 0, 0, time.Local)
	if t.Month() != time.Month(month) || t.Day() != day {
		return time.Time{}, false
	}
	return t, true
}

func weekdayDate(name string, weekOf time.Time) (time.Time, error) {
	if weekOf.IsZero() {
		return time.Time{}, fmt.Errorf("needs a week, because the sheet says a weekday instead of a date")
	}
	want, ok := weekday(name)
	if !ok {
		return time.Time{}, fmt.Errorf("has a weekday Pitchboard cannot read")
	}
	monday := weekOf
	for monday.Weekday() != time.Monday {
		monday = monday.AddDate(0, 0, -1)
	}
	return monday.AddDate(0, 0, int(want-time.Monday+7)%7), nil
}

func weekday(name string) (time.Weekday, bool) {
	switch norm(name) {
	case "monday", "mon":
		return time.Monday, true
	case "tuesday", "tue", "tues":
		return time.Tuesday, true
	case "wednesday", "wed":
		return time.Wednesday, true
	case "thursday", "thu", "thur", "thurs":
		return time.Thursday, true
	case "friday", "fri":
		return time.Friday, true
	case "saturday", "sat":
		return time.Saturday, true
	case "sunday", "sun":
		return time.Sunday, true
	default:
		return 0, false
	}
}

func parseClock(value string) (int, bool) {
	value = strings.TrimSpace(value)
	if n, ok := numeric(value); ok {
		if n >= 0 && n < 1 {
			return int(math.Round(n * 24 * 60)), true
		}
		if n > 1 && n < 80000 {
			frac := n - math.Floor(n)
			return int(math.Round(frac * 24 * 60)), true
		}
	}
	upper := strings.ToUpper(value)
	ampm := 0
	if strings.HasSuffix(upper, "AM") || strings.HasSuffix(upper, "A.M.") {
		ampm = -1
		upper = strings.TrimSpace(strings.TrimSuffix(strings.TrimSuffix(upper, "A.M."), "AM"))
	} else if strings.HasSuffix(upper, "PM") || strings.HasSuffix(upper, "P.M.") {
		ampm = 1
		upper = strings.TrimSpace(strings.TrimSuffix(strings.TrimSuffix(upper, "P.M."), "PM"))
	}
	parts := strings.Split(upper, ":")
	if len(parts) < 1 || len(parts) > 3 {
		return 0, false
	}
	h, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return 0, false
	}
	m := 0
	if len(parts) > 1 && strings.TrimSpace(parts[1]) != "" {
		m, err = strconv.Atoi(strings.TrimSpace(parts[1]))
		if err != nil {
			return 0, false
		}
	}
	if ampm != 0 {
		if h < 1 || h > 12 || m < 0 || m > 59 {
			return 0, false
		}
		if ampm < 0 && h == 12 {
			h = 0
		}
		if ampm > 0 && h != 12 {
			h += 12
		}
		return h*60 + m, true
	}
	if h < 0 || h > 23 || m < 0 || m > 59 {
		return 0, false
	}
	return h*60 + m, true
}

func parseDuration(value string) (int, bool) {
	value = strings.TrimSpace(strings.ToLower(value))
	value = strings.TrimSuffix(value, "minutes")
	value = strings.TrimSuffix(value, "minute")
	value = strings.TrimSuffix(value, "mins")
	value = strings.TrimSuffix(value, "min")
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false
	}
	if strings.Contains(value, ":") {
		clock, ok := parseClock(value)
		if !ok || clock <= 0 || clock > 12*60 {
			return 0, false
		}
		return clock, true
	}
	n, err := strconv.Atoi(value)
	if err != nil || n <= 0 || n > 12*60 {
		return 0, false
	}
	return n, true
}

func parseDateTime(value string) (time.Time, bool) {
	value = strings.TrimSpace(value)
	if n, ok := numeric(value); ok && n > 20000 && n < 80000 {
		day := excelDay(n)
		frac := n - math.Floor(n)
		mins := int(math.Round(frac * 24 * 60))
		return time.Date(day.Year(), day.Month(), day.Day(), mins/60, mins%60, 0, 0, time.Local), true
	}
	for _, layout := range []string{"2006-01-02 15:04", "2006-01-02 3:04 PM", "2006-01-02 3:04PM"} {
		if t, err := time.ParseInLocation(layout, value, time.Local); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

func numeric(value string) (float64, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false
	}
	n, err := strconv.ParseFloat(value, 64)
	return n, err == nil
}

func excelDay(serial float64) time.Time {
	base := time.Date(1899, 12, 30, 0, 0, 0, 0, time.UTC)
	return base.AddDate(0, 0, int(math.Floor(serial)))
}
