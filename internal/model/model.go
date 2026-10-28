// Package model holds the club schedule stored on this computer.
package model

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

const Version = 1

// WallLayout is a club-local clock time with no timezone suffix.
const WallLayout = "2006-01-02T15:04:05"

type Database struct {
	Version        int             `json:"version"`
	NextID         int             `json:"nextId"`
	Settings       Settings        `json:"settings"`
	AgeGroups      []AgeGroup      `json:"ageGroups"`
	Teams          []Team          `json:"teams"`
	Venues         []Venue         `json:"venues"`
	Pitches        []Pitch         `json:"pitches"`
	Surfaces       []Surface       `json:"surfaces"`
	Formats        []Format        `json:"formats"`
	EventTypes     []EventType     `json:"eventTypes"`
	Events         []Event         `json:"events"`
	Blackouts      []Blackout      `json:"blackouts"`
	ExportProfiles []ExportProfile `json:"exportProfiles"`
}

type Settings struct {
	Initialized          bool     `json:"initialized"`
	ClubName             string   `json:"clubName"`
	SeasonName           string   `json:"seasonName"`
	SeasonStart          string   `json:"seasonStart"`
	SeasonEnd            string   `json:"seasonEnd"`
	TimezoneLabel        string   `json:"timezoneLabel"`
	WeekStartsOn         int      `json:"weekStartsOn"`
	DayStartHour         int      `json:"dayStartHour"`
	DayEndHour           int      `json:"dayEndHour"`
	SlotMinutes          int      `json:"slotMinutes"`
	DefaultView          string   `json:"defaultView"`
	ShowWeekends         bool     `json:"showWeekends"`
	TimeFormat           string   `json:"timeFormat"`
	PriorityCutoffAge    int      `json:"priorityCutoffAge"`
	EarlyWindowStart     string   `json:"earlyWindowStart"`
	EarlyWindowEnd       string   `json:"earlyWindowEnd"`
	PriorityMode         string   `json:"priorityMode"`
	PriorityDays         []int    `json:"priorityDays"`
	PriorityCategories   []string `json:"priorityCategories"`
	BlockPitchConflicts  bool     `json:"blockPitchConflicts"`
	BlockTeamConflicts   bool     `json:"blockTeamConflicts"`
	WarnCoachConflicts   bool     `json:"warnCoachConflicts"`
	BlockBlackouts       bool     `json:"blockBlackouts"`
	OlderTeamsAvoidEarly bool     `json:"olderTeamsAvoidEarly"`
	LegendEnabled        bool     `json:"legendEnabled"`
	Theme                Theme    `json:"theme"`
}

// Theme is the club's colours and type. Aurora Soccer is the built-in default.
type Theme struct {
	Header      string `json:"header"`
	HeaderInk   string `json:"headerInk"`
	Rail        string `json:"rail"`
	Accent      string `json:"accent"`
	AccentInk   string `json:"accentInk"`
	Paper       string `json:"paper"`
	Card        string `json:"card"`
	Ink         string `json:"ink"`
	Muted       string `json:"muted"`
	Line        string `json:"line"`
	DisplayFont string `json:"displayFont"`
	BodyFont    string `json:"bodyFont"`
}

// SystemIDs are the numbers and names other registration sites use for this record.
type SystemIDs struct {
	Ramp        string `json:"ramp"`
	TeamLinkt   string `json:"teamlinkt"`
	TeamSnap    string `json:"teamsnap"`
	GameChanger string `json:"gamechanger"`
}

// ExportProfile is a saved file layout for one online system.
type ExportProfile struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	System     string   `json:"system"`
	Categories []string `json:"categories"`
}

