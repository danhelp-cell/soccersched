package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"pitchboard/internal/export"
	"pitchboard/internal/model"
	"pitchboard/internal/schedule"
	"pitchboard/internal/store"
)

func (a *App) setup(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Settings model.Settings `json:"settings"`
		Sample   bool           `json:"sample"`
	}
	if err := a.readJSON(r, &body); err != nil {
		a.fail(w, err)
		return
	}
	err := a.store.Update(func(db *model.Database) error {
		if db.Settings.Initialized {
			return bad("This computer already has a club. Reset it from Settings if you want to start over.")
		}
		body.Settings.Normalize()
		body.Settings.Initialized = true
		if strings.TrimSpace(body.Settings.ClubName) == "" {
			return bad("Enter the club name.")
		}
		db.Settings = body.Settings
		store.EnsureCatalogs(db)
		if body.Sample {
			store.ApplySample(db, time.Now())
		}
		return nil
	})
	if err != nil {
		a.fail(w, err)
		return
	}
	a.bootstrap(w, r)
}

func (a *App) reset(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Mode string `json:"mode"`
	}
	if err := a.readJSON(r, &body); err != nil {
		a.fail(w, err)
		return
	}
	err := a.store.Update(func(db *model.Database) error {
		switch body.Mode {
		case "sample":
			store.EnsureCatalogs(db)
			store.ApplySample(db, time.Now())
			db.Settings.Initialized = true
		case "blank":
			store.ClearClub(db)
			db.Settings.Initialized = true
		default:
			return bad("Choose sample or blank.")
		}
		return nil
	})
	if err != nil {
		a.fail(w, err)
		return
	}
	a.bootstrap(w, r)
}

func (a *App) putSettings(w http.ResponseWriter, r *http.Request) {
	var body model.Settings
	if err := a.readJSON(r, &body); err != nil {
		a.fail(w, err)
		return
	}
	err := a.store.Update(func(db *model.Database) error {
		body.Normalize()
		body.Initialized = true
		if body.ClubName == "" {
			return bad("Enter the club name.")
		}
		db.Settings = body
		return nil
	})
	if err != nil {
		a.fail(w, err)
		return
	}
	a.writeJSON(w, 200, map[string]any{"settings": a.store.View().Settings})
}

func (a *App) suggest(w http.ResponseWriter, r *http.Request) {
	var body schedule.SuggestRequest
	if err := a.readJSON(r, &body); err != nil {
		a.fail(w, err)
		return
	}
	db := a.store.View()
	result, err := schedule.Suggest(&db, body)
	if err != nil {
		a.writeJSON(w, 400, map[string]string{"error": err.Error()})
		return
	}
	views := make([]model.EventView, 0, len(result.Proposals))
	for _, p := range result.Proposals {
		views = append(views, db.Present(p.Event))
	}
	a.writeJSON(w, 200, map[string]any{"proposals": result.Proposals, "views": views, "unscheduled": result.Unscheduled})
}

func (a *App) fixes(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Start string `json:"start"`
		End   string `json:"end"`
	}
	if err := a.readJSON(r, &body); err != nil {
		a.fail(w, err)
		return
	}
	from, to, err := rangeOrSeason(a.store.View(), body.Start, body.End)
	if err != nil {
		a.writeJSON(w, 400, map[string]string{"error": err.Error()})
		return
	}
	db := a.store.View()
	a.writeJSON(w, 200, map[string]any{"proposals": schedule.Fixes(&db, from, to)})
}

func (a *App) earlier(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Event model.Event `json:"event"`
		Limit int         `json:"limit"`
	}
	if err := a.readJSON(r, &body); err != nil {
		a.fail(w, err)
		return
	}
	db := a.store.View()
	a.writeJSON(w, 200, map[string]any{"slots": schedule.Earlier(&db, body.Event, body.Limit)})
}

