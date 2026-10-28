package intake

import (
	"archive/zip"
	"bytes"
	"strconv"
	"strings"
	"testing"
	"time"

	"pitchboard/internal/model"
	"pitchboard/internal/store"
)

func TestSysTimetableCSV(t *testing.T) {
	db := sampleClub()
	week := time.Date(2026, 10, 26, 0, 0, 0, 0, time.Local)
	csv := strings.Join([]string{
		"Date,Start,End,Field,Type,Zone,Home,Away",
		"2026-10-26,4:00 PM,5:00 PM,Dome Turf,Practice,Aurora,U8,",
		"2026-10-26,6:00 PM,7:15 PM,Dome Turf,Game,Eastside,Eastside U12,Lakewood U12",
		"2026-10-27,5:00 PM,6:00 PM,Dome Turf,Available,,",
		"Monday,7:00 PM,8:00 PM,Dome Turf,Practice,Zone 3,U12,",
		"2026-10-28,6:00-7:00 PM,,Dome Turf,Practice,SUSC,SUSC U10,",
	}, "\n")
	grid, err := Read("timetable.csv", []byte(csv))
	if err != nil {
		t.Fatal(err)
	}
	res, err := Commit(&db, grid, Options{Source: "Saskatoon Youth Soccer", WeekOf: week})
	if err != nil {
		t.Fatal(err)
	}
	if res.Bookings != 2 || res.Swaps != 2 || res.Open != 1 || res.Skipped != 0 {
		t.Fatalf("counts bookings=%d open=%d swaps=%d skipped=%d\n%+v", res.Bookings, res.Open, res.Swaps, res.Skipped, summaries(res))
	}
	if len(db.Events) != 2 {
		t.Fatalf("events: %+v", db.Events)
	}
	if db.Events[0].TeamIDs[0] != "u8" || db.Events[0].Start != "2026-10-26T16:00:00" || db.Events[0].PitchID != "p1" {
		t.Fatalf("u8 booking: %+v", db.Events[0])
	}
	if db.Events[1].Start != "2026-10-26T19:00:00" || db.Events[1].TeamIDs[0] != "u12" {
		t.Fatalf("weekday should land on the Monday of that week: %+v", db.Events[1])
	}
	if countKind(db.Openings, "swap") != 2 || countKind(db.Openings, "open") != 1 {
		t.Fatalf("openings: %+v", db.Openings)
	}
	for _, op := range db.Openings {
		if op.Zone == "Eastside" && !strings.Contains(op.Holder, "Eastside U12") {
			t.Fatalf("eastside holder: %+v", op)
		}
		if op.Kind == "open" && op.Use != "either" {
			t.Fatalf("available row should stay open for practice or a game: %+v", op)
		}
	}
	again, err := Commit(&db, grid, Options{Source: "Saskatoon Youth Soccer", WeekOf: week})
	if err != nil {
		t.Fatal(err)
	}
	if again.Bookings != 0 || again.Skipped < 4 {
		t.Fatalf("second import should skip rows already added: %+v", summaries(again))
	}
}

