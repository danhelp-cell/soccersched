package export

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/go-pdf/fpdf"
	"pitchboard/internal/model"
)

func PDF(db *model.Database, rows []model.EventView, from, to time.Time) ([]byte, error) {
	pdf := fpdf.New("P", "mm", "Letter", "")
	pdf.SetMargins(14, 22, 14)
	pdf.SetAutoPageBreak(true, 16)
	pdf.AddUTF8FontFromBytes("DejaVu", "", fontRegular)
	pdf.AddUTF8FontFromBytes("DejaVu", "B", fontBold)
	club := db.Settings.ClubName
	if club == "" {
		club = "Soccer club"
	}
	pdf.SetHeaderFunc(func() {
		pdf.SetFillColor(23, 53, 40)
		pdf.Rect(0, 0, 216, 16, "F")
		pdf.SetFont("DejaVu", "B", 13)
		pdf.SetTextColor(244, 241, 234)
		pdf.SetXY(14, 4)
		pdf.CellFormat(140, 8, "Pitchboard", "", 0, "L", false, 0, "")
		pdf.SetFont("DejaVu", "", 10)
		pdf.CellFormat(0, 8, club, "", 0, "R", false, 0, "")
	})
	pdf.SetFooterFunc(func() {
		pdf.SetY(-12)
		pdf.SetFont("DejaVu", "", 8)
		pdf.SetTextColor(110, 102, 92)
		pdf.CellFormat(0, 6, fmt.Sprintf("Times are %s  ·  page %d", db.Settings.TimezoneLabel, pdf.PageNo()), "", 0, "L", false, 0, "")
	})
	pdf.AddPage()
	pdf.SetTextColor(28, 25, 21)
	pdf.SetFont("DejaVu", "B", 12)
	pdf.CellFormat(0, 7, "Schedule", "", 1, "L", false, 0, "")
	pdf.SetFont("DejaVu", "", 9)
	pdf.SetTextColor(70, 64, 56)
	sub := db.Settings.SeasonName
	if !from.IsZero() && !to.IsZero() {
		last := to.Add(-time.Second)
		sub = strings.TrimSpace(sub + "   " + from.Format("Jan 2, 2006") + " – " + last.Format("Jan 2, 2006"))
	}
	early := pretty(db.Settings.EarlyWindowStart, db.Settings.TimeFormat) + "–" + pretty(db.Settings.EarlyWindowEnd, db.Settings.TimeFormat)
	pdf.CellFormat(0, 5, sub, "", 1, "L", false, 0, "")
	pdf.CellFormat(0, 5, fmt.Sprintf("Players under %d are scheduled first between %s.", db.Settings.PriorityCutoffAge, early), "", 1, "L", false, 0, "")
	pdf.Ln(2)
	if len(rows) == 0 {
		pdf.SetTextColor(28, 25, 21)
		pdf.SetFont("DejaVu", "", 11)
		pdf.CellFormat(0, 8, "Nothing is scheduled in this range.", "", 1, "L", false, 0, "")
	}
	current := ""
	for _, row := range rows {
		start, _ := model.ParseFlexible(row.Start)
		end, _ := model.ParseFlexible(row.End)
		day := start.Format("Monday, January 2")
		if day != current {
			current = day
			pdf.Ln(1)
			pdf.SetFillColor(239, 234, 226)
			pdf.SetTextColor(23, 37, 32)
			pdf.SetFont("DejaVu", "B", 11)
			pdf.CellFormat(0, 8, day, "", 1, "L", true, 0, "")
		}
		when := model.FormatWhen(start, db.Settings.TimeFormat) + "–" + model.FormatWhen(end, db.Settings.TimeFormat)
		if row.AllDay {
			when = "All day"
		}
		pdf.SetTextColor(28, 25, 21)
		pdf.SetFont("DejaVu", "B", 10)
		title := row.Title
		if row.Status == "cancelled" {
			title += " (cancelled)"
		} else if row.Status == "tentative" {
			title += " (tentative)"
		}
		pdf.MultiCell(0, 5, when+"   "+title, "", "L", false)
		pdf.SetFont("DejaVu", "", 9)
		pdf.SetTextColor(60, 54, 46)
		meta := join([]string{row.TypeName, join(row.TeamNames), row.LocationLabel, row.PlaceDetail})
		if meta != "" {
			pdf.MultiCell(0, 4.5, meta, "", "L", false)
		}
		if row.LatePriority {
			pdf.SetTextColor(138, 90, 12)
			pdf.MultiCell(0, 4.5, "Starts after the early window.", "", "L", false)
		}
		if strings.TrimSpace(row.Notes) != "" {
			pdf.SetTextColor(90, 84, 74)
			pdf.MultiCell(0, 4.5, row.Notes, "", "L", false)
		}
		pdf.Ln(1.5)
	}
	if pdf.Err() {
		return nil, pdf.Error()
	}
	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func pretty(clock, format string) string {
	mins, ok := model.ClockMinutes(clock)
	if !ok {
		return clock
	}
	t := time.Date(2000, 1, 1, mins/60, mins%60, 0, 0, time.Local)
	return model.FormatWhen(t, format)
}