func (a *App) applyFix(w http.ResponseWriter, r *http.Request) {
	var body schedule.Proposal
	if err := a.readJSON(r, &body); err != nil {
		a.fail(w, err)
		return
	}
	var saved []model.EventView
	err := a.store.Update(func(db *model.Database) error {
		ev, ok := db.FindEvent(body.EventID)
		if !ok {
			return missing("That booking is no longer on the calendar.")
		}
		ev.Start = body.Start
		ev.End = body.End
		if body.PitchID != "" {
			ev.PitchID = body.PitchID
			ev.VenueID = body.VenueID
			ev.SurfaceID, ev.Environment, ev.FormatID = "", "", ""
		}
		ev, val := schedule.Normalize(db, ev)
		if len(filterLevel(val, "error")) > 0 {
			return reject(schedule.CheckResult{Issues: val})
		}
		ignore := []string{ev.ID}
		var other model.Event
		hasOther := false
		if body.Kind == "swap" && body.SwapEventID != "" {
			other, hasOther = db.FindEvent(body.SwapEventID)
			if !hasOther {
				return missing("The other booking is no longer on the calendar.")
			}
			other.Start = body.SwapStart
			other.End = body.SwapEnd
			if body.SwapPitchID != "" {
				other.PitchID = body.SwapPitchID
				other.VenueID = body.SwapVenueID
				other.SurfaceID, other.Environment, other.FormatID = "", "", ""
			}
			other, val = schedule.Normalize(db, other)
			if len(filterLevel(val, "error")) > 0 {
				return reject(schedule.CheckResult{Issues: val})
			}
			ignore = append(ignore, other.ID)
		}
		res := schedule.Check(db, ev, schedule.WriteOptions{AcknowledgeLate: true, IgnoreIDs: ignore})
		if !res.OK {
			return reject(res)
		}
		targets := []model.Event{ev}
		if hasOther {
			res = schedule.Check(db, other, schedule.WriteOptions{AcknowledgeLate: true, IgnoreIDs: ignore})
			if !res.OK {
				return reject(res)
			}
			targets = append(targets, other)
		}
		now := time.Now().Format(time.RFC3339)
		for i := range targets {
			targets[i].UpdatedAt = now
		}
		replaceEvents(db, targets)
		for _, item := range targets {
			saved = append(saved, db.Present(item))
		}
		return nil
	})
	if err != nil {
		a.fail(w, err)
		return
	}
	a.writeJSON(w, 200, map[string]any{"events": saved})
}

func (a *App) copyWeek(w http.ResponseWriter, r *http.Request) {
	var body struct {
		FromStart string `json:"fromStart"`
		FromEnd   string `json:"fromEnd"`
		Weeks     int    `json:"weeks"`
	}
	if err := a.readJSON(r, &body); err != nil {
		a.fail(w, err)
		return
	}
	if body.Weeks <= 0 {
		body.Weeks = 1
	}
	if body.Weeks > 12 {
		body.Weeks = 12
	}
	from, err1 := model.ParseFlexible(body.FromStart)
	to, err2 := model.ParseFlexible(body.FromEnd)
	if err1 != nil || err2 != nil || !to.After(from) {
		a.writeJSON(w, 400, map[string]string{"error": "Choose the week to copy."})
		return
	}
	shift := time.Duration(body.Weeks) * 7 * 24 * time.Hour
	var created []model.EventView
	var skipped []string
	err := a.store.Update(func(db *model.Database) error {
		now := time.Now().Format(time.RFC3339)
		for _, ev := range append([]model.Event{}, db.Events...) {
			if ev.Status == "cancelled" {
				continue
			}
			start, e1 := model.ParseFlexible(ev.Start)
			end, e2 := model.ParseFlexible(ev.End)
			if e1 != nil || e2 != nil || !model.RangesOverlap(start, end, from, to) {
				continue
			}
			ev.ID = ""
			ev.SeriesID = ""
			ev.Start = model.FormatWall(start.Add(shift))
			ev.End = model.FormatWall(end.Add(shift))
			ev, val := schedule.Normalize(db, ev)
			res := schedule.Check(db, ev, schedule.WriteOptions{AcknowledgeLate: true})
			if len(filterLevel(val, "error")) > 0 || !res.OK {
				skipped = append(skipped, ev.Title+" was skipped because the new time is taken or closed.")
				continue
			}
			ev.ID = db.NewID("evt")
			ev.CreatedAt = now
			ev.UpdatedAt = now
			db.Events = append(db.Events, ev)
			created = append(created, db.Present(ev))
		}
		return nil
	})
	if err != nil {
		a.fail(w, err)
		return
	}
	if created == nil {
		created = []model.EventView{}
	}
	if skipped == nil {
		skipped = []string{}
	}
	a.writeJSON(w, 200, map[string]any{"events": created, "skipped": skipped})
}

