package export

import (
	"fmt"
	"strings"
	"time"

	"pitchboard/internal/model"
)

func ICS(db *model.Database, rows []model.EventView) []byte {
	var b strings.Builder
	b.WriteString("BEGIN:VCALENDAR\r\n")
	b.WriteString("VERSION:2.0\r\n")
	b.WriteString("PRODID:-//Pitchboard//Soccer Club Scheduler//EN\r\n")
	b.WriteString("CALSCALE:GREGORIAN\r\n")
	b.WriteString("METHOD:PUBLISH\r\n")
	writeFold(&b, "X-WR-CALNAME:"+icsEscape(calendarName(db)))
	now := time.Now().UTC().Format("20060102T150405Z")
	for _, row := range rows {
		start, err1 := model.ParseFlexible(row.Start)
		end, err2 := model.ParseFlexible(row.End)
		if err1 != nil || err2 != nil {
			continue
		}
		b.WriteString("BEGIN:VEVENT\r\n")
		writeFold(&b, "UID:"+icsEscape(row.ID)+"@pitchboard.local")
		writeFold(&b, "DTSTAMP:"+now)
		if row.AllDay {
			writeFold(&b, "DTSTART;VALUE=DATE:"+start.Format("20060102"))
			writeFold(&b, "DTEND;VALUE=DATE:"+end.Format("20060102"))
		} else {
			writeFold(&b, "DTSTART:"+start.Format("20060102T150405"))
			writeFold(&b, "DTEND:"+end.Format("20060102T150405"))
		}
		writeFold(&b, "SUMMARY:"+icsEscape(row.Title))
		loc := row.LocationLabel
		if row.PlaceDetail != "" && row.HomeAway != "away" {
			if loc != "" {
				loc += " — " + row.PlaceDetail
			} else {
				loc = row.PlaceDetail
			}
		}
		if loc != "" {
			writeFold(&b, "LOCATION:"+icsEscape(loc))
		}
		desc := description(row)
		if desc != "" {
			writeFold(&b, "DESCRIPTION:"+icsEscape(desc))
		}
		if row.TypeName != "" {
			writeFold(&b, "CATEGORIES:"+icsEscape(row.TypeName))
		}
		switch row.Status {
		case "cancelled":
			writeFold(&b, "STATUS:CANCELLED")
		case "tentative":
			writeFold(&b, "STATUS:TENTATIVE")
		default:
			writeFold(&b, "STATUS:CONFIRMED")
		}
		b.WriteString("END:VEVENT\r\n")
	}
	b.WriteString("END:VCALENDAR\r\n")
	return []byte(b.String())
}

func calendarName(db *model.Database) string {
	if db.Settings.ClubName != "" {
		return db.Settings.ClubName
	}
	return "Pitchboard"
}

func description(row model.EventView) string {
	var lines []string
	if len(row.TeamNames) > 0 {
		lines = append(lines, "Teams: "+join(row.TeamNames))
	}
	if row.PlaceDetail != "" {
		lines = append(lines, row.PlaceDetail)
	}
	if row.Opponent != "" {
		lines = append(lines, "Opponent: "+row.Opponent)
	}
	if row.LatePriority {
		lines = append(lines, "Starts after the early window for players under the priority age.")
	}
	if row.Notes != "" {
		lines = append(lines, row.Notes)
	}
	return strings.Join(lines, "\n")
}

func icsEscape(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, "\r\n", `\n`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	s = strings.ReplaceAll(s, ",", `\,`)
	s = strings.ReplaceAll(s, ";", `\;`)
	return s
}

func writeFold(b *strings.Builder, line string) {
	// RFC 5545 fold at 75 octets. This text is mostly ASCII.
	for len(line) > 75 {
		fmt.Fprintf(b, "%s\r\n ", line[:75])
		line = line[75:]
	}
	b.WriteString(line)
	b.WriteString("\r\n")
}
