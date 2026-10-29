package export

import (
	"strings"
	"testing"
	"time"

	"pitchboard/internal/model"
	"pitchboard/internal/store"
)

func TestRampPracticeFile(t *testing.T) {
	when := time.Date(2026, 10, 26, 9, 0, 0, 0, time.Local)
	db := model.Database{Settings: model.DefaultSettings(when)}
	store.ApplySample(&db, when)
	rows := Select(&db, Query{})
	body, skipped, err := SystemCSV(&db, rows, "ramp", nil)
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	if !strings.HasPrefix(text, "\uFEFFTeam_ID,Event_Type,Start_Date,Start_Time,End_Time,Location_ID,Notes") && !strings.Contains(text, "Team_ID,Event_Type,Start_Date,Start_Time,End_Time,Location_ID,Notes") {
		t.Fatalf("ramp header missing: %s", firstLine(text))
	}
	if !strings.Contains(text, "1004,Practice,") || !strings.Contains(text, ",08:00 PM,09:00 PM,502,") {
		t.Fatalf("late U8 practice should export with a zero-padded clock and the Henk Ruys location id:\n%s", text)
	}
	if strings.Contains(text, ",Game,") || strings.Contains(text, "Senior Men") {
		t.Fatal("ramp file is practices only and skips a team with no id")
	}
	if skipped == 0 {
		t.Fatal("senior practices should be skipped until a RAMP team id is entered")
	}
	for _, line := range strings.Split(text, "\n") {
		if strings.Contains(line, ",Practice,") && !strings.Contains(line, "Practice,") {
			t.Fatal(line)
		}
		if strings.Count(line, ",") < 6 || !strings.Contains(line, "Practice") && !strings.HasPrefix(strings.TrimPrefix(line, "\uFEFF"), "Team_ID") && line != "" && strings.Contains(line, "Practice") {
			continue
		}
	}
}

func TestOtherSystemHeaders(t *testing.T) {
	when := time.Date(2026, 10, 26, 12, 0, 0, 0, time.Local)
	db := model.Database{Settings: model.DefaultSettings(when)}
	store.ApplySample(&db, when)
	rows := Select(&db, Query{})
	checks := map[string]string{
		"teamsnap":    "Date,Time,Duration (HH:MM),Arrival Time (Minutes),Name,Opponent Name",
		"teamlinkt":   "Date,Time,Type,Home Division,Home Team,Away Team,Location,Duration,Notes",
		"gamechanger": "date,time,home,away,location,duration",
	}
	for id, header := range checks {
		body, _, err := SystemCSV(&db, rows, id, nil)
		if err != nil {
			t.Fatal(id, err)
		}
		if !strings.Contains(string(body), header) {
			t.Fatalf("%s header mismatch: %s", id, firstLine(string(body)))
		}
	}
	snap, _, err := SystemCSV(&db, rows, "teamsnap", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(snap), "Harbour SC") || !strings.Contains(string(snap), "h,") {
		t.Fatal("teamsnap games should carry the opponent and a home marker")
	}
}

func firstLine(text string) string {
	if i := strings.IndexByte(text, '\n'); i >= 0 {
		return text[:i]
	}
	return text
}