func (a *App) createTeam(w http.ResponseWriter, r *http.Request) { a.saveTeam(w, r, "") }
func (a *App) updateTeam(w http.ResponseWriter, r *http.Request) { a.saveTeam(w, r, r.PathValue("id")) }

func (a *App) saveTeam(w http.ResponseWriter, r *http.Request, id string) {
	var body model.Team
	if err := a.readJSON(r, &body); err != nil {
		a.fail(w, err)
		return
	}
	err := a.store.Update(func(db *model.Database) error {
		body.Name = strings.TrimSpace(body.Name)
		body.ShortName = strings.TrimSpace(body.ShortName)
		if body.Name == "" {
			return bad("Enter the team name.")
		}
		if _, ok := db.FindAgeGroup(body.AgeGroupID); !ok {
			return bad("Choose an age group.")
		}
		switch body.Priority {
		case "always", "never", "auto":
		default:
			body.Priority = "auto"
		}
		if body.ShortName == "" {
			body.ShortName = body.Name
		}
		if id == "" {
			body.ID = db.NewID("team")
			body.Active = true
			db.Teams = append(db.Teams, body)
			return nil
		}
		for i, t := range db.Teams {
			if t.ID == id {
				body.ID = id
				db.Teams[i] = body
				return nil
			}
		}
		return missing("That team was not found.")
	})
	if err != nil {
		a.fail(w, err)
		return
	}
	a.bootstrap(w, r)
}

func (a *App) deleteTeam(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	err := a.store.Update(func(db *model.Database) error {
		n := 0
		for _, ev := range db.Events {
			for _, tid := range ev.TeamIDs {
				if tid == id {
					n++
				}
			}
		}
		if n > 0 {
			return bad(fmt.Sprintf("This team is on %d bookings. Move or delete those first.", n))
		}
		next := db.Teams[:0]
		found := false
		for _, t := range db.Teams {
			if t.ID == id {
				found = true
				continue
			}
			next = append(next, t)
		}
		if !found {
			return missing("That team was not found.")
		}
		db.Teams = next
		return nil
	})
	if err != nil {
		a.fail(w, err)
		return
	}
	a.bootstrap(w, r)
}

func (a *App) createVenue(w http.ResponseWriter, r *http.Request) { a.saveVenue(w, r, "") }
func (a *App) updateVenue(w http.ResponseWriter, r *http.Request) {
	a.saveVenue(w, r, r.PathValue("id"))
}

func (a *App) saveVenue(w http.ResponseWriter, r *http.Request, id string) {
	var body model.Venue
	if err := a.readJSON(r, &body); err != nil {
		a.fail(w, err)
		return
	}
	err := a.store.Update(func(db *model.Database) error {
		body.Name = strings.TrimSpace(body.Name)
		if body.Name == "" {
			return bad("Enter the location name.")
		}
		if _, ok := model.ClockMinutes(body.OpenTime); body.OpenTime != "" && !ok {
			return bad("Opening time should look like 08:00.")
		}
		if _, ok := model.ClockMinutes(body.CloseTime); body.CloseTime != "" && !ok {
			return bad("Closing time should look like 22:00.")
		}
		if id == "" {
			body.ID = db.NewID("ven")
			body.Active = true
			if body.Days == nil {
				body.Days = []int{0, 1, 2, 3, 4, 5, 6}
			}
			db.Venues = append(db.Venues, body)
			return nil
		}
		for i, v := range db.Venues {
			if v.ID == id {
				body.ID = id
				db.Venues[i] = body
				return nil
			}
		}
		return missing("That location was not found.")
	})
	if err != nil {
		a.fail(w, err)
		return
	}
	a.bootstrap(w, r)
}

