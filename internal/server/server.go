// Package server is the local-only API and the calendar page.
package server

import (
	"io/fs"
	"net/http"
	"strings"
	"time"

	"pitchboard/internal/export"
	"pitchboard/internal/model"
	"pitchboard/internal/schedule"
	"pitchboard/internal/store"
	"pitchboard/web"
)

type App struct {
	store      *store.Store
	version    string
	files      fs.FS
	onShutdown func()
}

func New(dataDir, version string) (*App, error) {
	st, err := store.Open(dataDir)
	if err != nil {
		return nil, err
	}
	sub, err := fs.Sub(web.Files, ".")
	if err != nil {
		return nil, err
	}
	return &App{store: st, version: version, files: sub}, nil
}

func (a *App) OnShutdown(fn func()) { a.onShutdown = fn }

func (a *App) DataDir() string { return a.store.Dir() }

func (a *App) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", a.health)
	mux.HandleFunc("GET /api/bootstrap", a.bootstrap)
	mux.HandleFunc("POST /api/setup", a.setup)
	mux.HandleFunc("POST /api/reset", a.reset)
	mux.HandleFunc("PUT /api/settings", a.putSettings)
	mux.HandleFunc("GET /api/events", a.listEvents)
	mux.HandleFunc("POST /api/events/check", a.checkEvent)
	mux.HandleFunc("POST /api/events/batch", a.batchEvents)
	mux.HandleFunc("POST /api/events", a.createEvent)
	mux.HandleFunc("PUT /api/events/{id}", a.updateEvent)
	mux.HandleFunc("DELETE /api/events/{id}", a.deleteEvent)
	mux.HandleFunc("GET /api/summary", a.summary)
	mux.HandleFunc("POST /api/schedule/suggest", a.suggest)
	mux.HandleFunc("POST /api/schedule/fixes", a.fixes)
	mux.HandleFunc("POST /api/schedule/apply", a.applyFix)
	mux.HandleFunc("POST /api/schedule/earlier", a.earlier)
	mux.HandleFunc("POST /api/schedule/copy", a.copyWeek)
	mux.HandleFunc("POST /api/teams", a.createTeam)
	mux.HandleFunc("PUT /api/teams/{id}", a.updateTeam)
	mux.HandleFunc("DELETE /api/teams/{id}", a.deleteTeam)
	mux.HandleFunc("POST /api/venues", a.createVenue)
	mux.HandleFunc("PUT /api/venues/{id}", a.updateVenue)
	mux.HandleFunc("DELETE /api/venues/{id}", a.deleteVenue)
	mux.HandleFunc("POST /api/pitches", a.createPitch)
	mux.HandleFunc("PUT /api/pitches/{id}", a.updatePitch)
	mux.HandleFunc("DELETE /api/pitches/{id}", a.deletePitch)
	mux.HandleFunc("POST /api/age-groups", a.createAgeGroup)
	mux.HandleFunc("PUT /api/age-groups/{id}", a.updateAgeGroup)
	mux.HandleFunc("DELETE /api/age-groups/{id}", a.deleteAgeGroup)
	mux.HandleFunc("POST /api/surfaces", a.createSurface)
	mux.HandleFunc("PUT /api/surfaces/{id}", a.updateSurface)
	mux.HandleFunc("DELETE /api/surfaces/{id}", a.deleteSurface)
	mux.HandleFunc("POST /api/formats", a.createFormat)
	mux.HandleFunc("PUT /api/formats/{id}", a.updateFormat)
	mux.HandleFunc("DELETE /api/formats/{id}", a.deleteFormat)
	mux.HandleFunc("POST /api/event-types", a.createEventType)
	mux.HandleFunc("PUT /api/event-types/{id}", a.updateEventType)
	mux.HandleFunc("DELETE /api/event-types/{id}", a.deleteEventType)
	mux.HandleFunc("POST /api/blackouts", a.createBlackout)
	mux.HandleFunc("PUT /api/blackouts/{id}", a.updateBlackout)
	mux.HandleFunc("DELETE /api/blackouts/{id}", a.deleteBlackout)
	mux.HandleFunc("POST /api/import/preview", a.importPreview)
	mux.HandleFunc("POST /api/import", a.importSheet)
	mux.HandleFunc("POST /api/openings/{id}/book", a.bookOpening)
	mux.HandleFunc("DELETE /api/openings/{id}", a.deleteOpening)
	mux.HandleFunc("GET /api/export/csv", a.exportCSV)
	mux.HandleFunc("GET /api/export/xlsx", a.exportXLSX)
	mux.HandleFunc("GET /api/export/pdf", a.exportPDF)
	mux.HandleFunc("GET /api/export/ics", a.exportICS)
	mux.HandleFunc("GET /api/export/system/{system}", a.exportSystem)
	mux.HandleFunc("POST /api/export-profiles", a.createExportProfile)
	mux.HandleFunc("PUT /api/export-profiles/{id}", a.updateExportProfile)
	mux.HandleFunc("DELETE /api/export-profiles/{id}", a.deleteExportProfile)
	mux.HandleFunc("GET /api/backup", a.backup)
	mux.HandleFunc("POST /api/backup", a.restore)
	mux.HandleFunc("POST /api/shutdown", a.shutdown)
	mux.HandleFunc("/", a.pages)
	return a.guard(mux)
}

