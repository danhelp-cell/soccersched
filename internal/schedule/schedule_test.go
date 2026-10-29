package schedule

import (
	"strings"
	"testing"
	"time"

	"pitchboard/internal/model"
	"pitchboard/internal/store"
)

func TestUnder11UsesMaxAge(t *testing.T) {
	db := club()
	u8, _ := db.FindTeam("u8")
	u12, _ := db.FindTeam("u12")
	if !db.TeamIsPriority(u8) {
		t.Fatal("U8 should have early priority when the cutoff is 11")
	}
	if db.TeamIsPriority(u12) {
		t.Fatal("U12 should not have early priority when the cutoff is 11")
	}
	always := u12
	always.Priority = "always"
	if !db.TeamIsPriority(always) {
		t.Fatal("always should override age")
	}
}

func TestBackToBackIsFree(t *testing.T) {
	db := club()
	first := booking(&db, "u8", "p1", "2026-10-26T16:00:00", "2026-10-26T17:00:00")
	db.Events = append(db.Events, first)
	second := booking(&db, "u12", "p1", "2026-10-26T17:00:00", "2026-10-26T18:00:00")
	res := Check(&db, second, WriteOptions{})
	if !res.OK {
		t.Fatalf("back-to-back should be allowed, got %+v", res.Issues)
	}
	overlap := booking(&db, "u16", "p1", "2026-10-26T16:30:00", "2026-10-26T17:30:00")
	res = Check(&db, overlap, WriteOptions{})
	if res.OK {
		t.Fatal("overlapping pitch should be blocked")
	}
}

func TestStrictLateNeedsAcknowledgement(t *testing.T) {
	db := club()
	db.Settings.PriorityMode = "strict"
	late := booking(&db, "u8", "p1", "2026-10-26T20:00:00", "2026-10-26T21:00:00")
	res := Check(&db, late, WriteOptions{})
	if res.OK || !res.LatePriority {
		t.Fatalf("strict mode should block a late youth practice: %+v", res)
	}
	res = Check(&db, late, WriteOptions{AcknowledgeLate: true})
	if !res.OK || !res.LatePriority {
		t.Fatalf("acknowledgement should allow the save and keep the flag: %+v", res)
	}
	db.Settings.PriorityMode = "warn"
	res = Check(&db, late, WriteOptions{})
	if !res.OK || !res.LatePriority {
		t.Fatalf("warn mode should save and flag the late start: %+v", res)
	}
}

