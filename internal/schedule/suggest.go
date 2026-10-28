package schedule

import (
	"fmt"
	"sort"
	"time"

	"pitchboard/internal/model"
)

type SuggestRequest struct {
	TeamIDs         []string `json:"teamIds"`
	EventTypeID     string   `json:"eventTypeId"`
	RangeStart      string   `json:"rangeStart"`
	RangeEnd        string   `json:"rangeEnd"`
	DaysOfWeek      []int    `json:"daysOfWeek"`
	SessionsPerWeek int      `json:"sessionsPerWeek"`
	DurationMinutes int      `json:"durationMinutes"`
	VenueIDs        []string `json:"venueIds"`
	PitchIDs        []string `json:"pitchIds"`
}

type ProposedEvent struct {
	Event    model.Event `json:"event"`
	Warnings []string    `json:"warnings"`
	Score    int         `json:"score"`
}

type Unscheduled struct {
	TeamID string `json:"teamId"`
	Team   string `json:"team"`
	WeekOf string `json:"weekOf"`
	Reason string `json:"reason"`
}

type SuggestResult struct {
	Proposals   []ProposedEvent `json:"proposals"`
	Unscheduled []Unscheduled   `json:"unscheduled"`
}

type pattern struct {
	dow      int
	startMin int
	pitch    model.Pitch
	venue    model.Venue
}

func Suggest(db *model.Database, req SuggestRequest) (SuggestResult, error) {
	if len(req.TeamIDs) == 0 {
		return SuggestResult{}, fmt.Errorf("choose at least one team")
	}
	if _, ok := db.FindEventType(req.EventTypeID); !ok {
		return SuggestResult{}, fmt.Errorf("choose what kind of session this is")
	}
	start, err := model.ParseFlexible(req.RangeStart)
	if err != nil {
		return SuggestResult{}, fmt.Errorf("choose a start date")
	}
	end, err := model.ParseFlexible(req.RangeEnd)
	if err != nil {
		return SuggestResult{}, fmt.Errorf("choose an end date")
	}
	start = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.Local)
	end = time.Date(end.Year(), end.Month(), end.Day(), 0, 0, 0, 0, time.Local)
	if len(req.RangeEnd) == 10 || (end.Hour() == 0 && end.Minute() == 0) {
		end = end.AddDate(0, 0, 1)
	}
	if !end.After(start) {
		return SuggestResult{}, fmt.Errorf("the end date must be after the start")
	}
	if end.Sub(start) > 200*24*time.Hour {
		return SuggestResult{}, fmt.Errorf("plan at most 26 weeks at a time")
	}
	days := uniqueDays(req.DaysOfWeek)
	if len(days) == 0 {
		return SuggestResult{}, fmt.Errorf("choose at least one day of the week")
	}
	if req.SessionsPerWeek <= 0 {
		req.SessionsPerWeek = 1
	}
	if req.SessionsPerWeek > len(days) {
		req.SessionsPerWeek = len(days)
	}
	if req.SessionsPerWeek > 7 {
		req.SessionsPerWeek = 7
	}

	var result SuggestResult
	busy := append([]model.Event{}, db.Events...)
	teams := sortTeams(db, req.TeamIDs)
	for _, team := range teams {
		duration := req.DurationMinutes
		if duration <= 0 {
			duration = practiceMinutes(db, team, req.EventTypeID)
		}
		patterns := enumerate(db, team, duration, days, req.VenueIDs, req.PitchIDs)
		if len(patterns) == 0 {
			result.Unscheduled = append(result.Unscheduled, Unscheduled{
				TeamID: team.ID, Team: team.Name, Reason: "No pitch is open long enough on those days.",
			})
			continue
		}
		usedDays := map[int]bool{}
		placedAny := false
		for session := 0; session < req.SessionsPerWeek; session++ {
			bestScore := -1 << 30
			var best []model.Event
			var bestWarn []string
			var bestDay int
			found := false
			for _, p := range patterns {
				if usedDays[p.dow] {
					continue
				}
				events, warns, score, ok := materialize(db, team, req, p, duration, start, end, busy)
				if !ok {
					continue
				}
				if score > bestScore {
					bestScore = score
					best = events
					bestWarn = warns
					bestDay = p.dow
					found = true
				}
			}
			if !found {
				result.Unscheduled = append(result.Unscheduled, Unscheduled{
					TeamID: team.ID, Team: team.Name, Reason: "No free pitch left at a suitable time.",
				})
				break
			}
			usedDays[bestDay] = true
			placedAny = true
			for i, ev := range best {
				warn := []string{}
				if i < len(bestWarn) && bestWarn[i] != "" {
					warn = []string{bestWarn[i]}
				}
				result.Proposals = append(result.Proposals, ProposedEvent{Event: ev, Warnings: warn, Score: bestScore})
				busy = append(busy, ev)
			}
		}
		if !placedAny && len(result.Unscheduled) == 0 {
			result.Unscheduled = append(result.Unscheduled, Unscheduled{TeamID: team.ID, Team: team.Name, Reason: "No free pitch left."})
		}
	}
	if result.Proposals == nil {
		result.Proposals = []ProposedEvent{}
	}
	if result.Unscheduled == nil {
		result.Unscheduled = []Unscheduled{}
	}
	return result, nil
}

