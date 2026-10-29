// Package schedule places younger teams in the early window and checks conflicts.
package schedule

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"pitchboard/internal/model"
)

type Issue struct {
	Kind    string `json:"kind"`
	Level   string `json:"level"`
	Message string `json:"message"`
}

type CheckResult struct {
	OK           bool    `json:"ok"`
	Issues       []Issue `json:"issues"`
	LatePriority bool    `json:"latePriority"`
}

func (r CheckResult) Errors() []Issue {
	out := []Issue{}
	for _, issue := range r.Issues {
		if issue.Level == "error" {
			out = append(out, issue)
		}
	}
	return out
}

type WriteOptions struct {
	AcknowledgeLate bool
	IgnoreIDs       []string
}

// Normalize fills blanks and reports problems that always block a save.
func Normalize(db *model.Database, ev model.Event) (model.Event, []Issue) {
	var issues []Issue
	ev.Title = strings.TrimSpace(ev.Title)
	ev.Notes = strings.TrimSpace(ev.Notes)
	ev.Opponent = strings.TrimSpace(ev.Opponent)
	ev.AwayLocation = strings.TrimSpace(ev.AwayLocation)
	ev.TeamIDs = compact(ev.TeamIDs)
	if ev.Status == "" {
		ev.Status = "scheduled"
	}
	switch ev.Status {
	case "scheduled", "tentative", "cancelled":
	default:
		issues = append(issues, Issue{Kind: "validation", Level: "error", Message: "Status must be scheduled, tentative, or cancelled."})
	}
	if ev.HomeAway == "" {
		ev.HomeAway = "home"
	}
	switch ev.HomeAway {
	case "home", "away":
	default:
		issues = append(issues, Issue{Kind: "validation", Level: "error", Message: "Choose home or away."})
	}
	typ, tok := db.FindEventType(ev.EventTypeID)
	if !tok {
		issues = append(issues, Issue{Kind: "validation", Level: "error", Message: "Choose a practice, camp, game, or other type."})
	}
	if len(ev.TeamIDs) == 0 && (typ.Category == "practice" || typ.Category == "game" || typ.Category == "camp" || typ.Category == "") {
		issues = append(issues, Issue{Kind: "validation", Level: "error", Message: "Choose at least one team."})
	}
	for _, id := range ev.TeamIDs {
		if _, ok := db.FindTeam(id); !ok {
			issues = append(issues, Issue{Kind: "validation", Level: "error", Message: "One of the teams no longer exists."})
			break
		}
	}
	start, errS := model.ParseFlexible(ev.Start)
	end, errE := model.ParseFlexible(ev.End)
	if errS != nil || errE != nil {
		issues = append(issues, Issue{Kind: "validation", Level: "error", Message: "Enter a start and end."})
	} else if !end.After(start) {
		issues = append(issues, Issue{Kind: "validation", Level: "error", Message: "The end must be after the start."})
	} else if !ev.AllDay && end.Sub(start) < 15*time.Minute {
		issues = append(issues, Issue{Kind: "validation", Level: "error", Message: "Book at least 15 minutes."})
	} else {
		ev.Start = model.FormatWall(start)
		ev.End = model.FormatWall(end)
	}
	if ev.HomeAway == "away" {
		if ev.Opponent == "" && ev.AwayLocation == "" {
			issues = append(issues, Issue{Kind: "validation", Level: "error", Message: "Add the opponent or the away location."})
		}
		if ev.PitchID != "" {
			if pitch, ok := db.FindPitch(ev.PitchID); ok {
				if ev.VenueID == "" {
					ev.VenueID = pitch.VenueID
				}
				snapshot(&ev, pitch)
			}
		}
	} else {
		pitch, ok := db.FindPitch(ev.PitchID)
		if !ok {
			issues = append(issues, Issue{Kind: "validation", Level: "error", Message: "Choose a pitch."})
		} else {
			if ev.VenueID == "" {
				ev.VenueID = pitch.VenueID
			}
			if ev.VenueID != pitch.VenueID {
				issues = append(issues, Issue{Kind: "validation", Level: "error", Message: "That pitch is not at the chosen location."})
			}
			snapshot(&ev, pitch)
		}
	}
	if ev.Title == "" {
		ev.Title = autoTitle(db, ev, typ)
	}
	return ev, issues
}

func snapshot(ev *model.Event, pitch model.Pitch) {
	if ev.SurfaceID == "" {
		ev.SurfaceID = pitch.SurfaceID
	}
	if ev.Environment == "" {
		ev.Environment = pitch.Environment
	}
	if ev.FormatID == "" {
		ev.FormatID = pitch.FormatID
	}
	if ev.VenueID == "" {
		ev.VenueID = pitch.VenueID
	}
}

func autoTitle(db *model.Database, ev model.Event, typ model.EventType) string {
	name := ""
	if len(ev.TeamIDs) == 1 {
		if t, ok := db.FindTeam(ev.TeamIDs[0]); ok {
			if t.ShortName != "" {
				name = t.ShortName
			} else {
				name = t.Name
			}
		}
	}
	if typ.Category == "game" && ev.Opponent != "" && name != "" {
		return name + " vs " + ev.Opponent
	}
	if name != "" && typ.Name != "" {
		return name + " " + typ.Name
	}
	if typ.Name != "" {
		return typ.Name
	}
	return "Booking"
}