type AgeGroup struct {
	ID                     string `json:"id"`
	Name                   string `json:"name"`
	MinAge                 int    `json:"minAge"`
	MaxAge                 int    `json:"maxAge"`
	PreferredFormatID      string `json:"preferredFormatId"`
	DefaultPracticeMinutes int    `json:"defaultPracticeMinutes"`
	DefaultGameMinutes     int    `json:"defaultGameMinutes"`
	SortOrder              int    `json:"sortOrder"`
	Program                string `json:"program"`
	WindowStart            string `json:"windowStart"`
	WindowEnd              string `json:"windowEnd"`
	Priority               string `json:"priority"`
	SessionsPerWeek        int    `json:"sessionsPerWeek"`
}

type Team struct {
	ID                 string    `json:"id"`
	Name               string    `json:"name"`
	ShortName          string    `json:"shortName"`
	AgeGroupID         string    `json:"ageGroupId"`
	Color              string    `json:"color"`
	Coach              string    `json:"coach"`
	CoachPhone         string    `json:"coachPhone"`
	CoachEmail         string    `json:"coachEmail"`
	Priority           string    `json:"priority"`
	PreferredVenueID   string    `json:"preferredVenueId"`
	PreferredSurfaceID string    `json:"preferredSurfaceId"`
	PreferredFormatID  string    `json:"preferredFormatId"`
	PracticeMinutes    int       `json:"practiceMinutes"`
	Notes              string    `json:"notes"`
	Active             bool      `json:"active"`
	Systems            SystemIDs `json:"systems"`
}

type Venue struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Address   string    `json:"address"`
	Notes     string    `json:"notes"`
	OpenTime  string    `json:"openTime"`
	CloseTime string    `json:"closeTime"`
	Days      []int     `json:"days"`
	Active    bool      `json:"active"`
	Systems   SystemIDs `json:"systems"`
}

type Pitch struct {
	ID          string    `json:"id"`
	VenueID     string    `json:"venueId"`
	Name        string    `json:"name"`
	Environment string    `json:"environment"`
	SurfaceID   string    `json:"surfaceId"`
	FormatID    string    `json:"formatId"`
	OpenTime    string    `json:"openTime"`
	CloseTime   string    `json:"closeTime"`
	Days        []int     `json:"days"`
	Notes       string    `json:"notes"`
	Active      bool      `json:"active"`
	Systems     SystemIDs `json:"systems"`
}

type Surface struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Environment string `json:"environment"`
}

type Format struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Sort int    `json:"sort"`
}

type EventType struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Category       string `json:"category"`
	Color          string `json:"color"`
	DefaultMinutes int    `json:"defaultMinutes"`
}

type Event struct {
	ID           string   `json:"id"`
	SeriesID     string   `json:"seriesId"`
	EventTypeID  string   `json:"eventTypeId"`
	Title        string   `json:"title"`
	TeamIDs      []string `json:"teamIds"`
	VenueID      string   `json:"venueId"`
	PitchID      string   `json:"pitchId"`
	SurfaceID    string   `json:"surfaceId"`
	Environment  string   `json:"environment"`
	FormatID     string   `json:"formatId"`
	HomeAway     string   `json:"homeAway"`
	Opponent     string   `json:"opponent"`
	AwayLocation string   `json:"awayLocation"`
	Start        string   `json:"start"`
	End          string   `json:"end"`
	AllDay       bool     `json:"allDay"`
	Notes        string   `json:"notes"`
	Status       string   `json:"status"`
	CreatedAt    string   `json:"createdAt"`
	UpdatedAt    string   `json:"updatedAt"`
}

type Blackout struct {
	ID      string `json:"id"`
	Start   string `json:"start"`
	End     string `json:"end"`
	VenueID string `json:"venueId"`
	Reason  string `json:"reason"`
}