func materialize(db *model.Database, team model.Team, req SuggestRequest, p pattern, duration int, start, end time.Time, busy []model.Event) ([]model.Event, []string, int, bool) {
	var events []model.Event
	var warns []string
	missed := 0
	weeksSeen := map[string]bool{}
	for day := start; day.Before(end); day = day.AddDate(0, 0, 1) {
		if int(day.Weekday()) != p.dow {
			continue
		}
		weekKey := day.AddDate(0, 0, -((int(day.Weekday()) - db.Settings.WeekStartsOn + 7) % 7)).Format("2006-01-02")
		weeksSeen[weekKey] = true
		slotStart := time.Date(day.Year(), day.Month(), day.Day(), p.startMin/60, p.startMin%60, 0, 0, time.Local)
		slotEnd := slotStart.Add(time.Duration(duration) * time.Minute)
		ev := draft(db, team, req.EventTypeID, p, slotStart, slotEnd)
		if blocked(db, ev, busy) {
			missed++
			continue
		}
		warn := ""
		if db.LatePriority(ev) {
			warn = lateMessage(db, ev)
		}
		events = append(events, ev)
		warns = append(warns, warn)
	}
	if len(events) == 0 {
		return nil, nil, 0, false
	}
	score := len(events)*100000 + timeScore(db, team, p)
	score -= missed * 1000
	_ = weeksSeen
	return events, warns, score, true
}

func draft(db *model.Database, team model.Team, typeID string, p pattern, start, end time.Time) model.Event {
	ev := model.Event{
		EventTypeID: typeID,
		TeamIDs:     []string{team.ID},
		VenueID:     p.venue.ID,
		PitchID:     p.pitch.ID,
		SurfaceID:   p.pitch.SurfaceID,
		Environment: p.pitch.Environment,
		FormatID:    p.pitch.FormatID,
		HomeAway:    "home",
		Start:       model.FormatWall(start),
		End:         model.FormatWall(end),
		Status:      "scheduled",
	}
	if typ, ok := db.FindEventType(typeID); ok {
		ev.Title = autoTitle(db, ev, typ)
	}
	return ev
}

func blocked(db *model.Database, ev model.Event, busy []model.Event) bool {
	start, err1 := model.ParseFlexible(ev.Start)
	end, err2 := model.ParseFlexible(ev.End)
	if err1 != nil || err2 != nil {
		return true
	}
	for _, date := range model.OccupiedDates(start, end) {
		for _, b := range db.Blackouts {
			if date < b.Start || date > b.End {
				continue
			}
			if b.VenueID == "" || b.VenueID == ev.VenueID {
				return true
			}
		}
	}
	for _, other := range busy {
		if other.Status == "cancelled" {
			continue
		}
		os, e1 := model.ParseFlexible(other.Start)
		oe, e2 := model.ParseFlexible(other.End)
		if e1 != nil || e2 != nil || !model.RangesOverlap(start, end, os, oe) {
			continue
		}
		if ev.PitchID != "" && ev.PitchID == other.PitchID {
			return true
		}
		if intersects(ev.TeamIDs, other.TeamIDs) {
			return true
		}
		if db.Settings.WarnCoachConflicts && coachesOverlap(db, ev, other) {
			return true
		}
	}
	return false
}