func (a *App) guard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; script-src 'self'; font-src 'self'")
		if strings.HasPrefix(r.URL.Path, "/api/") && r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodOptions {
			if r.Header.Get("X-Pitchboard") != "1" {
				a.writeJSON(w, http.StatusForbidden, map[string]string{"error": "Open Pitchboard from this computer."})
				return
			}
			if origin := r.Header.Get("Origin"); origin != "" && origin != "http://"+r.Host && origin != "https://"+r.Host {
				a.writeJSON(w, http.StatusForbidden, map[string]string{"error": "That page cannot change the schedule."})
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func (a *App) pages(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/api/") {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Cache-Control", "no-cache")
	http.FileServer(http.FS(a.files)).ServeHTTP(w, r)
}

func (a *App) health(w http.ResponseWriter, r *http.Request) {
	a.writeJSON(w, 200, map[string]string{"app": "pitchboard", "version": a.version})
}

func (a *App) bootstrap(w http.ResponseWriter, r *http.Request) {
	db := a.store.View()
	a.writeJSON(w, 200, map[string]any{
		"version":        a.version,
		"dataDir":        a.store.Dir(),
		"initialized":    db.Settings.Initialized,
		"settings":       db.Settings,
		"ageGroups":      db.AgeGroups,
		"teams":          db.Teams,
		"venues":         db.Venues,
		"pitches":        db.Pitches,
		"surfaces":       db.Surfaces,
		"formats":        db.Formats,
		"eventTypes":     db.EventTypes,
		"blackouts":      db.Blackouts,
		"exportProfiles": db.ExportProfiles,
	})
}

type eventBody struct {
	model.Event
	AcknowledgeLate bool   `json:"acknowledgeLate"`
	ApplyTo         string `json:"applyTo"`
	Repeat          string `json:"repeat"`
	RepeatUntil     string `json:"repeatUntil"`
	RepeatCount     int    `json:"repeatCount"`
}

func (a *App) createEvent(w http.ResponseWriter, r *http.Request) {
	var body eventBody
	if err := a.readJSON(r, &body); err != nil {
		a.fail(w, err)
		return
	}
	var saved []model.EventView
	var issues []schedule.Issue
	err := a.store.Update(func(db *model.Database) error {
		expanded, err := schedule.Expand(body.Event, body.Repeat, body.RepeatUntil, body.RepeatCount)
		if err != nil {
			return bad(err.Error())
		}
		opt := schedule.WriteOptions{AcknowledgeLate: body.AcknowledgeLate}
		res := schedule.CheckAll(db, expanded, opt)
		issues = res.Issues
		if !res.OK {
			return reject(res)
		}
		series := ""
		if len(expanded) > 1 {
			series = db.NewID("series")
		}
		now := time.Now().Format(time.RFC3339)
		for _, ev := range expanded {
			ev, _ = schedule.Normalize(db, ev)
			if body.ID != "" && len(expanded) == 1 {
				if _, exists := db.FindEvent(body.ID); exists {
					return bad("That booking already exists.")
				}
				ev.ID = body.ID
			} else {
				ev.ID = db.NewID("evt")
			}
			if series != "" {
				ev.SeriesID = series
			}
			ev.CreatedAt = now
			ev.UpdatedAt = now
			db.Events = append(db.Events, ev)
			saved = append(saved, db.Present(ev))
		}
		return nil
	})
	if err != nil {
		a.fail(w, err)
		return
	}
	a.writeJSON(w, 200, map[string]any{"events": saved, "issues": issues})
}

func (a *App) batchEvents(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Events          []model.Event `json:"events"`
		AcknowledgeLate bool          `json:"acknowledgeLate"`
	}
	if err := a.readJSON(r, &body); err != nil {
		a.fail(w, err)
		return
	}
	if len(body.Events) == 0 {
		a.writeJSON(w, 400, map[string]string{"error": "Nothing to add."})
		return
	}
	if len(body.Events) > 500 {
		a.writeJSON(w, 400, map[string]string{"error": "Add at most 500 bookings at once."})
		return
	}
	var saved []model.EventView
	var issues []schedule.Issue
	err := a.store.Update(func(db *model.Database) error {
		opt := schedule.WriteOptions{AcknowledgeLate: body.AcknowledgeLate}
		res := schedule.CheckAll(db, body.Events, opt)
		issues = res.Issues
		if !res.OK {
			return reject(res)
		}
		now := time.Now().Format(time.RFC3339)
		series := ""
		if len(body.Events) > 1 {
			series = db.NewID("series")
		}
		for _, ev := range body.Events {
			ev, _ = schedule.Normalize(db, ev)
			ev.ID = db.NewID("evt")
			if ev.SeriesID == "" && series != "" {
				ev.SeriesID = series
			}
			ev.CreatedAt = now
			ev.UpdatedAt = now
			db.Events = append(db.Events, ev)
			saved = append(saved, db.Present(ev))
		}
		return nil
	})
	if err != nil {
		a.fail(w, err)
		return
	}
	a.writeJSON(w, 200, map[string]any{"events": saved, "issues": issues})
}