// EventView is an event plus the names a calendar needs to draw it.
type EventView struct {
	Event
	TypeName            string   `json:"typeName"`
	Category            string   `json:"category"`
	TypeColor           string   `json:"typeColor"`
	TeamNames           []string `json:"teamNames"`
	TeamShorts          []string `json:"teamShorts"`
	AgeLabels           []string `json:"ageLabels"`
	CoachNames          []string `json:"coachNames"`
	VenueName           string   `json:"venueName"`
	PitchName           string   `json:"pitchName"`
	SurfaceName         string   `json:"surfaceName"`
	EnvironmentResolved string   `json:"environmentResolved"`
	FormatName          string   `json:"formatName"`
	LocationLabel       string   `json:"locationLabel"`
	PlaceDetail         string   `json:"placeDetail"`
	PriorityTeams       bool     `json:"priorityTeams"`
	LatePriority        bool     `json:"latePriority"`
	TeamColors          []string `json:"teamColors"`
}

func (db *Database) Normalize() {
	if db.Version == 0 {
		db.Version = Version
	}
	db.Settings.Normalize()
	db.AgeGroups = nonNil(db.AgeGroups)
	db.Teams = nonNil(db.Teams)
	db.Venues = nonNil(db.Venues)
	db.Pitches = nonNil(db.Pitches)
	db.Surfaces = nonNil(db.Surfaces)
	db.Formats = nonNil(db.Formats)
	db.EventTypes = nonNil(db.EventTypes)
	db.Events = nonNil(db.Events)
	db.Blackouts = nonNil(db.Blackouts)
	db.ExportProfiles = nonNil(db.ExportProfiles)
	for i := range db.AgeGroups {
		normalizeAgeGroup(&db.AgeGroups[i])
	}
	for i := range db.ExportProfiles {
		normalizeExportProfile(&db.ExportProfiles[i])
	}
	for i := range db.Events {
		if db.Events[i].TeamIDs == nil {
			db.Events[i].TeamIDs = []string{}
		}
	}
	for i := range db.Venues {
		if db.Venues[i].Days == nil {
			db.Venues[i].Days = []int{}
		}
	}
	for i := range db.Pitches {
		if db.Pitches[i].Days == nil {
			db.Pitches[i].Days = []int{}
		}
	}
}

func nonNil[T any](in []T) []T {
	if in == nil {
		return []T{}
	}
	return in
}

func (s *Settings) Normalize() {
	s.ClubName = strings.TrimSpace(s.ClubName)
	s.SeasonName = strings.TrimSpace(s.SeasonName)
	s.TimezoneLabel = strings.TrimSpace(s.TimezoneLabel)
	if s.TimezoneLabel == "" {
		s.TimezoneLabel = "Local time"
	}
	if s.WeekStartsOn < 0 || s.WeekStartsOn > 6 {
		s.WeekStartsOn = 1
	}
	if s.DayStartHour < 0 || s.DayStartHour > 20 {
		s.DayStartHour = 8
	}
	if s.DayEndHour <= s.DayStartHour || s.DayEndHour > 24 {
		s.DayEndHour = 22
	}
	switch s.SlotMinutes {
	case 10, 15, 20, 30, 60:
	default:
		s.SlotMinutes = 30
	}
	switch s.DefaultView {
	case "timeGridDay", "timeGridWeek", "dayGridMonth", "timeGridWorkWeek":
	default:
		s.DefaultView = "timeGridWeek"
	}
	if s.TimeFormat != "24" {
		s.TimeFormat = "12"
	}
	if s.PriorityCutoffAge < 4 || s.PriorityCutoffAge > 30 {
		s.PriorityCutoffAge = 11
	}
	if _, ok := ClockMinutes(s.EarlyWindowStart); !ok {
		s.EarlyWindowStart = "08:00"
	}
	if _, ok := ClockMinutes(s.EarlyWindowEnd); !ok {
		s.EarlyWindowEnd = "17:30"
	}
	es, _ := ClockMinutes(s.EarlyWindowStart)
	ee, _ := ClockMinutes(s.EarlyWindowEnd)
	if ee <= es {
		s.EarlyWindowStart = "08:00"
		s.EarlyWindowEnd = "17:30"
	}
	if s.PriorityMode != "strict" {
		s.PriorityMode = "warn"
	}
	if s.PriorityDays == nil {
		s.PriorityDays = []int{0, 1, 2, 3, 4, 5, 6}
	}
	if s.PriorityCategories == nil {
		s.PriorityCategories = []string{"practice", "camp", "game", "other"}
	}
	s.PriorityDays = uniqueInts(s.PriorityDays)
	s.PriorityCategories = uniqueStrings(s.PriorityCategories)
	s.Theme = normalizeTheme(s.Theme)
}