func timeScore(db *model.Database, team model.Team, p pattern) int {
	earlyEnd, _ := model.ClockMinutes(db.Settings.EarlyWindowEnd)
	score := 0
	if db.TeamIsPriority(team) {
		if p.startMin < earlyEnd {
			score = 20000 + (earlyEnd - p.startMin)
		} else {
			score = 1000 - (p.startMin - earlyEnd)
		}
	} else if db.Settings.OlderTeamsAvoidEarly {
		if p.startMin >= earlyEnd {
			score = 20000 - (p.startMin - earlyEnd)
		} else {
			score = 2000 + p.startMin
		}
	} else {
		score = 20000 - p.startMin
	}
	if want := preferredFormat(db, team); want != "" && p.pitch.FormatID == want {
		score += 500
	}
	if team.PreferredSurfaceID != "" && p.pitch.SurfaceID == team.PreferredSurfaceID {
		score += 120
	}
	if team.PreferredVenueID != "" && p.pitch.VenueID == team.PreferredVenueID {
		score += 180
	}
	return score
}

func preferredFormat(db *model.Database, team model.Team) string {
	if team.PreferredFormatID != "" {
		return team.PreferredFormatID
	}
	if ag, ok := db.FindAgeGroup(team.AgeGroupID); ok {
		return ag.PreferredFormatID
	}
	return ""
}

func practiceMinutes(db *model.Database, team model.Team, typeID string) int {
	typ, _ := db.FindEventType(typeID)
	if team.PracticeMinutes > 0 && (typ.Category == "practice" || typ.Category == "") {
		return team.PracticeMinutes
	}
	if ag, ok := db.FindAgeGroup(team.AgeGroupID); ok {
		if typ.Category == "game" && ag.DefaultGameMinutes > 0 {
			return ag.DefaultGameMinutes
		}
		if ag.DefaultPracticeMinutes > 0 && typ.Category != "game" && typ.Category != "camp" {
			return ag.DefaultPracticeMinutes
		}
	}
	if typ.DefaultMinutes > 0 {
		return typ.DefaultMinutes
	}
	return 60
}

func enumerate(db *model.Database, team model.Team, duration int, days []int, venues, pitches []string) []pattern {
	var out []pattern
	step := db.Settings.SlotMinutes
	if step <= 0 {
		step = 30
	}
	for _, pitch := range db.Pitches {
		if !pitch.Active {
			continue
		}
		if len(pitches) > 0 && !hasID(pitches, pitch.ID) {
			continue
		}
		venue, ok := db.FindVenue(pitch.VenueID)
		if !ok || !venue.Active {
			continue
		}
		if len(venues) > 0 && !hasID(venues, venue.ID) {
			continue
		}
		openDays := pitch.Days
		if len(openDays) == 0 {
			openDays = venue.Days
		}
		open, closeAt := hours(db, venue, pitch)
		for _, dow := range days {
			if len(openDays) > 0 && !containsInt(openDays, dow) {
				continue
			}
			for min := open; min+duration <= closeAt; min += step {
				out = append(out, pattern{dow: dow, startMin: min, pitch: pitch, venue: venue})
			}
		}
	}
	_ = team
	return out
}

func hours(db *model.Database, venue model.Venue, pitch model.Pitch) (int, int) {
	open := pitch.OpenTime
	if open == "" {
		open = venue.OpenTime
	}
	closeAt := pitch.CloseTime
	if closeAt == "" {
		closeAt = venue.CloseTime
	}
	if open == "" {
		open = model.FormatClock(db.Settings.DayStartHour * 60)
	}
	if closeAt == "" {
		closeAt = model.FormatClock(db.Settings.DayEndHour * 60)
	}
	om, ok1 := model.ClockMinutes(open)
	cm, ok2 := model.ClockMinutes(closeAt)
	if !ok1 {
		om = db.Settings.DayStartHour * 60
	}
	if !ok2 {
		cm = db.Settings.DayEndHour * 60
	}
	dayStart := db.Settings.DayStartHour * 60
	dayEnd := db.Settings.DayEndHour * 60
	if om < dayStart {
		om = dayStart
	}
	if cm > dayEnd {
		cm = dayEnd
	}
	return om, cm
}