func (a *App) updateEvent(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var body eventBody
	if err := a.readJSON(r, &body); err != nil {
		a.fail(w, err)
		return
	}
	var saved []model.EventView
	var issues []schedule.Issue
	err := a.store.Update(func(db *model.Database) error {
		old, ok := db.FindEvent(id)
		if !ok {
			return missing("That booking is no longer on the calendar.")
		}
		incoming := body.Event
		incoming.ID = old.ID
		incoming.CreatedAt = old.CreatedAt
		if incoming.SeriesID == "" {
			incoming.SeriesID = old.SeriesID
		}
		normalized, val := schedule.Normalize(db, incoming)
		if len(filterLevel(val, "error")) > 0 {
			return reject(schedule.CheckResult{Issues: val})
		}
		normalized.UpdatedAt = time.Now().Format(time.RFC3339)
		targets := []model.Event{normalized}
		if body.ApplyTo == "series" && old.SeriesID != "" {
			targets = retargetSeries(db, old, normalized)
		}
		ignore := make([]string, len(targets))
		for i, ev := range targets {
			ignore[i] = ev.ID
		}
		res := schedule.CheckAll(db, targets, schedule.WriteOptions{AcknowledgeLate: body.AcknowledgeLate, IgnoreIDs: ignore})
		issues = res.Issues
		if !res.OK {
			return reject(res)
		}
		replaceEvents(db, targets)
		for _, ev := range targets {
			if current, ok := db.FindEvent(ev.ID); ok {
				saved = append(saved, db.Present(current))
			}
		}
		return nil
	})
	if err != nil {
		a.fail(w, err)
		return
	}
	a.writeJSON(w, 200, map[string]any{"events": saved, "issues": issues})
}