func AuroraTheme() Theme {
	return Theme{
		Header:      "#171717",
		HeaderInk:   "#ffffff",
		Rail:        "#111111",
		Accent:      "#29683b",
		AccentInk:   "#ffffff",
		Paper:       "#f4f4f4",
		Card:        "#ffffff",
		Ink:         "#171717",
		Muted:       "#5e5e5e",
		Line:        "#e0e0e0",
		DisplayFont: "Oswald",
		BodyFont:    "Merriweather",
	}
}

func normalizeTheme(t Theme) Theme {
	base := AuroraTheme()
	t.Header = hexOr(t.Header, base.Header)
	t.HeaderInk = hexOr(t.HeaderInk, base.HeaderInk)
	t.Rail = hexOr(t.Rail, base.Rail)
	t.Accent = hexOr(t.Accent, base.Accent)
	t.AccentInk = hexOr(t.AccentInk, base.AccentInk)
	t.Paper = hexOr(t.Paper, base.Paper)
	t.Card = hexOr(t.Card, base.Card)
	t.Ink = hexOr(t.Ink, base.Ink)
	t.Muted = hexOr(t.Muted, base.Muted)
	t.Line = hexOr(t.Line, base.Line)
	t.DisplayFont = fontOr(t.DisplayFont, base.DisplayFont)
	t.BodyFont = fontOr(t.BodyFont, base.BodyFont)
	return t
}

func hexOr(value, fallback string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if len(value) != 7 || value[0] != '#' {
		return fallback
	}
	for _, c := range value[1:] {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return fallback
		}
	}
	return value
}

func fontOr(value, fallback string) string {
	switch strings.TrimSpace(value) {
	case "Oswald", "Merriweather", "Outfit", "Fraunces", "Segoe UI", "Georgia", "Arial":
		return strings.TrimSpace(value)
	default:
		return fallback
	}
}

func normalizeAgeGroup(g *AgeGroup) {
	g.Name = strings.TrimSpace(g.Name)
	g.Program = strings.TrimSpace(g.Program)
	if g.Priority != "always" && g.Priority != "never" {
		g.Priority = "auto"
	}
	if g.SessionsPerWeek < 0 {
		g.SessionsPerWeek = 0
	}
	if g.SessionsPerWeek > 7 {
		g.SessionsPerWeek = 7
	}
	ws, okS := ClockMinutes(g.WindowStart)
	we, okE := ClockMinutes(g.WindowEnd)
	if strings.TrimSpace(g.WindowStart) == "" || !okS {
		g.WindowStart = ""
	} else {
		g.WindowStart = FormatClock(ws)
	}
	if strings.TrimSpace(g.WindowEnd) == "" || !okE {
		g.WindowEnd = ""
	} else {
		g.WindowEnd = FormatClock(we)
	}
	if g.WindowStart != "" && g.WindowEnd != "" && we <= ws {
		g.WindowStart = ""
		g.WindowEnd = ""
	}
}

func normalizeExportProfile(p *ExportProfile) {
	p.Name = strings.TrimSpace(p.Name)
	switch p.System {
	case "ramp", "teamlinkt", "teamsnap", "gamechanger":
	default:
		p.System = "ramp"
	}
	p.Categories = uniqueStrings(p.Categories)
}

func (s SystemIDs) RampID() string { return strings.TrimSpace(s.Ramp) }

func (db *Database) NewID(prefix string) string {
	db.NextID++
	return fmt.Sprintf("%s%d", prefix, db.NextID)
}

