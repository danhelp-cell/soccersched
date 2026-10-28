package store

import (
	"fmt"
	"time"

	"pitchboard/internal/model"
)

func timeNow() time.Time { return time.Now() }

var allDays = []int{0, 1, 2, 3, 4, 5, 6}

// EnsureCatalogs adds the built-in age groups, surfaces, pitch types, and
// event types when they are missing. Existing rows are left alone.
func EnsureCatalogs(db *model.Database) {
	defaults := []model.Surface{
		{ID: "surf_indoor_turf", Name: "Indoor turf", Environment: "indoor"},
		{ID: "surf_hardcourt", Name: "Indoor hardcourt", Environment: "indoor"},
		{ID: "surf_gym", Name: "Gym", Environment: "indoor"},
		{ID: "surf_grass", Name: "Grass", Environment: "outdoor"},
		{ID: "surf_outdoor_turf", Name: "Outdoor turf", Environment: "outdoor"},
	}
	for _, s := range defaults {
		if _, ok := db.FindSurface(s.ID); !ok {
			db.Surfaces = append(db.Surfaces, s)
		}
	}
	formats := []model.Format{
		{ID: "fmt_3", Name: "3v3", Sort: 1},
		{ID: "fmt_5", Name: "5v5", Sort: 2},
		{ID: "fmt_7", Name: "7v7", Sort: 3},
		{ID: "fmt_9", Name: "9v9", Sort: 4},
		{ID: "fmt_11", Name: "11v11", Sort: 5},
		{ID: "fmt_gym", Name: "Gym floor", Sort: 6},
	}
	for _, f := range formats {
		if _, ok := db.FindFormat(f.ID); !ok {
			db.Formats = append(db.Formats, f)
		}
	}
	groups := []model.AgeGroup{
		{ID: "ag_u6", Name: "U6", MinAge: 4, MaxAge: 6, PreferredFormatID: "fmt_3", DefaultPracticeMinutes: 60, DefaultGameMinutes: 45, SortOrder: 1},
		{ID: "ag_u8", Name: "U8", MinAge: 7, MaxAge: 8, PreferredFormatID: "fmt_5", DefaultPracticeMinutes: 60, DefaultGameMinutes: 50, SortOrder: 2},
		{ID: "ag_u10", Name: "U10", MinAge: 9, MaxAge: 10, PreferredFormatID: "fmt_7", DefaultPracticeMinutes: 75, DefaultGameMinutes: 60, SortOrder: 3},
		{ID: "ag_u12", Name: "U12", MinAge: 11, MaxAge: 12, PreferredFormatID: "fmt_9", DefaultPracticeMinutes: 75, DefaultGameMinutes: 70, SortOrder: 4},
		{ID: "ag_u14", Name: "U14", MinAge: 13, MaxAge: 14, PreferredFormatID: "fmt_11", DefaultPracticeMinutes: 90, DefaultGameMinutes: 80, SortOrder: 5},
		{ID: "ag_u16", Name: "U16", MinAge: 15, MaxAge: 16, PreferredFormatID: "fmt_11", DefaultPracticeMinutes: 90, DefaultGameMinutes: 90, SortOrder: 6},
		{ID: "ag_u18", Name: "U18", MinAge: 17, MaxAge: 18, PreferredFormatID: "fmt_11", DefaultPracticeMinutes: 90, DefaultGameMinutes: 90, SortOrder: 7},
		{ID: "ag_sr", Name: "Senior", MinAge: 19, MaxAge: 99, PreferredFormatID: "fmt_11", DefaultPracticeMinutes: 90, DefaultGameMinutes: 90, SortOrder: 8},
	}
	for _, g := range groups {
		if _, ok := db.FindAgeGroup(g.ID); !ok {
			db.AgeGroups = append(db.AgeGroups, g)
		}
	}
	types := []model.EventType{
		{ID: "type_practice", Name: "Practice", Category: "practice", Color: "#2f6a4a", DefaultMinutes: 60},
		{ID: "type_camp", Name: "Camp", Category: "camp", Color: "#3d7a8c", DefaultMinutes: 180},
		{ID: "type_game", Name: "Game", Category: "game", Color: "#c4553a", DefaultMinutes: 75},
	}
	for _, t := range types {
		if _, ok := db.FindEventType(t.ID); !ok {
			db.EventTypes = append(db.EventTypes, t)
		}
	}
}