func TestAcademyUsesItsOwnHours(t *testing.T) {
	db := club()
	for i := range db.AgeGroups {
		if db.AgeGroups[i].ID != "ag_academy" {
			continue
		}
		db.AgeGroups[i].WindowStart = "16:30"
		db.AgeGroups[i].WindowEnd = "20:30"
		db.AgeGroups[i].Priority = "never"
	}
	db.Teams = append(db.Teams, model.Team{
		ID: "academy", Name: "Academy Green", AgeGroupID: "ag_academy", Priority: "auto", Active: true,
	})
	res, err := Suggest(&db, SuggestRequest{
		TeamIDs:         []string{"academy"},
		EventTypeID:     "type_practice",
		RangeStart:      "2026-10-26",
		RangeEnd:        "2026-11-01",
		DaysOfWeek:      []int{1},
		SessionsPerWeek: 1,
		DurationMinutes: 90,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Proposals) != 1 {
		t.Fatalf("expected one academy practice, got %+v %+v", res.Proposals, res.Unscheduled)
	}
	if res.Proposals[0].Event.Start != "2026-10-26T16:30:00" {
		t.Fatalf("academy should start at its own window, got %s", res.Proposals[0].Event.Start)
	}
}

func TestAgeGroupPriorityOverride(t *testing.T) {
	db := club()
	for i := range db.AgeGroups {
		if db.AgeGroups[i].ID == "ag_u16" {
			db.AgeGroups[i].Priority = "always"
		}
	}
	u16, _ := db.FindTeam("u16")
	if !db.TeamIsPriority(u16) {
		t.Fatal("an age group can force early hours even when the players are older")
	}
}

func TestYoungerTeamsGetEarlierHours(t *testing.T) {
	db := club()
	res, err := Suggest(&db, SuggestRequest{
		TeamIDs:         []string{"u8", "u16"},
		EventTypeID:     "type_practice",
		RangeStart:      "2026-10-26",
		RangeEnd:        "2026-11-01",
		DaysOfWeek:      []int{1},
		SessionsPerWeek: 1,
		DurationMinutes: 60,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Proposals) != 2 {
		t.Fatalf("expected two practices, got %+v unscheduled %+v", res.Proposals, res.Unscheduled)
	}
	byTeam := map[string]string{}
	for _, p := range res.Proposals {
		byTeam[p.Event.TeamIDs[0]] = p.Event.Start
	}
	if byTeam["u8"] != "2026-10-26T08:00:00" {
		t.Fatalf("U8 should take the earliest hour, got %s", byTeam["u8"])
	}
	if byTeam["u16"] != "2026-10-26T17:30:00" {
		t.Fatalf("U16 should start at the end of the early window, got %s", byTeam["u16"])
	}
}

func TestSampleWeekIsPlayable(t *testing.T) {
	when := time.Date(2026, 10, 27, 9, 0, 0, 0, time.Local)
	db := model.Database{Settings: model.DefaultSettings(when)}
	store.ApplySample(&db, when)
	late := 0
	for i := range db.Events {
		a := db.Events[i]
		if a.Status == "cancelled" {
			continue
		}
		if db.LatePriority(a) {
			late++
		}
		as, _ := model.ParseFlexible(a.Start)
		ae, _ := model.ParseFlexible(a.End)
		for j := i + 1; j < len(db.Events); j++ {
			b := db.Events[j]
			if b.Status == "cancelled" || a.PitchID == "" || a.PitchID != b.PitchID {
				continue
			}
			bs, _ := model.ParseFlexible(b.Start)
			be, _ := model.ParseFlexible(b.End)
			if model.RangesOverlap(as, ae, bs, be) {
				t.Fatalf("%s overlaps %s on %s", a.Title, b.Title, a.PitchID)
			}
		}
	}
	if late == 0 {
		t.Fatal("sample week should include a youth team booked after the early window")
	}
}

func TestFixesMoveLateTeamEarlier(t *testing.T) {
	db := club()
	late := booking(&db, "u8", "p1", "2026-10-26T20:00:00", "2026-10-26T21:00:00")
	late.ID = "late"
	db.Events = append(db.Events, late)
	from := time.Date(2026, 10, 26, 0, 0, 0, 0, time.Local)
	props := Fixes(&db, from, from.AddDate(0, 0, 1))
	if len(props) == 0 || props[0].Kind != "move" {
		t.Fatalf("expected a move onto an earlier pitch, got %+v", props)
	}
	if !strings.HasPrefix(props[0].Start, "2026-10-26T") || props[0].Start >= "2026-10-26T17:30:00" {
		t.Fatalf("move should land inside the early window, got %s", props[0].Start)
	}
}

func club() model.Database {
	when := time.Date(2026, 10, 26, 9, 0, 0, 0, time.Local)
	db := model.Database{Settings: model.DefaultSettings(when)}
	store.EnsureCatalogs(&db)
	db.Settings.EarlyWindowEnd = "17:30"
	db.Venues = []model.Venue{{
		ID: "v1", Name: "Dome", OpenTime: "08:00", CloseTime: "22:00", Days: []int{0, 1, 2, 3, 4, 5, 6}, Active: true,
	}}
	db.Pitches = []model.Pitch{{
		ID: "p1", VenueID: "v1", Name: "Turf", Environment: "indoor", SurfaceID: "surf_indoor_turf", FormatID: "fmt_5",
		Days: []int{0, 1, 2, 3, 4, 5, 6}, Active: true,
	}}
	db.Teams = []model.Team{
		{ID: "u8", Name: "U8", ShortName: "U8", AgeGroupID: "ag_u8", Priority: "auto", Active: true},
		{ID: "u12", Name: "U12", ShortName: "U12", AgeGroupID: "ag_u12", Priority: "auto", Active: true},
		{ID: "u16", Name: "U16", ShortName: "U16", AgeGroupID: "ag_u16", Priority: "auto", Active: true},
	}
	db.Normalize()
	return db
}

func booking(db *model.Database, team, pitch, start, end string) model.Event {
	p, _ := db.FindPitch(pitch)
	return model.Event{
		EventTypeID: "type_practice",
		Title:       team + " Practice",
		TeamIDs:     []string{team},
		VenueID:     p.VenueID,
		PitchID:     pitch,
		SurfaceID:   p.SurfaceID,
		Environment: p.Environment,
		FormatID:    p.FormatID,
		HomeAway:    "home",
		Start:       start,
		End:         end,
		Status:      "scheduled",
	}
}
