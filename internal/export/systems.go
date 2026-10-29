package export

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"strings"
	"time"

	"pitchboard/internal/model"
)

// System describes a file another club website can import.
type System struct {
	ID            string
	Name          string
	FileNote      string
	Categories    []string
	OneRowPerTeam bool
}

func Systems() []System {
	return []System{
		{ID: "ramp", Name: "RAMP InterActive", FileNote: "Practice file. Team_ID and Location_ID are the numbers from RAMP. Event_Type is always Practice.", Categories: []string{"practice"}, OneRowPerTeam: true},
		{ID: "teamlinkt", Name: "TeamLinkt", FileNote: "Names must match the teams and locations already in TeamLinkt. Type is Practice, Game, or Camp.", OneRowPerTeam: true},
		{ID: "teamsnap", Name: "TeamSnap", FileNote: "Team schedule template. Practices omit the opponent. Dates are MM/DD/YYYY.", OneRowPerTeam: true},
		{ID: "gamechanger", Name: "GameChanger", FileNote: "Organization schedule. Duration is minutes. Practices leave the away column empty.", OneRowPerTeam: true},
	}
}

func FindSystem(id string) (System, bool) {
	for _, system := range Systems() {
		if system.ID == id {
			return system, true
		}
	}
	return System{}, false
}

// SystemCSV writes one of the built-in online-system files.
// skipped counts rows that could not be included because a required id was missing.
func SystemCSV(db *model.Database, rows []model.EventView, systemID string, categories []string) ([]byte, int, error) {
	system, ok := FindSystem(systemID)
	if !ok {
		return nil, 0, fmt.Errorf("unknown export")
	}
	allowed := categories
	if system.ID == "ramp" {
		allowed = []string{"practice"}
	}
	var buf bytes.Buffer
	buf.Write([]byte{0xEF, 0xBB, 0xBF})
	w := csv.NewWriter(&buf)
	header, err := systemHeader(system.ID)
	if err != nil {
		return nil, 0, err
	}
	if err := w.Write(header); err != nil {
		return nil, 0, err
	}
	skipped := 0
	for _, row := range rows {
		if len(allowed) > 0 && !contains(allowed, row.Category) {
			continue
		}
		lines, miss := systemRows(db, row, system.ID)
		skipped += miss
		for _, line := range lines {
			if err := w.Write(line); err != nil {
				return nil, 0, err
			}
		}
	}
	w.Flush()
	return buf.Bytes(), skipped, w.Error()
}

func systemHeader(id string) ([]string, error) {
	switch id {
	case "ramp":
		return []string{"Team_ID", "Event_Type", "Start_Date", "Start_Time", "End_Time", "Location_ID", "Notes"}, nil
	case "teamlinkt":
		return []string{"Date", "Time", "Type", "Home Division", "Home Team", "Away Team", "Location", "Duration", "Notes"}, nil
	case "teamsnap":
		return []string{
			"Date", "Time", "Duration (HH:MM)", "Arrival Time (Minutes)", "Name", "Opponent Name",
			"Opponent Contact Name", "Opponent Contact Phone Number", "Opponent Contact E-mail Address",
			"Location Name", "Location Address", "Location Details", "Location URL", "Home or Away",
			"Uniform", "Extra Label", "Notes",
		}, nil
	case "gamechanger":
		return []string{"date", "time", "home", "away", "location", "duration"}, nil
	default:
		return nil, fmt.Errorf("unknown export")
	}
}

func systemRows(db *model.Database, row model.EventView, systemID string) ([][]string, int) {
	start, err1 := model.ParseFlexible(row.Start)
	end, err2 := model.ParseFlexible(row.End)
	if err1 != nil || err2 != nil || len(row.TeamIDs) == 0 {
		return nil, 1
	}
	teams := row.TeamIDs
	var lines [][]string
	missed := 0
	for i, teamID := range teams {
		team, ok := db.FindTeam(teamID)
		if !ok {
			missed++
			continue
		}
		switch systemID {
		case "ramp":
			line, ok := rampRow(db, row, team, start, end)
			if !ok {
				missed++
				continue
			}
			lines = append(lines, line)
		case "teamlinkt":
			lines = append(lines, teamLinktRow(db, row, team, start, end, i))
		case "teamsnap":
			lines = append(lines, teamSnapRow(db, row, team, start, end))
		case "gamechanger":
			lines = append(lines, gameChangerRow(db, row, team, start, end))
		}
	}
	return lines, missed
}