func (a *App) deleteVenue(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	err := a.store.Update(func(db *model.Database) error {
		for _, ev := range db.Events {
			if ev.VenueID == id {
				return bad("Bookings still use this location.")
			}
		}
		for _, p := range db.Pitches {
			if p.VenueID == id {
				return bad("Remove the pitches at this location first.")
			}
		}
		next := db.Venues[:0]
		found := false
		for _, v := range db.Venues {
			if v.ID == id {
				found = true
				continue
			}
			next = append(next, v)
		}
		if !found {
			return missing("That location was not found.")
		}
		db.Venues = next
		return nil
	})
	if err != nil {
		a.fail(w, err)
		return
	}
	a.bootstrap(w, r)
}

func (a *App) createPitch(w http.ResponseWriter, r *http.Request) { a.savePitch(w, r, "") }
func (a *App) updatePitch(w http.ResponseWriter, r *http.Request) {
	a.savePitch(w, r, r.PathValue("id"))
}

func (a *App) savePitch(w http.ResponseWriter, r *http.Request, id string) {
	var body model.Pitch
	if err := a.readJSON(r, &body); err != nil {
		a.fail(w, err)
		return
	}
	err := a.store.Update(func(db *model.Database) error {
		body.Name = strings.TrimSpace(body.Name)
		if body.Name == "" {
			return bad("Enter the pitch name.")
		}
		if _, ok := db.FindVenue(body.VenueID); !ok {
			return bad("Choose the location this pitch belongs to.")
		}
		if body.Environment != "indoor" && body.Environment != "outdoor" {
			return bad("Mark the pitch indoor or outdoor.")
		}
		if _, ok := db.FindSurface(body.SurfaceID); !ok {
			return bad("Choose the surface.")
		}
		if _, ok := db.FindFormat(body.FormatID); !ok {
			return bad("Choose the pitch type.")
		}
		if id == "" {
			body.ID = db.NewID("pit")
			body.Active = true
			db.Pitches = append(db.Pitches, body)
			return nil
		}
		for i, p := range db.Pitches {
			if p.ID == id {
				body.ID = id
				db.Pitches[i] = body
				return nil
			}
		}
		return missing("That pitch was not found.")
	})
	if err != nil {
		a.fail(w, err)
		return
	}
	a.bootstrap(w, r)
}

func (a *App) deletePitch(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	err := a.store.Update(func(db *model.Database) error {
		for _, ev := range db.Events {
			if ev.PitchID == id {
				return bad("Bookings still use this pitch.")
			}
		}
		next := db.Pitches[:0]
		found := false
		for _, p := range db.Pitches {
			if p.ID == id {
				found = true
				continue
			}
			next = append(next, p)
		}
		if !found {
			return missing("That pitch was not found.")
		}
		db.Pitches = next
		return nil
	})
	if err != nil {
		a.fail(w, err)
		return
	}
	a.bootstrap(w, r)
}

func (a *App) createAgeGroup(w http.ResponseWriter, r *http.Request) {
	a.saveCatalog(w, r, "", func(db *model.Database, raw jsonRaw, id string) error {
		var body model.AgeGroup
		if err := jsonCopy(raw, &body); err != nil {
			return err
		}
		body.Name = strings.TrimSpace(body.Name)
		if body.Name == "" {
			return bad("Enter the age group name.")
		}
		if body.MaxAge < body.MinAge {
			return bad("The oldest age must be at least the youngest age.")
		}
		if id == "" {
			body.ID = db.NewID("ag")
			db.AgeGroups = append(db.AgeGroups, body)
			return nil
		}
		for i := range db.AgeGroups {
			if db.AgeGroups[i].ID == id {
				body.ID = id
				db.AgeGroups[i] = body
				return nil
			}
		}
		return missing("That age group was not found.")
	})
}
func (a *App) updateAgeGroup(w http.ResponseWriter, r *http.Request) {
	a.createAgeGroup(w, r)
}

func (a *App) deleteAgeGroup(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	err := a.store.Update(func(db *model.Database) error {
		for _, t := range db.Teams {
			if t.AgeGroupID == id {
				return bad("A team still uses this age group.")
			}
		}
		return dropByID(&db.AgeGroups, id, "That age group was not found.")
	})
	if err != nil {
		a.fail(w, err)
		return
	}
	a.bootstrap(w, r)
}

