package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"pitchboard/internal/export"
	"pitchboard/internal/model"
	"pitchboard/internal/schedule"
)

type statusError struct {
	code    int
	payload any
}

func (e *statusError) Error() string {
	if m, ok := e.payload.(map[string]any); ok {
		if s, ok := m["error"].(string); ok {
			return s
		}
	}
	return fmt.Sprintf("status %d", e.code)
}

func bad(message string) error {
	return &statusError{code: 400, payload: map[string]any{"error": message}}
}

func missing(message string) error {
	return &statusError{code: 404, payload: map[string]any{"error": message}}
}

func reject(res schedule.CheckResult) error {
	code := 422
	for _, issue := range res.Issues {
		if issue.Level == "error" && issue.Kind != "priority" {
			code = 409
			break
		}
	}
	msg := "That booking could not be saved."
	if errs := res.Errors(); len(errs) > 0 {
		msg = errs[0].Message
	}
	if res.Issues == nil {
		res.Issues = []schedule.Issue{}
	}
	return &statusError{code: code, payload: map[string]any{
		"error":        msg,
		"issues":       res.Issues,
		"latePriority": res.LatePriority,
	}}
}

func filterLevel(issues []schedule.Issue, level string) []schedule.Issue {
	var out []schedule.Issue
	for _, issue := range issues {
		if issue.Level == level {
			out = append(out, issue)
		}
	}
	return out
}

func (a *App) writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(v)
}

func (a *App) readJSON(r *http.Request, dst any) error {
	dec := json.NewDecoder(io.LimitReader(r.Body, 8<<20))
	if err := dec.Decode(dst); err != nil {
		return bad("The request could not be read.")
	}
	return nil
}

func (a *App) fail(w http.ResponseWriter, err error) {
	var se *statusError
	if errors.As(err, &se) {
		a.writeJSON(w, se.code, se.payload)
		return
	}
	a.writeJSON(w, 500, map[string]string{"error": "Something went wrong saving the schedule."})
}

func (a *App) download(w http.ResponseWriter, name, contentType string, body []byte) {
	a.downloadNotice(w, name, contentType, body, "")
}

func (a *App) downloadNotice(w http.ResponseWriter, name, contentType string, body []byte, notice string) {
	if notice != "" {
		w.Header().Set("X-Pitchboard-Notice", notice)
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", name))
	w.WriteHeader(200)
	_, _ = w.Write(body)
}

func (a *App) query(r *http.Request) export.Query {
	q := r.URL.Query()
	start, _ := model.ParseFlexible(q.Get("start"))
	end, _ := model.ParseFlexible(q.Get("end"))
	return export.Query{
		Start:            start,
		End:              end,
		TeamIDs:          splitCSV(q.Get("teams")),
		VenueIDs:         splitCSV(q.Get("venues")),
		TypeIDs:          splitCSV(q.Get("types")),
		Environments:     splitCSV(q.Get("env")),
		SurfaceIDs:       splitCSV(q.Get("surfaces")),
		IncludeCancelled: q.Get("cancelled") == "1",
		Text:             q.Get("q"),
	}
}

func splitCSV(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	var out []string
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

type jsonRaw = json.RawMessage

func jsonCopy(raw json.RawMessage, dst any) error {
	if len(raw) == 0 {
		return bad("The request could not be read.")
	}
	if err := json.Unmarshal(raw, dst); err != nil {
		return bad("The request could not be read.")
	}
	return nil
}

func jsonMarshal(v any) ([]byte, error) {
	raw, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(raw, '\n'), nil
}

func (a *App) saveCatalog(w http.ResponseWriter, r *http.Request, _ string, fn func(db *model.Database, raw json.RawMessage, id string) error) {
	raw, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		a.writeJSON(w, 400, map[string]string{"error": "The request could not be read."})
		return
	}
	id := r.PathValue("id")
	err = a.store.Update(func(db *model.Database) error {
		return fn(db, raw, id)
	})
	if err != nil {
		a.fail(w, err)
		return
	}
	a.bootstrap(w, r)
}

func (a *App) saveNamed(w http.ResponseWriter, r *http.Request, fn func(db *model.Database, name, env, id string, raw json.RawMessage) error) {
	a.saveCatalog(w, r, "", func(db *model.Database, raw json.RawMessage, id string) error {
		var body struct {
			Name        string `json:"name"`
			Environment string `json:"environment"`
		}
		if err := jsonCopy(raw, &body); err != nil {
			return err
		}
		body.Name = strings.TrimSpace(body.Name)
		if body.Name == "" {
			return bad("Enter a name.")
		}
		return fn(db, body.Name, body.Environment, id, raw)
	})
}

func dropByID[T interface{ Ident() string }](items *[]T, id, missingMsg string) error {
	list := *items
	next := list[:0]
	found := false
	for _, item := range list {
		if item.Ident() == id {
			found = true
			continue
		}
		next = append(next, item)
	}
	if !found {
		return missing(missingMsg)
	}
	*items = next
	return nil
}
