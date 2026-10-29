package export

import (
	"archive/zip"
	"bytes"
	"strings"
	"testing"
	"time"

	"pitchboard/internal/model"
	"pitchboard/internal/store"
)

func TestExports(t *testing.T) {
	when := time.Date(2026, 10, 27, 9, 0, 0, 0, time.Local)
	db := model.Database{Settings: model.DefaultSettings(when)}
	db.Settings.ClubName = "Aurora Soccer Club"
	store.ApplySample(&db, when)
	rows := Select(&db, Query{})
	if len(rows) < 10 {
		t.Fatalf("expected a full sample week, got %d", len(rows))
	}
	csvBytes, err := CSV(&db, rows)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(csvBytes, []byte{0xEF, 0xBB, 0xBF}) {
		t.Fatal("csv should start with a UTF-8 BOM for Excel on Windows")
	}
	if !strings.Contains(string(csvBytes), "U8 Black Practice") || !strings.Contains(string(csvBytes), "Indoor hardcourt") {
		t.Fatal("csv should include the team and the surface")
	}
	xlsx, err := XLSX(&db, rows)
	if err != nil {
		t.Fatal(err)
	}
	zr, err := zip.NewReader(bytes.NewReader(xlsx), int64(len(xlsx)))
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, f := range zr.File {
		if strings.HasSuffix(f.Name, "sheet1.xml") {
			rc, err := f.Open()
			if err != nil {
				t.Fatal(err)
			}
			buf := new(bytes.Buffer)
			_, _ = buf.ReadFrom(rc)
			rc.Close()
			if strings.Contains(buf.String(), "U7 Green") {
				found = true
			}
		}
	}
	if !found {
		t.Fatal("workbook should contain the schedule sheet")
	}
	ics := string(ICS(&db, rows))
	if !strings.Contains(ics, "BEGIN:VEVENT") || !strings.Contains(ics, "SaskTel Sports Centre") {
		t.Fatal("ics should contain events and a location")
	}
	pdf, err := PDF(&db, rows, when, when.AddDate(0, 0, 7))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(pdf, []byte("%PDF")) {
		t.Fatal("pdf should be a real PDF")
	}
}