func (a *App) createSurface(w http.ResponseWriter, r *http.Request) {
	a.saveNamed(w, r, func(db *model.Database, name, env, id string, raw jsonRaw) error {
		if env != "indoor" && env != "outdoor" && env != "either" {
			return bad("Choose indoor, outdoor, or either.")
		}
		body := model.Surface{ID: id, Name: name, Environment: env}
		if id == "" {
			body.ID = db.NewID("surf")
			db.Surfaces = append(db.Surfaces, body)
			return nil
		}
		for i := range db.Surfaces {
			if db.Surfaces[i].ID == id {
				db.Surfaces[i] = body
				return nil
			}
		}
		return missing("That surface was not found.")
	})
}
func (a *App) updateSurface(w http.ResponseWriter, r *http.Request) { a.createSurface(w, r) }

func (a *App) deleteSurface(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	err := a.store.Update(func(db *model.Database) error {
		for _, p := range db.Pitches {
			if p.SurfaceID == id {
				return bad("A pitch still uses this surface.")
			}
		}
		return dropByID(&db.Surfaces, id, "That surface was not found.")
	})
	if err != nil {
		a.fail(w, err)
		return
	}
	a.bootstrap(w, r)
}

func (a *App) createFormat(w http.ResponseWriter, r *http.Request) {
	a.saveNamed(w, r, func(db *model.Database, name, _, id string, raw jsonRaw) error {
		var body model.Format
		_ = jsonCopy(raw, &body)
		body.Name = name
		body.ID = id
		if id == "" {
			body.ID = db.NewID("fmt")
			db.Formats = append(db.Formats, body)
			return nil
		}
		for i := range db.Formats {
			if db.Formats[i].ID == id {
				body.Sort = db.Formats[i].Sort
				if body.Sort == 0 {
					body.Sort = db.Formats[i].Sort
				}
				db.Formats[i] = body
				return nil
			}
		}
		return missing("That pitch type was not found.")
	})
}
func (a *App) updateFormat(w http.ResponseWriter, r *http.Request) { a.createFormat(w, r) }

func (a *App) deleteFormat(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	err := a.store.Update(func(db *model.Database) error {
		for _, p := range db.Pitches {
			if p.FormatID == id {
				return bad("A pitch still uses this pitch type.")
			}
		}
		return dropByID(&db.Formats, id, "That pitch type was not found.")
	})
	if err != nil {
		a.fail(w, err)
		return
	}
	a.bootstrap(w, r)
}

func (a *App) createEventType(w http.ResponseWriter, r *http.Request) {
	a.saveCatalog(w, r, "", func(db *model.Database, raw jsonRaw, id string) error {
		var body model.EventType
		if err := jsonCopy(raw, &body); err != nil {
			return err
		}
		body.Name = strings.TrimSpace(body.Name)
		if body.Name == "" {
			return bad("Enter a name.")
		}
		switch body.Category {
		case "practice", "camp", "game", "other":
		default:
			return bad("Category must be practice, camp, game, or other.")
		}
		if body.DefaultMinutes <= 0 {
			body.DefaultMinutes = 60
		}
		if id == "" {
			body.ID = db.NewID("type")
			db.EventTypes = append(db.EventTypes, body)
			return nil
		}
		for i := range db.EventTypes {
			if db.EventTypes[i].ID == id {
				body.ID = id
				db.EventTypes[i] = body
				return nil
			}
		}
		return missing("That type was not found.")
	})
}
func (a *App) updateEventType(w http.ResponseWriter, r *http.Request) { a.createEventType(w, r) }

func (a *App) deleteEventType(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	err := a.store.Update(func(db *model.Database) error {
		if len(db.EventTypes) <= 1 {
			return bad("Keep at least one event type.")
		}
		for _, ev := range db.Events {
			if ev.EventTypeID == id {
				return bad("Bookings still use this type.")
			}
		}
		return dropByID(&db.EventTypes, id, "That type was not found.")
	})
	if err != nil {
		a.fail(w, err)
		return
	}
	a.bootstrap(w, r)
}

