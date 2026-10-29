package export

import (
	"bytes"
	"encoding/csv"
	"strings"

	"pitchboard/internal/model"
)

func CSV(db *model.Database, rows []model.EventView) ([]byte, error) {
	var buf bytes.Buffer
	buf.Write([]byte{0xEF, 0xBB, 0xBF})
	w := csv.NewWriter(&buf)
	header := []string{
		"Date", "Start", "End", "All day", "Type", "Category", "Title", "Teams", "Age groups",
		"Coaches", "Venue", "Pitch", "Environment", "Surface", "Pitch type", "Home or away",
		"Opponent", "Away location", "Status", "Priority", "Notes",
	}
	if err := w.Write(header); err != nil {
		return nil, err
	}
	for _, row := range rows {
		start, _ := model.ParseFlexible(row.Start)
		end, _ := model.ParseFlexible(row.End)
		allDay := "no"
		if row.AllDay {
			allDay = "yes"
		}
		rec := []string{
			start.Format("2006-01-02"),
			model.FormatWhen(start, db.Settings.TimeFormat),
			model.FormatWhen(end, db.Settings.TimeFormat),
			allDay,
			row.TypeName,
			row.Category,
			row.Title,
			join(row.TeamNames),
			join(row.AgeLabels),
			join(row.CoachNames),
			row.VenueName,
			row.PitchName,
			row.EnvironmentResolved,
			row.SurfaceName,
			row.FormatName,
			row.HomeAway,
			row.Opponent,
			row.AwayLocation,
			row.Status,
			priorityLabel(row),
			row.Notes,
		}
		if err := w.Write(rec); err != nil {
			return nil, err
		}
	}
	w.Flush()
	return buf.Bytes(), w.Error()
}

func join(xs []string) string {
	out := make([]string, 0, len(xs))
	for _, s := range xs {
		if strings.TrimSpace(s) != "" {
			out = append(out, strings.TrimSpace(s))
		}
	}
	return strings.Join(out, "; ")
}