func rampRow(db *model.Database, row model.EventView, team model.Team, start, end time.Time) ([]string, bool) {
	teamID := team.Systems.RampID()
	locationID := rampLocation(db, row)
	if teamID == "" || locationID == "" || !digitsOnly(teamID) || !digitsOnly(locationID) {
		return nil, false
	}
	return []string{
		teamID,
		"Practice",
		start.Format("2006-01-02"),
		clock12(start),
		clock12(end),
		locationID,
		row.Notes,
	}, true
}

func rampLocation(db *model.Database, row model.EventView) string {
	if pitch, ok := db.FindPitch(row.PitchID); ok && pitch.Systems.RampID() != "" {
		return pitch.Systems.RampID()
	}
	if venue, ok := db.FindVenue(row.VenueID); ok {
		return venue.Systems.RampID()
	}
	return ""
}

func teamLinktRow(db *model.Database, row model.EventView, team model.Team, start, end time.Time, index int) []string {
	division := ""
	if index < len(row.AgeLabels) {
		division = row.AgeLabels[index]
	}
	if ag, ok := db.FindAgeGroup(team.AgeGroupID); ok && ag.Program != "" {
		if division != "" {
			division = ag.Program + " " + division
		} else {
			division = ag.Program
		}
	}
	away := ""
	if row.Category == "game" {
		away = row.Opponent
	}
	name := team.Name
	if external := strings.TrimSpace(team.Systems.TeamLinkt); external != "" {
		name = external
	}
	location := locationName(db, row)
	if venue, ok := db.FindVenue(row.VenueID); ok && strings.TrimSpace(venue.Systems.TeamLinkt) != "" {
		location = strings.TrimSpace(venue.Systems.TeamLinkt)
	}
	return []string{
		start.Format("2006-01-02"),
		start.Format("3:04 PM"),
		typeLabel(row),
		division,
		name,
		away,
		location,
		fmt.Sprintf("%d", durationMinutes(start, end)),
		row.Notes,
	}
}

func teamSnapRow(db *model.Database, row model.EventView, team model.Team, start, end time.Time) []string {
	name := row.Title
	opponent := ""
	home := ""
	if row.Category == "game" {
		name = ""
		opponent = row.Opponent
		if row.HomeAway == "away" {
			home = "a"
		} else {
			home = "h"
		}
	}
	if name == "" && row.Category != "game" {
		name = team.Name + " " + typeLabel(row)
	}
	venue, _ := db.FindVenue(row.VenueID)
	return []string{
		start.Format("01/02/2006"),
		clock12(start),
		durationHHMM(start, end),
		"",
		name,
		opponent,
		"",
		"",
		"",
		locationName(db, row),
		venue.Address,
		strings.TrimSpace(row.PlaceDetail),
		"",
		home,
		"",
		team.Name,
		row.Notes,
	}
}

func gameChangerRow(db *model.Database, row model.EventView, team model.Team, start, end time.Time) []string {
	away := ""
	if row.Category == "game" {
		away = row.Opponent
	}
	home := team.Name
	if external := strings.TrimSpace(team.Systems.GameChanger); external != "" {
		home = external
	}
	return []string{
		start.Format("01/02/2006"),
		start.Format("3:04 PM"),
		home,
		away,
		locationName(db, row),
		fmt.Sprintf("%d", durationMinutes(start, end)),
	}
}

func locationName(db *model.Database, row model.EventView) string {
	if row.HomeAway == "away" && strings.TrimSpace(row.AwayLocation) != "" {
		return strings.TrimSpace(row.AwayLocation)
	}
	venue, _ := db.FindVenue(row.VenueID)
	pitch, _ := db.FindPitch(row.PitchID)
	if venue.Name != "" && pitch.Name != "" {
		return venue.Name + " - " + pitch.Name
	}
	if row.LocationLabel != "" {
		return row.LocationLabel
	}
	return strings.TrimSpace(venue.Name + " " + pitch.Name)
}

func typeLabel(row model.EventView) string {
	switch row.Category {
	case "practice":
		return "Practice"
	case "game":
		return "Game"
	case "camp":
		return "Camp"
	default:
		if row.TypeName != "" {
			return row.TypeName
		}
		return "Event"
	}
}

func clock12(t time.Time) string {
	return t.Format("03:04 PM")
}

func durationMinutes(start, end time.Time) int {
	mins := int(end.Sub(start).Minutes())
	if mins < 0 {
		return 0
	}
	return mins
}

func durationHHMM(start, end time.Time) string {
	mins := durationMinutes(start, end)
	return fmt.Sprintf("%02d:%02d", mins/60, mins%60)
}

func digitsOnly(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