func (a *App) createBlackout(w http.ResponseWriter, r *http.Request) {
	a.saveCatalog(w, r, "", func(db *model.Database, raw jsonRaw, id string) error {
		var body model.Blackout
		if err := jsonCopy(raw, &body); err != nil {
			return err
		}
		if len(body.Start) != 10 || len(body.End) != 10 || body.End < body.Start {
			return bad("Enter a start and end date.")
		}
		body.Reason = strings.TrimSpace(body.Reason)
		if body.VenueID != "" {
			if _, ok := db.FindVenue(body.VenueID); !ok {
				return bad("That location was not found.")
			}
		}
		if id == "" {
			body.ID = db.NewID("blk")
			db.Blackouts = append(db.Blackouts, body)
			return nil
		}
		for i := range db.Blackouts {
			if db.Blackouts[i].ID == id {
				body.ID = id
				db.Blackouts[i] = body
				return nil
			}
		}
		return missing("That closure was not found.")
	})
}
func (a *App) updateBlackout(w http.ResponseWriter, r *http.Request) { a.createBlackout(w, r) }

func (a *App) deleteBlackout(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	err := a.store.Update(func(db *model.Database) error {
		return dropByID(&db.Blackouts, id, "That closure was not found.")
	})
	if err != nil {
		a.fail(w, err)
		return
	}
	a.bootstrap(w, r)
}

func (a *App) exportCSV(w http.ResponseWriter, r *http.Request) {
	db := a.store.View()
	q := a.query(r)
	body, err := export.CSV(&db, export.Select(&db, q))
	if err != nil {
		a.writeJSON(w, 500, map[string]string{"error": "Could not build the CSV file."})
		return
	}
	a.download(w, export.FileName(db.Settings.ClubName, "csv", q.Start, q.End), "text/csv; charset=utf-8", body)
}

func (a *App) exportXLSX(w http.ResponseWriter, r *http.Request) {
	db := a.store.View()
	q := a.query(r)
	body, err := export.XLSX(&db, export.Select(&db, q))
	if err != nil {
		a.writeJSON(w, 500, map[string]string{"error": "Could not build the Excel file."})
		return
	}
	a.download(w, export.FileName(db.Settings.ClubName, "xlsx", q.Start, q.End), "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", body)
}

func (a *App) exportPDF(w http.ResponseWriter, r *http.Request) {
	db := a.store.View()
	q := a.query(r)
	body, err := export.PDF(&db, export.Select(&db, q), q.Start, q.End)
	if err != nil {
		a.writeJSON(w, 500, map[string]string{"error": "Could not build the PDF."})
		return
	}
	a.download(w, export.FileName(db.Settings.ClubName, "pdf", q.Start, q.End), "application/pdf", body)
}

func (a *App) exportSystem(w http.ResponseWriter, r *http.Request) {
	system := r.PathValue("system")
	if _, ok := export.FindSystem(system); !ok {
		a.writeJSON(w, 404, map[string]string{"error": "That export is not available."})
		return
	}
	db := a.store.View()
	categories := splitCSV(r.URL.Query().Get("categories"))
	label := system
	if profileID := strings.TrimSpace(r.URL.Query().Get("profile")); profileID != "" {
		found := false
		for _, profile := range db.ExportProfiles {
			if profile.ID != profileID {
				continue
			}
			if _, ok := export.FindSystem(profile.System); !ok {
				a.writeJSON(w, 400, map[string]string{"error": "That saved export uses an unknown system."})
				return
			}
			system = profile.System
			if len(profile.Categories) > 0 {
				categories = profile.Categories
			}
			if profile.Name != "" {
				label = profile.Name
			}
			found = true
			break
		}
		if !found {
			a.writeJSON(w, 404, map[string]string{"error": "That saved export was not found."})
			return
		}
	}
	body, skipped, err := export.SystemCSV(&db, export.Select(&db, a.query(r)), system, categories)
	if err != nil {
		a.writeJSON(w, 500, map[string]string{"error": "Could not build that export."})
		return
	}
	notice := ""
	if skipped > 0 {
		noun := "rows were"
		if skipped == 1 {
			noun = "row was"
		}
		notice = fmt.Sprintf("%d %s left out. RAMP needs a numeric team id and location id. The other systems use the team and place names when an id is blank.", skipped, noun)
	}
	q := a.query(r)
	a.downloadNotice(w, export.FileName(db.Settings.ClubName+" "+label, "csv", q.Start, q.End), "text/csv; charset=utf-8", body, notice)
}