// Check reports hard conflicts and early-window warnings for one event.
func Check(db *model.Database, ev model.Event, opt WriteOptions) CheckResult {
	ev, issues := Normalize(db, ev)
	result := CheckResult{Issues: issues, LatePriority: db.LatePriority(ev)}
	if len(result.Errors()) > 0 {
		result.OK = false
		return result
	}
	start, _ := model.ParseFlexible(ev.Start)
	end, _ := model.ParseFlexible(ev.End)
	if ev.Status != "cancelled" {
		for _, other := range db.Events {
			if other.Status == "cancelled" {
				continue
			}
			if ev.ID != "" && other.ID == ev.ID {
				continue
			}
			if other.ID != "" && hasID(opt.IgnoreIDs, other.ID) {
				continue
			}
			os, err1 := model.ParseFlexible(other.Start)
			oe, err2 := model.ParseFlexible(other.End)
			if err1 != nil || err2 != nil || !model.RangesOverlap(start, end, os, oe) {
				continue
			}
			if db.Settings.BlockPitchConflicts && ev.PitchID != "" && ev.PitchID == other.PitchID {
				result.Issues = append(result.Issues, Issue{
					Kind:    "pitch",
					Level:   "error",
					Message: fmt.Sprintf("%s is already booked by %s, %s–%s.", pitchLabel(db, other), other.Title, model.FormatWhen(os, db.Settings.TimeFormat), model.FormatWhen(oe, db.Settings.TimeFormat)),
				})
			}
			if db.Settings.BlockTeamConflicts && intersects(ev.TeamIDs, other.TeamIDs) {
				result.Issues = append(result.Issues, Issue{
					Kind:    "team",
					Level:   "error",
					Message: fmt.Sprintf("%s is already booked then (%s).", teamList(db, intersect(ev.TeamIDs, other.TeamIDs)), other.Title),
				})
			}
			if db.Settings.WarnCoachConflicts && coachesOverlap(db, ev, other) {
				result.Issues = append(result.Issues, Issue{
					Kind:    "coach",
					Level:   "warning",
					Message: fmt.Sprintf("The coach is also with %s at that time.", other.Title),
				})
			}
		}
		for _, date := range model.OccupiedDates(start, end) {
			for _, b := range db.Blackouts {
				if date < b.Start || date > b.End {
					continue
				}
				if b.VenueID != "" && b.VenueID != ev.VenueID {
					continue
				}
				if ev.HomeAway == "away" && ev.VenueID == "" && b.VenueID != "" {
					continue
				}
				level := "warning"
				if db.Settings.BlockBlackouts {
					level = "error"
				}
				place := "The club"
				if b.VenueID != "" {
					if v, ok := db.FindVenue(b.VenueID); ok {
						place = v.Name
					}
				}
				reason := b.Reason
				if reason == "" {
					reason = "Closed"
				}
				result.Issues = append(result.Issues, Issue{
					Kind:    "blackout",
					Level:   level,
					Message: fmt.Sprintf("%s is closed on %s (%s).", place, date, reason),
				})
			}
		}
	}
	if result.LatePriority {
		level := "warning"
		if db.Settings.PriorityMode == "strict" && !opt.AcknowledgeLate {
			level = "error"
		}
		result.Issues = append(result.Issues, Issue{
			Kind:    "priority",
			Level:   level,
			Message: lateMessage(db, ev),
		})
	}
	result.OK = len(result.Errors()) == 0
	if result.Issues == nil {
		result.Issues = []Issue{}
	}
	return result
}

func lateMessage(db *model.Database, ev model.Event) string {
	who := "This team"
	if len(ev.TeamIDs) == 1 {
		if t, ok := db.FindTeam(ev.TeamIDs[0]); ok {
			who = t.Name
		}
	} else if db.HasPriorityTeam(ev) {
		var names []string
		for _, id := range ev.TeamIDs {
			t, ok := db.FindTeam(id)
			if ok && db.TeamIsPriority(t) {
				names = append(names, t.Name)
			}
		}
		if len(names) > 0 {
			who = strings.Join(names, " and ")
		}
	}
	end := prettyClock(db.Settings.EarlyWindowEnd, db.Settings.TimeFormat)
	return fmt.Sprintf("%s is under %d and this starts after %s. Earlier hours are kept for them first.", who, db.Settings.PriorityCutoffAge, end)
}

func prettyClock(clock, format string) string {
	mins, ok := model.ClockMinutes(clock)
	if !ok {
		return clock
	}
	t := time.Date(2000, 1, 1, mins/60, mins%60, 0, 0, time.Local)
	return model.FormatWhen(t, format)
}

func pitchLabel(db *model.Database, ev model.Event) string {
	pitch, ok := db.FindPitch(ev.PitchID)
	if !ok {
		return "That pitch"
	}
	if venue, vok := db.FindVenue(pitch.VenueID); vok {
		return venue.Name + " " + pitch.Name
	}
	return pitch.Name
}