func (db *Database) FindTeam(id string) (Team, bool)       { return findByID(db.Teams, id) }
func (db *Database) FindVenue(id string) (Venue, bool)     { return findByID(db.Venues, id) }
func (db *Database) FindPitch(id string) (Pitch, bool)     { return findByID(db.Pitches, id) }
func (db *Database) FindSurface(id string) (Surface, bool) { return findByID(db.Surfaces, id) }
func (db *Database) FindFormat(id string) (Format, bool)   { return findByID(db.Formats, id) }
func (db *Database) FindAgeGroup(id string) (AgeGroup, bool) {
	return findByID(db.AgeGroups, id)
}
func (db *Database) FindEventType(id string) (EventType, bool) {
	return findByID(db.EventTypes, id)
}
func (db *Database) FindEvent(id string) (Event, bool) { return findByID(db.Events, id) }

type identified interface {
	Ident() string
}

func (t Team) Ident() string          { return t.ID }
func (v Venue) Ident() string         { return v.ID }
func (p Pitch) Ident() string         { return p.ID }
func (s Surface) Ident() string       { return s.ID }
func (f Format) Ident() string        { return f.ID }
func (a AgeGroup) Ident() string      { return a.ID }
func (e EventType) Ident() string     { return e.ID }
func (e Event) Ident() string         { return e.ID }
func (b Blackout) Ident() string      { return b.ID }
func (p ExportProfile) Ident() string { return p.ID }

func findByID[T identified](items []T, id string) (T, bool) {
	for _, item := range items {
		if item.Ident() == id {
			return item, true
		}
	}
	var zero T
	return zero, false
}

// TeamIsPriority reports whether a team should be placed in the early window.
// Age groups qualify when their oldest age is under the cutoff (10 is under 11).
func (db *Database) TeamIsPriority(t Team) bool {
	switch t.Priority {
	case "always":
		return true
	case "never":
		return false
	}
	ag, ok := db.FindAgeGroup(t.AgeGroupID)
	if !ok {
		return false
	}
	switch ag.Priority {
	case "always":
		return true
	case "never":
		return false
	default:
		return ag.MaxAge < db.Settings.PriorityCutoffAge
	}
}

// TeamWindow is the hour band this team should be offered.
// A custom age-group window, such as Academy evenings, replaces the club early window.
func (db *Database) TeamWindow(t Team) (start, end int, custom bool) {
	start, _ = ClockMinutes(db.Settings.EarlyWindowStart)
	end, _ = ClockMinutes(db.Settings.EarlyWindowEnd)
	ag, ok := db.FindAgeGroup(t.AgeGroupID)
	if !ok {
		return start, end, false
	}
	ws, okS := ClockMinutes(ag.WindowStart)
	we, okE := ClockMinutes(ag.WindowEnd)
	if okS && okE && we > ws {
		return ws, we, true
	}
	return start, end, false
}

func (db *Database) HasPriorityTeam(ev Event) bool {
	for _, id := range ev.TeamIDs {
		t, ok := db.FindTeam(id)
		if ok && db.TeamIsPriority(t) {
			return true
		}
	}
	return false
}

func (db *Database) LatePriority(ev Event) bool {
	if ev.AllDay || !db.HasPriorityTeam(ev) {
		return false
	}
	if typ, ok := db.FindEventType(ev.EventTypeID); ok {
		cats := db.Settings.PriorityCategories
		if len(cats) > 0 && !containsStr(cats, typ.Category) {
			return false
		}
	}
	start, err := ParseFlexible(ev.Start)
	if err != nil {
		return false
	}
	days := db.Settings.PriorityDays
	if len(days) > 0 && !containsInt(days, int(start.Weekday())) {
		return false
	}
	mins := start.Hour()*60 + start.Minute()
	limit := 0
	found := false
	for _, id := range ev.TeamIDs {
		team, ok := db.FindTeam(id)
		if !ok || !db.TeamIsPriority(team) {
			continue
		}
		_, endMin, _ := db.TeamWindow(team)
		if !found || endMin < limit {
			limit = endMin
			found = true
		}
	}
	if !found {
		return false
	}
	return mins >= limit
}