// ApplySample replaces teams, places, and events with a working club week.
// Club name and priority rules already chosen by the user stay in place.
func ApplySample(db *model.Database, now time.Time) {
	EnsureCatalogs(db)
	if db.Settings.ClubName == "" {
		db.Settings.ClubName = "Northfield Soccer Club"
	}
	db.Teams = nil
	db.Venues = nil
	db.Pitches = nil
	db.Events = nil
	db.Blackouts = nil

	db.Venues = []model.Venue{
		{ID: "ven_dome", Name: "Northfield Dome", Address: "100 Dome Road", Notes: "Indoor soccer shoes only. Lobby opens 15 minutes early.", OpenTime: "08:00", CloseTime: "22:00", Days: allDays, Active: true},
		{ID: "ven_lake", Name: "Lakeside Recreation Centre", Address: "45 Lake Avenue", Notes: "Gym and hardcourt are inside the main doors. Clean shoes on the gym floor.", OpenTime: "08:00", CloseTime: "21:00", Days: allDays, Active: true},
		{ID: "ven_park", Name: "Memorial Park", Address: "800 Park Lane", Notes: "Parking lot off Park Lane. Corner flags are in the shed by Grass 1.", OpenTime: "08:00", CloseTime: "21:00", Days: allDays, Active: true},
		{ID: "ven_plex", Name: "Riverside Sportsplex", Address: "12 River Road", Notes: "Outdoor turf. No metal cleats.", OpenTime: "08:00", CloseTime: "22:00", Days: allDays, Active: true},
	}
	db.Pitches = []model.Pitch{
		{ID: "pit_dome_a", VenueID: "ven_dome", Name: "Turf A", Environment: "indoor", SurfaceID: "surf_indoor_turf", FormatID: "fmt_7", Days: allDays, Active: true},
		{ID: "pit_dome_b", VenueID: "ven_dome", Name: "Turf B", Environment: "indoor", SurfaceID: "surf_indoor_turf", FormatID: "fmt_5", Days: allDays, Active: true},
		{ID: "pit_court", VenueID: "ven_lake", Name: "Court 1", Environment: "indoor", SurfaceID: "surf_hardcourt", FormatID: "fmt_5", Days: allDays, Active: true},
		{ID: "pit_gym", VenueID: "ven_lake", Name: "Gymnasium", Environment: "indoor", SurfaceID: "surf_gym", FormatID: "fmt_3", Days: allDays, Notes: "Small-sided only.", Active: true},
		{ID: "pit_g1", VenueID: "ven_park", Name: "Grass 1", Environment: "outdoor", SurfaceID: "surf_grass", FormatID: "fmt_11", Days: allDays, Active: true},
		{ID: "pit_g2", VenueID: "ven_park", Name: "Grass 2", Environment: "outdoor", SurfaceID: "surf_grass", FormatID: "fmt_9", Days: allDays, Active: true},
		{ID: "pit_g3", VenueID: "ven_park", Name: "Grass 3", Environment: "outdoor", SurfaceID: "surf_grass", FormatID: "fmt_5", Days: allDays, Active: true},
		{ID: "pit_t1", VenueID: "ven_plex", Name: "Turf 1", Environment: "outdoor", SurfaceID: "surf_outdoor_turf", FormatID: "fmt_11", Days: allDays, Active: true},
		{ID: "pit_t2", VenueID: "ven_plex", Name: "Turf 2", Environment: "outdoor", SurfaceID: "surf_outdoor_turf", FormatID: "fmt_7", Days: allDays, Active: true},
	}
	db.Teams = []model.Team{
		team("team_u6", "U6 Sparks", "U6 Sparks", "ag_u6", "#c4892a", "Maya Chen", "306-555-0101", "maya.chen@northfield.example", "ven_lake", "surf_gym", "Parents can watch from the lobby."),
		team("team_u8b", "U8 Blue", "U8 Blue", "ag_u8", "#2f6f9f", "Luis Ortega", "306-555-0102", "luis.ortega@northfield.example", "ven_dome", "surf_indoor_turf", ""),
		team("team_u8r", "U8 Red", "U8 Red", "ag_u8", "#c4553a", "Priya Shah", "306-555-0108", "priya.shah@northfield.example", "ven_dome", "surf_indoor_turf", "One practice this week is booked after the early window so the planner can offer an earlier pitch."),
		team("team_u10g", "U10 Girls", "U10 Girls", "ag_u10", "#7a4e8a", "Elena Rossi", "306-555-0104", "elena.rossi@northfield.example", "ven_dome", "surf_indoor_turf", "Shares a coach with U10 Boys."),
		team("team_u10b", "U10 Boys", "U10 Boys", "ag_u10", "#2f6a4a", "Elena Rossi", "306-555-0104", "elena.rossi@northfield.example", "ven_dome", "surf_indoor_turf", "Shares a coach with U10 Girls."),
		team("team_u12", "U12 United", "U12 United", "ag_u12", "#3d7a8c", "Chris Adelayo", "306-555-0112", "chris.adelayo@northfield.example", "ven_plex", "surf_outdoor_turf", ""),
		team("team_u14", "U14 City", "U14 City", "ag_u14", "#8a4b2f", "Nadia Petrov", "306-555-0114", "nadia.petrov@northfield.example", "ven_park", "surf_grass", ""),
		team("team_u16", "U16 Elite", "U16 Elite", "ag_u16", "#1e3d30", "Marcus Hale", "306-555-0116", "marcus.hale@northfield.example", "ven_plex", "surf_outdoor_turf", ""),
		team("team_u18", "U18 Academy", "U18 Academy", "ag_u18", "#5c4d7a", "Imani Brooks", "306-555-0118", "imani.brooks@northfield.example", "ven_park", "surf_grass", ""),
		team("team_sr", "Senior Men", "Senior Men", "ag_sr", "#3e4c44", "Owen Blake", "306-555-0120", "owen.blake@northfield.example", "ven_plex", "surf_outdoor_turf", ""),
	}

	monday := mondayOf(now)
	at := func(day int, h, m int) string {
		t := monday.AddDate(0, 0, day)
		return model.FormatWall(time.Date(t.Year(), t.Month(), t.Day(), h, m, 0, 0, t.Location()))
	}
	type spec struct {
		typ, title, pitch, start, end, opponent, away, home, notes, status, series string
		teams                                                                      []string
	}
	rows := []spec{
		{typ: "type_practice", title: "U6 Sparks Practice", teams: []string{"team_u6"}, pitch: "pit_gym", start: at(0, 16, 0), end: at(0, 17, 0), notes: "Parents stay in the lobby."},
		{typ: "type_practice", title: "U8 Blue Practice", teams: []string{"team_u8b"}, pitch: "pit_dome_b", start: at(0, 16, 0), end: at(0, 17, 0)},
		{typ: "type_practice", title: "U10 Boys Practice", teams: []string{"team_u10b"}, pitch: "pit_dome_a", start: at(0, 16, 30), end: at(0, 17, 45)},
		{typ: "type_practice", title: "U10 Girls Practice", teams: []string{"team_u10g"}, pitch: "pit_dome_b", start: at(0, 17, 0), end: at(0, 18, 15), notes: "Coach also has U10 Boys until 5:45."},
		{typ: "type_practice", title: "U14 City Practice", teams: []string{"team_u14"}, pitch: "pit_g1", start: at(0, 18, 30), end: at(0, 20, 0)},
		{typ: "type_practice", title: "U16 Elite Practice", teams: []string{"team_u16"}, pitch: "pit_t1", start: at(0, 18, 30), end: at(0, 20, 0)},
		{typ: "type_practice", title: "U8 Red Practice", teams: []string{"team_u8r"}, pitch: "pit_court", start: at(0, 20, 0), end: at(0, 21, 0), notes: "Booked late on purpose. Plan a week can move this into the early window."},

		{typ: "type_practice", title: "U8 Red Practice", teams: []string{"team_u8r"}, pitch: "pit_dome_b", start: at(1, 16, 15), end: at(1, 17, 15)},
		{typ: "type_practice", title: "U12 United Practice", teams: []string{"team_u12"}, pitch: "pit_t2", start: at(1, 17, 30), end: at(1, 18, 45)},
		{typ: "type_practice", title: "U18 Academy Practice", teams: []string{"team_u18"}, pitch: "pit_g1", start: at(1, 18, 0), end: at(1, 19, 30)},

		{typ: "type_practice", title: "U6 Sparks Practice", teams: []string{"team_u6"}, pitch: "pit_gym", start: at(2, 16, 0), end: at(2, 17, 0)},
		{typ: "type_practice", title: "U10 Girls Practice", teams: []string{"team_u10g"}, pitch: "pit_dome_a", start: at(2, 16, 30), end: at(2, 17, 45)},
		{typ: "type_practice", title: "U14 City Practice", teams: []string{"team_u14"}, pitch: "pit_t1", start: at(2, 18, 0), end: at(2, 19, 30)},
		{typ: "type_practice", title: "U12 United Practice", teams: []string{"team_u12"}, pitch: "pit_g2", start: at(2, 18, 30), end: at(2, 19, 45)},

		{typ: "type_practice", title: "U8 Blue Practice", teams: []string{"team_u8b"}, pitch: "pit_g3", start: at(3, 16, 0), end: at(3, 17, 0), notes: "Outdoor grass, weather permitting."},
		{typ: "type_practice", title: "U10 Boys Practice", teams: []string{"team_u10b"}, pitch: "pit_t2", start: at(3, 16, 30), end: at(3, 17, 45)},
		{typ: "type_practice", title: "U16 Elite Practice", teams: []string{"team_u16"}, pitch: "pit_g1", start: at(3, 19, 0), end: at(3, 20, 30)},
		{typ: "type_practice", title: "Senior Men Practice", teams: []string{"team_sr"}, pitch: "pit_t1", start: at(3, 20, 0), end: at(3, 21, 30)},

		{typ: "type_practice", title: "U6 Sparks Practice", teams: []string{"team_u6"}, pitch: "pit_gym", start: at(4, 16, 0), end: at(4, 17, 0)},
		{typ: "type_practice", title: "U8 Blue Practice", teams: []string{"team_u8b"}, pitch: "pit_dome_b", start: at(4, 16, 30), end: at(4, 17, 30)},
		{typ: "type_practice", title: "U14 City Practice", teams: []string{"team_u14"}, pitch: "pit_g1", start: at(4, 18, 30), end: at(4, 20, 0)},
		{typ: "type_practice", title: "U18 Academy Practice", teams: []string{"team_u18"}, pitch: "pit_t1", start: at(4, 19, 0), end: at(4, 20, 30)},

		{typ: "type_game", title: "U10 Girls vs Westend FC", teams: []string{"team_u10g"}, pitch: "pit_t2", start: at(5, 9, 0), end: at(5, 10, 0), opponent: "Westend FC", home: "home"},
		{typ: "type_game", title: "U8 Blue vs Harbour SC", teams: []string{"team_u8b"}, pitch: "pit_g3", start: at(5, 10, 30), end: at(5, 11, 20), opponent: "Harbour SC", home: "home"},
		{typ: "type_game", title: "U12 United vs Hillcrest", teams: []string{"team_u12"}, pitch: "pit_g2", start: at(5, 11, 0), end: at(5, 12, 10), opponent: "Hillcrest", home: "home"},
		{typ: "type_game", title: "U14 City vs Capital United", teams: []string{"team_u14"}, start: at(5, 13, 0), end: at(5, 14, 20), opponent: "Capital United", away: "Capital Stadium", home: "away"},
		{typ: "type_game", title: "U16 Elite vs Lakeshore", teams: []string{"team_u16"}, pitch: "pit_g1", start: at(5, 15, 30), end: at(5, 17, 0), opponent: "Lakeshore", home: "home"},

		{typ: "type_camp", title: "Weekend Skills Camp", teams: []string{"team_u8r", "team_u10b"}, pitch: "pit_dome_a", start: at(6, 9, 0), end: at(6, 12, 0), notes: "Bring indoor shoes, water, and a light snack.", series: "series_skills_camp"},
		{typ: "type_camp", title: "Weekend Skills Camp", teams: []string{"team_u8r", "team_u10b"}, pitch: "pit_dome_a", start: at(13, 9, 0), end: at(13, 12, 0), notes: "Second Sunday of the camp.", series: "series_skills_camp"},
		{typ: "type_game", title: "U6 Sparks Jamboree", teams: []string{"team_u6"}, pitch: "pit_gym", start: at(6, 10, 0), end: at(6, 11, 0), opponent: "Mini Kickers", home: "home", notes: "Short games, every player plays."},
		{typ: "type_practice", title: "U18 Academy Practice", teams: []string{"team_u18"}, pitch: "pit_g1", start: at(6, 18, 0), end: at(6, 19, 30), status: "tentative", notes: "Hold until the coach confirms the field."},
	}
	stamp := time.Now().Format(time.RFC3339)
	for _, row := range rows {
		ev := model.Event{
			EventTypeID:  row.typ,
			Title:        row.title,
			TeamIDs:      row.teams,
			PitchID:      row.pitch,
			Start:        row.start,
			End:          row.end,
			Opponent:     row.opponent,
			AwayLocation: row.away,
			HomeAway:     row.home,
			Notes:        row.notes,
			Status:       row.status,
			SeriesID:     row.series,
			CreatedAt:    stamp,
			UpdatedAt:    stamp,
		}
		if ev.HomeAway == "" {
			ev.HomeAway = "home"
		}
		if ev.Status == "" {
			ev.Status = "scheduled"
		}
		if pitch, ok := db.FindPitch(ev.PitchID); ok {
			ev.VenueID = pitch.VenueID
			ev.SurfaceID = pitch.SurfaceID
			ev.Environment = pitch.Environment
			ev.FormatID = pitch.FormatID
		}
		ev.ID = db.NewID("evt")
		db.Events = append(db.Events, ev)
	}
	blackoutDay := monday.AddDate(0, 0, 9).Format("2006-01-02")
	db.Blackouts = []model.Blackout{{
		ID:      db.NewID("blk"),
		Start:   blackoutDay,
		End:     blackoutDay,
		VenueID: "ven_lake",
		Reason:  "School photos — gym and court closed",
	}}
}

func team(id, name, short, age, color, coach, phone, email, venue, surface, notes string) model.Team {
	return model.Team{
		ID: id, Name: name, ShortName: short, AgeGroupID: age, Color: color,
		Coach: coach, CoachPhone: phone, CoachEmail: email,
		Priority: "auto", PreferredVenueID: venue, PreferredSurfaceID: surface,
		Notes: notes, Active: true,
	}
}

func mondayOf(t time.Time) time.Time {
	t = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	delta := (int(t.Weekday()) + 6) % 7
	return t.AddDate(0, 0, -delta)
}

// ClearClub removes teams, places, and bookings and keeps settings and catalogs.
func ClearClub(db *model.Database) {
	db.Teams = []model.Team{}
	db.Venues = []model.Venue{}
	db.Pitches = []model.Pitch{}
	db.Events = []model.Event{}
	db.Blackouts = []model.Blackout{}
}

func SeasonLabel(now time.Time) string {
	return fmt.Sprintf("%s", model.DefaultSettings(now).SeasonName)
}