func uniqueDays(days []int) []int {
	seen := map[int]bool{}
	var out []int
	for _, d := range days {
		if d < 0 || d > 6 || seen[d] {
			continue
		}
		seen[d] = true
		out = append(out, d)
	}
	sort.Ints(out)
	return out
}

func containsInt(xs []int, v int) bool {
	for _, x := range xs {
		if x == v {
			return true
		}
	}
	return false
}

type Slot struct {
	Start   string `json:"start"`
	End     string `json:"end"`
	PitchID string `json:"pitchId"`
	VenueID string `json:"venueId"`
	Label   string `json:"label"`
	Score   int    `json:"score"`
	Late    bool   `json:"late"`
}

// Earlier finds open times for a draft, preferring the early window for young teams.
func Earlier(db *model.Database, ev model.Event, limit int) []Slot {
	if limit <= 0 || limit > 8 {
		limit = 4
	}
	start, err1 := model.ParseFlexible(ev.Start)
	end, err2 := model.ParseFlexible(ev.End)
	if err1 != nil || err2 != nil || !end.After(start) {
		return []Slot{}
	}
	duration := int(end.Sub(start).Minutes())
	if ev.AllDay {
		return []Slot{}
	}
	var team model.Team
	if len(ev.TeamIDs) > 0 {
		team, _ = db.FindTeam(ev.TeamIDs[0])
	}
	from := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.Local)
	until := from.AddDate(0, 0, 7)
	ignore := []string{ev.ID}
	var slots []Slot
	step := db.Settings.SlotMinutes
	if step <= 0 {
		step = 30
	}
	for _, pitch := range db.Pitches {
		if !pitch.Active {
			continue
		}
		venue, ok := db.FindVenue(pitch.VenueID)
		if !ok || !venue.Active {
			continue
		}
		open, closeAt := hours(db, venue, pitch)
		openDays := pitch.Days
		if len(openDays) == 0 {
			openDays = venue.Days
		}
		for day := from; day.Before(until); day = day.AddDate(0, 0, 1) {
			if len(openDays) > 0 && !containsInt(openDays, int(day.Weekday())) {
				continue
			}
			for min := open; min+duration <= closeAt; min += step {
				slotStart := time.Date(day.Year(), day.Month(), day.Day(), min/60, min%60, 0, 0, time.Local)
				slotEnd := slotStart.Add(time.Duration(duration) * time.Minute)
				if slotStart.Equal(start) && pitch.ID == ev.PitchID {
					continue
				}
				trial := ev
				trial.PitchID = pitch.ID
				trial.VenueID = venue.ID
				trial.SurfaceID = pitch.SurfaceID
				trial.Environment = pitch.Environment
				trial.FormatID = pitch.FormatID
				trial.HomeAway = "home"
				trial.Start = model.FormatWall(slotStart)
				trial.End = model.FormatWall(slotEnd)
				trial.Status = "scheduled"
				if blockedExcept(db, trial, ignore) {
					continue
				}
				p := pattern{dow: int(day.Weekday()), startMin: min, pitch: pitch, venue: venue}
				score := timeScore(db, team, p)
				if day.Equal(from) {
					score += 50
				}
				label := slotStart.Format("Mon 3:04 PM")
				if db.Settings.TimeFormat == "24" {
					label = slotStart.Format("Mon 15:04")
				}
				label += " · " + venue.Name + " " + pitch.Name
				slots = append(slots, Slot{
					Start: model.FormatWall(slotStart), End: model.FormatWall(slotEnd),
					PitchID: pitch.ID, VenueID: venue.ID, Label: label, Score: score,
					Late: db.LatePriority(trial),
				})
			}
		}
	}
	sort.Slice(slots, func(i, j int) bool {
		if slots[i].Score != slots[j].Score {
			return slots[i].Score > slots[j].Score
		}
		return slots[i].Start < slots[j].Start
	})
	if len(slots) > limit {
		slots = slots[:limit]
	}
	if slots == nil {
		slots = []Slot{}
	}
	return slots
}