func (db *Database) Present(ev Event) EventView {
	view := EventView{Event: ev}
	if view.TeamIDs == nil {
		view.TeamIDs = []string{}
	}
	if typ, ok := db.FindEventType(ev.EventTypeID); ok {
		view.TypeName = typ.Name
		view.Category = typ.Category
		view.TypeColor = typ.Color
	}
	seenCoach := map[string]bool{}
	for _, id := range ev.TeamIDs {
		t, ok := db.FindTeam(id)
		if !ok {
			continue
		}
		view.TeamNames = append(view.TeamNames, t.Name)
		short := t.ShortName
		if short == "" {
			short = t.Name
		}
		view.TeamShorts = append(view.TeamShorts, short)
		view.TeamColors = append(view.TeamColors, t.Color)
		if ag, ok := db.FindAgeGroup(t.AgeGroupID); ok {
			view.AgeLabels = append(view.AgeLabels, ag.Name)
		}
		coach := strings.TrimSpace(t.Coach)
		if coach != "" && !seenCoach[strings.ToLower(coach)] {
			seenCoach[strings.ToLower(coach)] = true
			view.CoachNames = append(view.CoachNames, coach)
		}
	}
	view.TeamNames = nonNil(view.TeamNames)
	view.TeamShorts = nonNil(view.TeamShorts)
	view.AgeLabels = nonNil(view.AgeLabels)
	view.CoachNames = nonNil(view.CoachNames)
	view.TeamColors = nonNil(view.TeamColors)

	env := ev.Environment
	surfaceID := ev.SurfaceID
	formatID := ev.FormatID
	if pitch, ok := db.FindPitch(ev.PitchID); ok {
		view.PitchName = pitch.Name
		if ev.VenueID == "" {
			view.VenueID = pitch.VenueID
		}
		if env == "" {
			env = pitch.Environment
		}
		if surfaceID == "" {
			surfaceID = pitch.SurfaceID
		}
		if formatID == "" {
			formatID = pitch.FormatID
		}
	}
	if venue, ok := db.FindVenue(view.VenueID); ok {
		view.VenueName = venue.Name
	}
	if surface, ok := db.FindSurface(surfaceID); ok {
		view.SurfaceName = surface.Name
		if env == "" {
			env = surface.Environment
		}
	}
	if format, ok := db.FindFormat(formatID); ok {
		view.FormatName = format.Name
	}
	switch env {
	case "indoor":
		view.EnvironmentResolved = "Indoor"
	case "outdoor":
		view.EnvironmentResolved = "Outdoor"
	default:
		view.EnvironmentResolved = ""
	}
	view.PriorityTeams = db.HasPriorityTeam(ev)
	view.LatePriority = db.LatePriority(ev)

	if ev.HomeAway == "away" {
		if strings.TrimSpace(ev.AwayLocation) != "" {
			view.LocationLabel = strings.TrimSpace(ev.AwayLocation) + " (away)"
		} else if ev.Opponent != "" {
			view.LocationLabel = "Away vs " + ev.Opponent
		} else {
			view.LocationLabel = "Away"
		}
	} else {
		view.LocationLabel = joinParts(view.VenueName, view.PitchName)
	}
	view.PlaceDetail = joinParts(view.EnvironmentResolved, view.SurfaceName, view.FormatName)
	return view
}

func (db *Database) PresentAll(events []Event) []EventView {
	out := make([]EventView, 0, len(events))
	for _, ev := range events {
		out = append(out, db.Present(ev))
	}
	return out
}

func joinParts(parts ...string) string {
	clean := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			clean = append(clean, p)
		}
	}
	return strings.Join(clean, " · ")
}