func (a *App) createExportProfile(w http.ResponseWriter, r *http.Request) {
	a.saveCatalog(w, r, "", func(db *model.Database, raw jsonRaw, id string) error {
		var body model.ExportProfile
		if err := jsonCopy(raw, &body); err != nil {
			return err
		}
		body.Name = strings.TrimSpace(body.Name)
		if body.Name == "" {
			return bad("Name this export.")
		}
		if _, ok := export.FindSystem(body.System); !ok {
			return bad("Choose RAMP, TeamLinkt, TeamSnap, or GameChanger.")
		}
		if id == "" {
			body.ID = db.NewID("exp")
			db.ExportProfiles = append(db.ExportProfiles, body)
			return nil
		}
		for i := range db.ExportProfiles {
			if db.ExportProfiles[i].ID == id {
				body.ID = id
				db.ExportProfiles[i] = body
				return nil
			}
		}
		return missing("That saved export was not found.")
	})
}

func (a *App) updateExportProfile(w http.ResponseWriter, r *http.Request) {
	a.createExportProfile(w, r)
}

func (a *App) deleteExportProfile(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	err := a.store.Update(func(db *model.Database) error {
		return dropByID(&db.ExportProfiles, id, "That saved export was not found.")
	})
	if err != nil {
		a.fail(w, err)
		return
	}
	a.bootstrap(w, r)
}

func (a *App) exportICS(w http.ResponseWriter, r *http.Request) {
	db := a.store.View()
	q := a.query(r)
	body := export.ICS(&db, export.Select(&db, q))
	a.download(w, export.FileName(db.Settings.ClubName, "ics", q.Start, q.End), "text/calendar; charset=utf-8", body)
}

func (a *App) backup(w http.ResponseWriter, r *http.Request) {
	db := a.store.View()
	encoded, err := jsonMarshal(db)
	if err != nil {
		a.writeJSON(w, 500, map[string]string{"error": "Could not build the backup."})
		return
	}
	name := export.FileName(db.Settings.ClubName, "json", time.Time{}, time.Time{})
	name = strings.TrimSuffix(name, "-schedule.json") + "-backup.json"
	a.download(w, name, "application/json", encoded)
}

func (a *App) restore(w http.ResponseWriter, r *http.Request) {
	raw, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 20<<20))
	if err != nil {
		a.writeJSON(w, 400, map[string]string{"error": "That backup could not be read."})
		return
	}
	var incoming model.Database
	if err := json.Unmarshal(raw, &incoming); err != nil || (incoming.Version != 0 && incoming.Version != model.Version) {
		a.writeJSON(w, 400, map[string]string{"error": "That file is not a Pitchboard backup."})
		return
	}
	err = a.store.Update(func(db *model.Database) error {
		*db = incoming
		db.Version = model.Version
		db.Normalize()
		db.Settings.Initialized = true
		return nil
	})
	if err != nil {
		a.fail(w, err)
		return
	}
	a.bootstrap(w, r)
}

func (a *App) shutdown(w http.ResponseWriter, r *http.Request) {
	a.writeJSON(w, 200, map[string]string{"status": "stopping"})
	if a.onShutdown != nil {
		go a.onShutdown()
	}
}

func rangeOrSeason(db model.Database, start, end string) (time.Time, time.Time, error) {
	if start == "" {
		start = db.Settings.SeasonStart
	}
	if end == "" {
		end = db.Settings.SeasonEnd
	}
	from, err1 := model.ParseFlexible(start)
	to, err2 := model.ParseFlexible(end)
	if err1 != nil || err2 != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("choose a date range")
	}
	from = time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, time.Local)
	if len(end) == 10 {
		to = time.Date(to.Year(), to.Month(), to.Day(), 0, 0, 0, 0, time.Local).AddDate(0, 0, 1)
	}
	if !to.After(from) {
		return time.Time{}, time.Time{}, fmt.Errorf("the end must be after the start")
	}
	return from, to, nil
}