func blockedExcept(db *model.Database, ev model.Event, ignore []string) bool {
	kept := make([]model.Event, 0, len(db.Events))
	for _, other := range db.Events {
		if hasID(ignore, other.ID) {
			continue
		}
		kept = append(kept, other)
	}
	return blocked(db, ev, kept)
}

type Proposal struct {
	Kind        string `json:"kind"`
	Summary     string `json:"summary"`
	EventID     string `json:"eventId"`
	Start       string `json:"start"`
	End         string `json:"end"`
	PitchID     string `json:"pitchId"`
	VenueID     string `json:"venueId"`
	SwapEventID string `json:"swapEventId,omitempty"`
	SwapStart   string `json:"swapStart,omitempty"`
	SwapEnd     string `json:"swapEnd,omitempty"`
	SwapPitchID string `json:"swapPitchId,omitempty"`
	SwapVenueID string `json:"swapVenueId,omitempty"`
}

// Fixes suggests earlier pitches for young teams booked after the early window.
func Fixes(db *model.Database, from, to time.Time) []Proposal {
	var out []Proposal
	earlyEnd, _ := model.ClockMinutes(db.Settings.EarlyWindowEnd)
	for _, ev := range db.Events {
		if ev.Status == "cancelled" || !db.LatePriority(ev) {
			continue
		}
		start, err1 := model.ParseFlexible(ev.Start)
		end, err2 := model.ParseFlexible(ev.End)
		if err1 != nil || err2 != nil || !model.RangesOverlap(start, end, from, to) {
			continue
		}
		if slot, ok := earliestOpen(db, ev, start, end.Sub(start), earlyEnd); ok {
			out = append(out, Proposal{
				Kind:    "move",
				Summary: fmt.Sprintf("Move %s to %s.", ev.Title, slotLabel(db, slot.start, slot.pitch)),
				EventID: ev.ID,
				Start:   model.FormatWall(slot.start),
				End:     model.FormatWall(slot.end),
				PitchID: slot.pitch.ID,
				VenueID: slot.pitch.VenueID,
			})
			continue
		}
		if swap, ok := findSwap(db, ev, start, end); ok {
			out = append(out, swap)
		}
	}
	if out == nil {
		out = []Proposal{}
	}
	return out
}

type openSlot struct {
	start, end time.Time
	pitch      model.Pitch
	score      int
}

func earliestOpen(db *model.Database, ev model.Event, day time.Time, dur time.Duration, earlyEnd int) (openSlot, bool) {
	var team model.Team
	if len(ev.TeamIDs) > 0 {
		team, _ = db.FindTeam(ev.TeamIDs[0])
	}
	step := db.Settings.SlotMinutes
	if step <= 0 {
		step = 30
	}
	minutes := int(dur.Minutes())
	var best openSlot
	found := false
	date := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, time.Local)
	for _, pitch := range db.Pitches {
		if !pitch.Active {
			continue
		}
		venue, ok := db.FindVenue(pitch.VenueID)
		if !ok || !venue.Active {
			continue
		}
		open, closeAt := hours(db, venue, pitch)
		openDays := pitch.Days
		if len(openDays) == 0 {
			openDays = venue.Days
		}
		if len(openDays) > 0 && !containsInt(openDays, int(date.Weekday())) {
			continue
		}
		for min := open; min+minutes <= closeAt && min < earlyEnd; min += step {
			slotStart := date.Add(time.Duration(min) * time.Minute)
			slotEnd := slotStart.Add(dur)
			if !slotStart.Before(day) && pitch.ID == ev.PitchID {
				// still allow a move earlier on the same pitch
			}
			trial := ev
			trial.PitchID = pitch.ID
			trial.VenueID = venue.ID
			trial.SurfaceID = pitch.SurfaceID
			trial.Environment = pitch.Environment
			trial.FormatID = pitch.FormatID
			trial.Start = model.FormatWall(slotStart)
			trial.End = model.FormatWall(slotEnd)
			if blockedExcept(db, trial, []string{ev.ID}) {
				continue
			}
			if db.LatePriority(trial) {
				continue
			}
			score := timeScore(db, team, pattern{startMin: min, pitch: pitch, venue: venue})
			if !found || score > best.score {
				best = openSlot{start: slotStart, end: slotEnd, pitch: pitch, score: score}
				found = true
			}
		}
	}
	return best, found
}

