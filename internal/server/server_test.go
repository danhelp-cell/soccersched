package server

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSetupConflictAndExport(t *testing.T) {
	app, err := New(t.TempDir(), "test")
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(app.Handler())
	defer ts.Close()
	res := call(t, ts, http.MethodPost, "/api/setup", `{
		"sample": true,
		"settings": {
			"clubName": "Test FC",
			"seasonName": "Indoor",
			"seasonStart": "2026-10-01",
			"seasonEnd": "2027-03-01",
			"priorityCutoffAge": 11,
			"earlyWindowStart": "08:00",
			"earlyWindowEnd": "17:30",
			"priorityMode": "strict",
			"dayStartHour": 8,
			"dayEndHour": 22,
			"slotMinutes": 30,
			"blockPitchConflicts": true,
			"blockTeamConflicts": true
		}
	}`)
	if res.StatusCode != 200 {
		t.Fatalf("setup %d %s", res.StatusCode, res.Body)
	}
	events := call(t, ts, http.MethodGet, "/api/events?start=2026-01-01T00:00:00&end=2027-01-01T00:00:00", "")
	var listed struct {
		Events []map[string]any `json:"events"`
	}
	if err := json.Unmarshal(events.Raw, &listed); err != nil {
		t.Fatal(err)
	}
	if len(listed.Events) < 10 {
		t.Fatalf("expected sample events, got %d", len(listed.Events))
	}
	var late map[string]any
	for _, ev := range listed.Events {
		if ev["title"] == "U8 Red Practice" && ev["latePriority"] == true {
			late = ev
		}
	}
	if late == nil {
		t.Fatal("sample should flag the late U8 practice")
	}
	delete(late, "id")
	delete(late, "seriesId")
	raw, _ := json.Marshal(late)
	conflict := call(t, ts, http.MethodPost, "/api/events", string(raw))
	if conflict.StatusCode != 409 {
		t.Fatalf("duplicate pitch should conflict, got %d %s", conflict.StatusCode, conflict.Body)
	}
	file := call(t, ts, http.MethodGet, "/api/export/csv?start=2026-01-01T00:00:00&end=2027-01-01T00:00:00", "")
	if file.StatusCode != 200 || !strings.Contains(file.Body, "Test FC") && !bytes.Contains(file.Raw, []byte("U8 Red")) {
		t.Fatalf("csv export failed: %d %s", file.StatusCode, file.Body[:min(200, len(file.Body))])
	}
	if !bytes.HasPrefix(file.Raw, []byte{0xEF, 0xBB, 0xBF}) {
		t.Fatal("download should keep the Excel BOM")
	}
}

type reply struct {
	StatusCode int
	Body       string
	Raw        []byte
}

func call(t *testing.T, ts *httptest.Server, method, path, body string) reply {
	t.Helper()
	var rdr io.Reader
	if body != "" {
		rdr = strings.NewReader(body)
	}
	req, err := http.NewRequest(method, ts.URL+path, rdr)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("X-Pitchboard", "1")
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	return reply{StatusCode: res.StatusCode, Body: string(raw), Raw: raw}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