func TestXLSXSharedStringsAndSerialDate(t *testing.T) {
	db := sampleClub()
	base := time.Date(1899, 12, 30, 0, 0, 0, 0, time.UTC)
	day := time.Date(2026, 10, 28, 0, 0, 0, 0, time.UTC)
	serial := day.Sub(base).Hours() / 24
	body := [][]string{
		{"Date", "Time", "Field", "Type", "Zone", "Team"},
		{trimFloat(serial), "0.75", "Dome Turf", "Practice", "Aurora", "U16"},
	}
	grid, err := Read("master.xlsx", xlsxShared(body))
	if err != nil {
		t.Fatal(err)
	}
	res, err := Classify(&db, grid, Options{Source: "Saskatoon Youth Soccer"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Bookings != 1 || res.Rows[0].Start != "2026-10-28T18:00:00" {
		t.Fatalf("serial date and fraction time: %+v", res.Rows)
	}
}

func TestZoneNumbers(t *testing.T) {
	if canonicalZone("Zone 5") != "Eastside" || canonicalZone("Zone 1 - SUSC") != "SUSC" {
		t.Fatal(canonicalZone("Zone 5"), canonicalZone("Zone 1 - SUSC"))
	}
	if zoneIsOurs("Aurora", "Aurora", "Aurora Soccer Club") != true || zoneIsOurs("Eastside", "Aurora", "Aurora Soccer Club") != false {
		t.Fatal("zone match")
	}
}

func sampleClub() model.Database {
	when := time.Date(2026, 10, 26, 9, 0, 0, 0, time.Local)
	db := model.Database{Settings: model.DefaultSettings(when)}
	store.EnsureCatalogs(&db)
	db.Settings.ClubName = "Aurora Soccer Club"
	db.Settings.ZoneName = "Aurora"
	db.Venues = []model.Venue{{ID: "v1", Name: "Dome", OpenTime: "08:00", CloseTime: "22:00", Days: []int{0, 1, 2, 3, 4, 5, 6}, Active: true}}
	db.Pitches = []model.Pitch{{
		ID: "p1", VenueID: "v1", Name: "Turf", Environment: "indoor", SurfaceID: "surf_indoor_turf", FormatID: "fmt_5",
		Days: []int{0, 1, 2, 3, 4, 5, 6}, Active: true,
	}}
	db.Teams = []model.Team{
		{ID: "u8", Name: "U8", ShortName: "U8", AgeGroupID: "ag_u8", Active: true},
		{ID: "u12", Name: "U12", ShortName: "U12", AgeGroupID: "ag_u12", Active: true},
		{ID: "u16", Name: "U16", ShortName: "U16", AgeGroupID: "ag_u16", Active: true},
	}
	db.Normalize()
	return db
}

func xlsxShared(rows [][]string) []byte {
	var shared []string
	index := map[string]int{}
	var sheet strings.Builder
	sheet.WriteString(`<?xml version="1.0" encoding="UTF-8"?><worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData>`)
	for r, row := range rows {
		sheet.WriteString(`<row r="` + itoa(r+1) + `">`)
		for c, value := range row {
			ref := string(rune('A'+c)) + itoa(r+1)
			if r == 0 {
				n, ok := index[value]
				if !ok {
					n = len(shared)
					shared = append(shared, value)
					index[value] = n
				}
				sheet.WriteString(`<c r="` + ref + `" t="s"><v>` + itoa(n) + `</v></c>`)
				continue
			}
			sheet.WriteString(`<c r="` + ref + `"><v>` + value + `</v></c>`)
		}
		sheet.WriteString(`</row>`)
	}
	sheet.WriteString(`</sheetData></worksheet>`)
	var sst strings.Builder
	sst.WriteString(`<?xml version="1.0" encoding="UTF-8"?><sst xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">`)
	for _, s := range shared {
		sst.WriteString(`<si><t>` + s + `</t></si>`)
	}
	sst.WriteString(`</sst>`)
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	write := func(name, body string) {
		w, _ := zw.Create(name)
		_, _ = w.Write([]byte(body))
	}
	write("[Content_Types].xml", `<?xml version="1.0"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"></Types>`)
	write("xl/sharedStrings.xml", sst.String())
	write("xl/worksheets/sheet1.xml", sheet.String())
	write("xl/workbook.xml", `<?xml version="1.0"?><workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets><sheet name="Master" sheetId="1" r:id="rId1"/></sheets></workbook>`)
	write("xl/_rels/workbook.xml.rels", `<?xml version="1.0"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/></Relationships>`)
	_ = zw.Close()
	return buf.Bytes()
}

func summaries(res Result) []string {
	out := make([]string, len(res.Rows))
	for i, row := range res.Rows {
		out[i] = row.Kind + " " + row.Summary + " " + row.Problem
	}
	return out
}

func countKind(rows []model.Opening, kind string) int {
	n := 0
	for _, row := range rows {
		if row.Kind == kind {
			n++
		}
	}
	return n
}

func trimFloat(n float64) string {
	return strconv.FormatFloat(n, 'f', -1, 64)
}

func itoa(n int) string { return strconv.Itoa(n) }