func findSwap(db *model.Database, ev model.Event, start, end time.Time) (Proposal, bool) {
	dur := end.Sub(start)
	for _, other := range db.Events {
		if other.ID == ev.ID || other.Status == "cancelled" || db.HasPriorityTeam(other) {
			continue
		}
		os, e1 := model.ParseFlexible(other.Start)
		oe, e2 := model.ParseFlexible(other.End)
		if e1 != nil || e2 != nil || os.Year() != start.Year() || os.YearDay() != start.YearDay() || oe.Sub(os) != dur {
			continue
		}
		// Prefer keeping each team on its own pitch and only exchanging the hour.
		if proposal, ok := swapProposal(db, ev, other, os, oe, start, end, false); ok {
			return proposal, true
		}
		if proposal, ok := swapProposal(db, ev, other, os, oe, start, end, true); ok {
			return proposal, true
		}
	}
	return Proposal{}, false
}

func swapProposal(db *model.Database, ev, other model.Event, os, oe, start, end time.Time, exchangePitch bool) (Proposal, bool) {
	youth := ev
	youth.Start = model.FormatWall(os)
	youth.End = model.FormatWall(oe)
	older := other
	older.Start = model.FormatWall(start)
	older.End = model.FormatWall(end)
	if exchangePitch {
		youth.PitchID, older.PitchID = other.PitchID, ev.PitchID
		youth.VenueID, older.VenueID = other.VenueID, ev.VenueID
		if p, ok := db.FindPitch(youth.PitchID); ok {
			youth.SurfaceID, youth.Environment, youth.FormatID = p.SurfaceID, p.Environment, p.FormatID
		}
		if p, ok := db.FindPitch(older.PitchID); ok {
			older.SurfaceID, older.Environment, older.FormatID = p.SurfaceID, p.Environment, p.FormatID
		}
	}
	if db.LatePriority(youth) || !fitsHours(db, youth) || !fitsHours(db, older) {
		return Proposal{}, false
	}
	if blockedExcept(db, youth, []string{ev.ID, other.ID}) || blockedExcept(db, older, []string{ev.ID, other.ID}) {
		return Proposal{}, false
	}
	summary := fmt.Sprintf("Swap times with %s so %s starts at %s.", other.Title, ev.Title, model.FormatWhen(os, db.Settings.TimeFormat))
	if exchangePitch {
		summary = fmt.Sprintf("Swap times and pitches with %s so %s starts at %s.", other.Title, ev.Title, model.FormatWhen(os, db.Settings.TimeFormat))
	}
	return Proposal{
		Kind:        "swap",
		Summary:     summary,
		EventID:     ev.ID,
		Start:       youth.Start,
		End:         youth.End,
		PitchID:     youth.PitchID,
		VenueID:     youth.VenueID,
		SwapEventID: other.ID,
		SwapStart:   older.Start,
		SwapEnd:     older.End,
		SwapPitchID: older.PitchID,
		SwapVenueID: older.VenueID,
	}, true
}

func fitsHours(db *model.Database, ev model.Event) bool {
	pitch, ok := db.FindPitch(ev.PitchID)
	if !ok || !pitch.Active {
		return ev.HomeAway == "away"
	}
	venue, ok := db.FindVenue(pitch.VenueID)
	if !ok || !venue.Active {
		return false
	}
	start, err1 := model.ParseFlexible(ev.Start)
	end, err2 := model.ParseFlexible(ev.End)
	if err1 != nil || err2 != nil || end.Day() != start.Day() {
		return false
	}
	open, closeAt := hours(db, venue, pitch)
	min := start.Hour()*60 + start.Minute()
	endMin := end.Hour()*60 + end.Minute()
	return min >= open && endMin <= closeAt
}

func slotLabel(db *model.Database, start time.Time, pitch model.Pitch) string {
	venue, _ := db.FindVenue(pitch.VenueID)
	when := start.Format("Mon 3:04 PM")
	if db.Settings.TimeFormat == "24" {
		when = start.Format("Mon 15:04")
	}
	return when + " at " + venue.Name + " " + pitch.Name
}