func ParseFlexible(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, fmt.Errorf("missing time")
	}
	if len(s) == 10 {
		return time.ParseInLocation("2006-01-02", s, time.Local)
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.In(time.Local), nil
	}
	if t, err := time.ParseInLocation("2006-01-02T15:04:05", s, time.Local); err == nil {
		return t, nil
	}
	if t, err := time.ParseInLocation("2006-01-02T15:04", s, time.Local); err == nil {
		return t, nil
	}
	return time.Time{}, fmt.Errorf("invalid time %q", s)
}

func FormatWall(t time.Time) string {
	return t.In(time.Local).Format(WallLayout)
}

func ClockMinutes(s string) (int, bool) {
	parts := strings.Split(strings.TrimSpace(s), ":")
	if len(parts) < 2 || len(parts) > 3 {
		return 0, false
	}
	h, err1 := strconv.Atoi(parts[0])
	m, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil || h < 0 || h > 23 || m < 0 || m > 59 {
		return 0, false
	}
	return h*60 + m, true
}

func FormatClock(mins int) string {
	if mins < 0 {
		mins = 0
	}
	return fmt.Sprintf("%02d:%02d", mins/60%24, mins%60)
}

func FormatWhen(t time.Time, clock string) string {
	if clock == "24" {
		return t.Format("15:04")
	}
	return t.Format("3:04 PM")
}

func RangesOverlap(a0, a1, b0, b1 time.Time) bool {
	return a0.Before(b1) && b0.Before(a1)
}

func OccupiedDates(start, end time.Time) []string {
	if !end.After(start) {
		return nil
	}
	d := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, start.Location())
	var out []string
	for d.Before(end) {
		out = append(out, d.Format("2006-01-02"))
		d = d.AddDate(0, 0, 1)
		if len(out) > 400 {
			break
		}
	}
	return out
}

func DefaultSettings(now time.Time) Settings {
	name := "Outdoor season"
	if now.Month() < 4 || now.Month() > 9 {
		name = "Indoor season"
	}
	label := fmt.Sprintf("%d", now.Year())
	if now.Month() >= 8 {
		label = fmt.Sprintf("%d-%02d", now.Year(), (now.Year()+1)%100)
	}
	s := Settings{
		ClubName:             "",
		SeasonName:           label + " " + name,
		SeasonStart:          now.Format("2006-01-02"),
		SeasonEnd:            now.AddDate(0, 0, 120).Format("2006-01-02"),
		TimezoneLabel:        "Local time",
		WeekStartsOn:         1,
		DayStartHour:         8,
		DayEndHour:           22,
		SlotMinutes:          30,
		DefaultView:          "timeGridWeek",
		ShowWeekends:         true,
		TimeFormat:           "12",
		PriorityCutoffAge:    11,
		EarlyWindowStart:     "08:00",
		EarlyWindowEnd:       "17:30",
		PriorityMode:         "warn",
		PriorityDays:         []int{0, 1, 2, 3, 4, 5, 6},
		PriorityCategories:   []string{"practice", "camp", "game", "other"},
		BlockPitchConflicts:  true,
		BlockTeamConflicts:   true,
		WarnCoachConflicts:   true,
		BlockBlackouts:       true,
		OlderTeamsAvoidEarly: true,
		LegendEnabled:        true,
		Theme:                AuroraTheme(),
	}
	s.Normalize()
	return s
}

func containsInt(xs []int, v int) bool {
	for _, x := range xs {
		if x == v {
			return true
		}
	}
	return false
}

func containsStr(xs []string, v string) bool {
	for _, x := range xs {
		if x == v {
			return true
		}
	}
	return false
}

func uniqueInts(in []int) []int {
	seen := map[int]bool{}
	out := make([]int, 0, len(in))
	for _, n := range in {
		if n < 0 || n > 6 || seen[n] {
			continue
		}
		seen[n] = true
		out = append(out, n)
	}
	return out
}

func uniqueStrings(in []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}