func teamList(db *model.Database, ids []string) string {
	var names []string
	for _, id := range ids {
		if t, ok := db.FindTeam(id); ok {
			names = append(names, t.Name)
		}
	}
	if len(names) == 0 {
		return "A team"
	}
	return strings.Join(names, ", ")
}

func coachesOverlap(db *model.Database, a, b model.Event) bool {
	left := coachSet(db, a)
	for _, id := range b.TeamIDs {
		t, ok := db.FindTeam(id)
		if !ok {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(t.Coach))
		if key != "" && left[key] {
			return true
		}
	}
	return false
}

func coachSet(db *model.Database, ev model.Event) map[string]bool {
	out := map[string]bool{}
	for _, id := range ev.TeamIDs {
		t, ok := db.FindTeam(id)
		if !ok {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(t.Coach))
		if key != "" {
			out[key] = true
		}
	}
	return out
}

func hasID(ids []string, id string) bool {
	for _, x := range ids {
		if x == id {
			return true
		}
	}
	return false
}

func intersects(a, b []string) bool {
	return len(intersect(a, b)) > 0
}

func intersect(a, b []string) []string {
	set := map[string]bool{}
	for _, x := range a {
		set[x] = true
	}
	var out []string
	seen := map[string]bool{}
	for _, x := range b {
		if set[x] && !seen[x] {
			seen[x] = true
			out = append(out, x)
		}
	}
	return out
}

func compact(ids []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out
}

// Expand repeats a booking daily or weekly. The end date is inclusive.
func Expand(ev model.Event, repeat, until string, count int) ([]model.Event, error) {
	repeat = strings.TrimSpace(repeat)
	if repeat == "" || repeat == "none" {
		return []model.Event{ev}, nil
	}
	if repeat != "weekly" && repeat != "daily" {
		return nil, fmt.Errorf("repeat must be none, daily, or weekly")
	}
	if count <= 0 {
		count = 8
	}
	if count > 60 {
		count = 60
	}
	start, err := model.ParseFlexible(ev.Start)
	if err != nil {
		return nil, err
	}
	end, err := model.ParseFlexible(ev.End)
	if err != nil {
		return nil, err
	}
	var untilT time.Time
	if strings.TrimSpace(until) != "" {
		untilT, err = model.ParseFlexible(until)
		if err != nil {
			return nil, fmt.Errorf("repeat-until date is not valid")
		}
		untilT = time.Date(untilT.Year(), untilT.Month(), untilT.Day(), 23, 59, 59, 0, untilT.Location())
	}
	step := 7
	if repeat == "daily" {
		step = 1
	}
	out := []model.Event{}
	for i := 0; i < count; i++ {
		if !untilT.IsZero() && start.After(untilT) {
			break
		}
		next := ev
		next.ID = ""
		next.Start = model.FormatWall(start)
		next.End = model.FormatWall(end)
		out = append(out, next)
		start = start.AddDate(0, 0, step)
		end = end.AddDate(0, 0, step)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("that repeat does not land on any dates")
	}
	return out, nil
}

// CheckAll validates a batch against saved bookings and against itself.
func CheckAll(db *model.Database, events []model.Event, opt WriteOptions) CheckResult {
	merged := CheckResult{OK: true, Issues: []Issue{}}
	scratch := *db
	scratch.Events = append([]model.Event{}, db.Events...)
	for i, ev := range events {
		res := Check(&scratch, ev, opt)
		if res.LatePriority {
			merged.LatePriority = true
		}
		for _, issue := range res.Issues {
			issue.Message = prefixDate(ev, issue.Message)
			merged.Issues = append(merged.Issues, issue)
		}
		if len(res.Errors()) == 0 {
			ev.ID = fmt.Sprintf("pending%d", i)
			scratch.Events = append(scratch.Events, ev)
		}
	}
	merged.OK = len(merged.Errors()) == 0
	return merged
}

func prefixDate(ev model.Event, message string) string {
	start, err := model.ParseFlexible(ev.Start)
	if err != nil {
		return message
	}
	return start.Format("Jan 2") + ": " + message
}

func sortTeams(db *model.Database, ids []string) []model.Team {
	var teams []model.Team
	seen := map[string]bool{}
	for _, id := range ids {
		if seen[id] {
			continue
		}
		t, ok := db.FindTeam(id)
		if !ok || !t.Active {
			continue
		}
		seen[id] = true
		teams = append(teams, t)
	}
	sort.Slice(teams, func(i, j int) bool {
		pi := db.TeamIsPriority(teams[i])
		pj := db.TeamIsPriority(teams[j])
		if pi != pj {
			return pi
		}
		ai, aj := maxAge(db, teams[i]), maxAge(db, teams[j])
		if ai != aj {
			return ai < aj
		}
		return teams[i].Name < teams[j].Name
	})
	return teams
}

func maxAge(db *model.Database, t model.Team) int {
	if ag, ok := db.FindAgeGroup(t.AgeGroupID); ok {
		return ag.MaxAge
	}
	return 99
}
