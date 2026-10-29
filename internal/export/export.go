// Package export writes the schedule to CSV, Excel, PDF, and calendar files.
package export

import (
	"strings"
	"time"

	"pitchboard/internal/model"
)

type Query struct {
	Start            time.Time
	End              time.Time
	TeamIDs          []string
	VenueIDs         []string
	TypeIDs          []string
	Environments     []string
	SurfaceIDs       []string
	IncludeCancelled bool
	Text             string
}

func Select(db *model.Database, q Query) []model.EventView {
	var views []model.EventView
	needle := strings.ToLower(strings.TrimSpace(q.Text))
	for _, ev := range db.Events {
		if ev.Status == "cancelled" && !q.IncludeCancelled {
			continue
		}
		start, err1 := model.ParseFlexible(ev.Start)
		end, err2 := model.ParseFlexible(ev.End)
		if err1 != nil || err2 != nil {
			continue
		}
		if !q.Start.IsZero() || !q.End.IsZero() {
			rs, re := q.Start, q.End
			if rs.IsZero() {
				rs = time.Date(1, 1, 1, 0, 0, 0, 0, time.Local)
			}
			if re.IsZero() {
				re = time.Date(9999, 1, 1, 0, 0, 0, 0, time.Local)
			}
			if !model.RangesOverlap(start, end, rs, re) {
				continue
			}
		}
		view := db.Present(ev)
		if len(q.TeamIDs) > 0 && !overlap(ev.TeamIDs, q.TeamIDs) {
			continue
		}
		if len(q.VenueIDs) > 0 && !contains(q.VenueIDs, ev.VenueID) {
			continue
		}
		if len(q.TypeIDs) > 0 && !contains(q.TypeIDs, ev.EventTypeID) {
			continue
		}
		if len(q.Environments) > 0 && !contains(q.Environments, ev.Environment) {
			continue
		}
		if len(q.SurfaceIDs) > 0 && !contains(q.SurfaceIDs, ev.SurfaceID) {
			continue
		}
		if needle != "" && !matchesText(view, needle) {
			continue
		}
		views = append(views, view)
	}
	sortViews(views)
	if views == nil {
		views = []model.EventView{}
	}
	return views
}

func sortViews(views []model.EventView) {
	for i := 1; i < len(views); i++ {
		j := i
		for j > 0 && (views[j].Start < views[j-1].Start || (views[j].Start == views[j-1].Start && views[j].Title < views[j-1].Title)) {
			views[j], views[j-1] = views[j-1], views[j]
			j--
		}
	}
}

func matchesText(v model.EventView, needle string) bool {
	blob := strings.ToLower(strings.Join([]string{
		v.Title, v.Notes, v.Opponent, v.AwayLocation, v.LocationLabel, v.PlaceDetail,
		v.VenueName, v.PitchName, v.SurfaceName, v.TypeName, strings.Join(v.TeamNames, " "),
		strings.Join(v.CoachNames, " "),
	}, " "))
	return strings.Contains(blob, needle)
}

func overlap(a, b []string) bool {
	set := map[string]bool{}
	for _, x := range b {
		set[x] = true
	}
	for _, x := range a {
		if set[x] {
			return true
		}
	}
	return false
}

func contains(xs []string, v string) bool {
	for _, x := range xs {
		if x == v {
			return true
		}
	}
	return false
}

func priorityLabel(v model.EventView) string {
	if v.LatePriority {
		return "Under cutoff, late start"
	}
	if v.PriorityTeams {
		return "Early priority"
	}
	return ""
}

func safeName(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "Pitchboard"
	}
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '<', '>', ':', '"', '/', '\\', '|', '?', '*':
			b.WriteByte('-')
		default:
			if r < 32 {
				continue
			}
			b.WriteRune(r)
		}
	}
	out := strings.TrimSpace(b.String())
	if out == "" {
		return "Pitchboard"
	}
	return out
}

func FileName(club, ext string, start, end time.Time) string {
	name := safeName(club) + "-schedule"
	if !start.IsZero() {
		name += "-" + start.Format("2006-01-02")
	}
	if !end.IsZero() {
		last := end.Add(-time.Second)
		if last.Before(start) {
			last = end
		}
		name += "-to-" + last.Format("2006-01-02")
	}
	return name + "." + ext
}