func retargetSeries(db *model.Database, old, incoming model.Event) []model.Event {
	oldStart, err1 := model.ParseFlexible(old.Start)
	newStart, err2 := model.ParseFlexible(incoming.Start)
	oldEnd, err3 := model.ParseFlexible(old.End)
	newEnd, err4 := model.ParseFlexible(incoming.End)
	if err1 != nil || err2 != nil || err3 != nil || err4 != nil {
		return []model.Event{incoming}
	}
	dStart := newStart.Sub(oldStart)
	dEnd := newEnd.Sub(oldEnd)
	out := []model.Event{}
	for _, ev := range db.Events {
		if ev.SeriesID != old.SeriesID {
			continue
		}
		if ev.ID == old.ID {
			out = append(out, incoming)
			continue
		}
		s, e1 := model.ParseFlexible(ev.Start)
		e, e2 := model.ParseFlexible(ev.End)
		if e1 != nil || e2 != nil {
			continue
		}
		ev.Start = model.FormatWall(s.Add(dStart))
		ev.End = model.FormatWall(e.Add(dEnd))
		ev.EventTypeID = incoming.EventTypeID
		ev.Title = incoming.Title
		ev.TeamIDs = append([]string{}, incoming.TeamIDs...)
		ev.VenueID = incoming.VenueID
		ev.PitchID = incoming.PitchID
		ev.SurfaceID = incoming.SurfaceID
		ev.Environment = incoming.Environment
		ev.FormatID = incoming.FormatID
		ev.HomeAway = incoming.HomeAway
		ev.Opponent = incoming.Opponent
		ev.AwayLocation = incoming.AwayLocation
		ev.Notes = incoming.Notes
		ev.Status = incoming.Status
		ev.AllDay = incoming.AllDay
		ev.UpdatedAt = incoming.UpdatedAt
		ev, _ = schedule.Normalize(db, ev)
		out = append(out, ev)
	}
	return out
}

func replaceEvents(db *model.Database, events []model.Event) {
	byID := map[string]model.Event{}
	for _, ev := range events {
		byID[ev.ID] = ev
	}
	for i, ev := range db.Events {
		if next, ok := byID[ev.ID]; ok {
			db.Events[i] = next
		}
	}
}

func (a *App) deleteEvent(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	scope := r.URL.Query().Get("scope")
	var deleted []model.Event
	err := a.store.Update(func(db *model.Database) error {
		target, ok := db.FindEvent(id)
		if !ok {
			return missing("That booking is no longer on the calendar.")
		}
		keep := db.Events[:0]
		for _, ev := range db.Events {
			drop := ev.ID == id || (scope == "series" && target.SeriesID != "" && ev.SeriesID == target.SeriesID)
			if drop {
				deleted = append(deleted, ev)
				continue
			}
			keep = append(keep, ev)
		}
		db.Events = keep
		return nil
	})
	if err != nil {
		a.fail(w, err)
		return
	}
	a.writeJSON(w, 200, map[string]any{"deleted": deleted})
}

func (a *App) checkEvent(w http.ResponseWriter, r *http.Request) {
	var body eventBody
	if err := a.readJSON(r, &body); err != nil {
		a.fail(w, err)
		return
	}
	db := a.store.View()
	res := schedule.Check(&db, body.Event, schedule.WriteOptions{AcknowledgeLate: body.AcknowledgeLate, IgnoreIDs: []string{body.ID}})
	ev, _ := schedule.Normalize(&db, body.Event)
	view := db.Present(ev)
	a.writeJSON(w, 200, map[string]any{
		"ok": res.OK, "issues": res.Issues, "latePriority": res.LatePriority,
		"locationLabel": view.LocationLabel, "placeDetail": view.PlaceDetail, "title": view.Title,
	})
}

func (a *App) listEvents(w http.ResponseWriter, r *http.Request) {
	db := a.store.View()
	q := a.query(r)
	rows := export.Select(&db, q)
	a.writeJSON(w, 200, map[string]any{"events": rows, "openings": openingsIn(&db, q)})
}

func (a *App) summary(w http.ResponseWriter, r *http.Request) {
	db := a.store.View()
	q := a.query(r)
	rows := export.Select(&db, q)
	type day struct {
		Date  string `json:"date"`
		Count int    `json:"count"`
		Late  int    `json:"late"`
	}
	by := map[string]*day{}
	var order []string
	for _, row := range rows {
		start, err := model.ParseFlexible(row.Start)
		if err != nil {
			continue
		}
		key := start.Format("2006-01-02")
		slot, ok := by[key]
		if !ok {
			slot = &day{Date: key}
			by[key] = slot
			order = append(order, key)
		}
		slot.Count++
		if row.LatePriority {
			slot.Late++
		}
	}
	days := make([]day, 0, len(order))
	for _, key := range order {
		days = append(days, *by[key])
	}
	a.writeJSON(w, 200, map[string]any{"days": days})
}
