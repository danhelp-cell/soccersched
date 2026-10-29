package export

import (
	"archive/zip"
	"bytes"
	"fmt"
	"strings"

	"pitchboard/internal/model"
)

type cell struct {
	value string
	style int
}

type sheet struct {
	name string
	cols []float64
	rows [][]cell
}

func XLSX(db *model.Database, rows []model.EventView) ([]byte, error) {
	schedule := sheet{
		name: "Schedule",
		cols: []float64{14, 12, 12, 14, 16, 28, 22, 16, 28, 18, 14, 18, 14, 14, 16, 22, 28},
		rows: [][]cell{headerRow([]string{
			"Date", "Start", "End", "Type", "Title", "Teams", "Age groups", "Venue", "Pitch",
			"Environment", "Surface", "Pitch type", "Home or away", "Opponent", "Status", "Priority", "Notes",
		})},
	}
	for _, row := range rows {
		start, _ := model.ParseFlexible(row.Start)
		end, _ := model.ParseFlexible(row.End)
		style := 0
		if row.LatePriority {
			style = 2
		}
		schedule.rows = append(schedule.rows, styled([]string{
			start.Format("2006-01-02"),
			model.FormatWhen(start, db.Settings.TimeFormat),
			model.FormatWhen(end, db.Settings.TimeFormat),
			row.TypeName,
			row.Title,
			join(row.TeamNames),
			join(row.AgeLabels),
			row.VenueName,
			row.PitchName,
			row.EnvironmentResolved,
			row.SurfaceName,
			row.FormatName,
			row.HomeAway,
			row.Opponent,
			row.Status,
			priorityLabel(row),
			row.Notes,
		}, style))
	}

	teams := sheet{
		name: "Teams",
		cols: []float64{24, 16, 12, 14, 22, 18, 32, 28},
		rows: [][]cell{headerRow([]string{"Team", "Age group", "Max age", "Early priority", "Coach", "Phone", "Email", "Notes"})},
	}
	for _, t := range db.Teams {
		ag, _ := db.FindAgeGroup(t.AgeGroupID)
		early := "No"
		if db.TeamIsPriority(t) {
			early = "Yes"
		}
		teams.rows = append(teams.rows, styled([]string{t.Name, ag.Name, fmt.Sprintf("%d", ag.MaxAge), early, t.Coach, t.CoachPhone, t.CoachEmail, t.Notes}, 0))
	}

	places := sheet{
		name: "Places",
		cols: []float64{28, 32, 18, 14, 20, 14, 16, 28},
		rows: [][]cell{headerRow([]string{"Venue", "Address", "Pitch", "Environment", "Surface", "Pitch type", "Hours", "Notes"})},
	}
	for _, p := range db.Pitches {
		v, _ := db.FindVenue(p.VenueID)
		s, _ := db.FindSurface(p.SurfaceID)
		f, _ := db.FindFormat(p.FormatID)
		env := p.Environment
		if env == "indoor" {
			env = "Indoor"
		} else if env == "outdoor" {
			env = "Outdoor"
		}
		open, closeAt := p.OpenTime, p.CloseTime
		if open == "" {
			open = v.OpenTime
		}
		if closeAt == "" {
			closeAt = v.CloseTime
		}
		places.rows = append(places.rows, styled([]string{v.Name, v.Address, p.Name, env, s.Name, f.Name, open + "–" + closeAt, strings.TrimSpace(v.Notes + " " + p.Notes)}, 0))
	}
	return writeXLSX([]sheet{schedule, teams, places})
}

func headerRow(values []string) []cell {
	return styled(values, 1)
}

func styled(values []string, style int) []cell {
	out := make([]cell, len(values))
	for i, v := range values {
		out[i] = cell{value: v, style: style}
	}
	return out
}

func writeXLSX(sheets []sheet) ([]byte, error) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	files := map[string]string{}
	files["[Content_Types].xml"] = contentTypes(len(sheets))
	files["_rels/.rels"] = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/></Relationships>`
	files["xl/workbook.xml"] = workbookXML(sheets)
	files["xl/_rels/workbook.xml.rels"] = workbookRels(len(sheets))
	files["xl/styles.xml"] = stylesXML()
	for i, sh := range sheets {
		files[fmt.Sprintf("xl/worksheets/sheet%d.xml", i+1)] = sheetXML(sh)
	}
	order := []string{"[Content_Types].xml", "_rels/.rels", "xl/workbook.xml", "xl/_rels/workbook.xml.rels", "xl/styles.xml"}
	for i := range sheets {
		order = append(order, fmt.Sprintf("xl/worksheets/sheet%d.xml", i+1))
	}
	for _, name := range order {
		w, err := zw.Create(name)
		if err != nil {
			return nil, err
		}
		if _, err := w.Write([]byte(files[name])); err != nil {
			return nil, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func contentTypes(n int) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`)
	b.WriteString(`<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">`)
	b.WriteString(`<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>`)
	b.WriteString(`<Default Extension="xml" ContentType="application/xml"/>`)
	b.WriteString(`<Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/>`)
	b.WriteString(`<Override PartName="/xl/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.styles+xml"/>`)
	for i := 1; i <= n; i++ {
		fmt.Fprintf(&b, `<Override PartName="/xl/worksheets/sheet%d.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>`, i)
	}
	b.WriteString(`</Types>`)
	return b.String()
}

func workbookXML(sheets []sheet) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`)
	b.WriteString(`<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets>`)
	for i, sh := range sheets {
		fmt.Fprintf(&b, `<sheet name="%s" sheetId="%d" r:id="rId%d"/>`, xmlEscape(sheetName(sh.name)), i+1, i+1)
	}
	b.WriteString(`</sheets></workbook>`)
	return b.String()
}

func workbookRels(n int) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`)
	b.WriteString(`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">`)
	for i := 1; i <= n; i++ {
		fmt.Fprintf(&b, `<Relationship Id="rId%d" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet%d.xml"/>`, i, i)
	}
	fmt.Fprintf(&b, `<Relationship Id="rId%d" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/>`, n+1)
	b.WriteString(`</Relationships>`)
	return b.String()
}

func stylesXML() string {
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
		`<styleSheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">` +
		`<fonts count="2"><font><sz val="11"/><name val="Calibri"/></font><font><b/><sz val="11"/><color rgb="FFF4F1EA"/><name val="Calibri"/></font></fonts>` +
		`<fills count="4"><fill><patternFill patternType="none"/></fill><fill><patternFill patternType="gray125"/></fill>` +
		`<fill><patternFill patternType="solid"><fgColor rgb="FF173528"/></patternFill></fill>` +
		`<fill><patternFill patternType="solid"><fgColor rgb="FFF3E2B8"/></patternFill></fill></fills>` +
		`<borders count="1"><border><left/><right/><top/><bottom/><diagonal/></border></borders>` +
		`<cellStyleXfs count="1"><xf numFmtId="0" fontId="0" fillId="0" borderId="0"/></cellStyleXfs>` +
		`<cellXfs count="3">` +
		`<xf numFmtId="0" fontId="0" fillId="0" borderId="0" xfId="0"/>` +
		`<xf numFmtId="0" fontId="1" fillId="2" borderId="0" xfId="0" applyFont="1" applyFill="1"/>` +
		`<xf numFmtId="0" fontId="0" fillId="3" borderId="0" xfId="0" applyFill="1"/>` +
		`</cellXfs></styleSheet>`
}

func sheetXML(sh sheet) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`)
	b.WriteString(`<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">`)
	if len(sh.rows) > 0 && len(sh.rows[0]) > 0 {
		last := fmt.Sprintf("%s%d", colName(len(sh.rows[0])), len(sh.rows))
		fmt.Fprintf(&b, `<dimension ref="A1:%s"/>`, last)
		b.WriteString(`<sheetViews><sheetView workbookViewId="0"><pane ySplit="1" topLeftCell="A2" activePane="bottomLeft" state="frozen"/></sheetView></sheetViews>`)
	}
	if len(sh.cols) > 0 {
		b.WriteString(`<cols>`)
		for i, w := range sh.cols {
			fmt.Fprintf(&b, `<col min="%d" max="%d" width="%.1f" customWidth="1"/>`, i+1, i+1, w)
		}
		b.WriteString(`</cols>`)
	}
	b.WriteString(`<sheetData>`)
	for r, row := range sh.rows {
		fmt.Fprintf(&b, `<row r="%d">`, r+1)
		for c, cell := range row {
			ref := fmt.Sprintf("%s%d", colName(c+1), r+1)
			fmt.Fprintf(&b, `<c r="%s" t="inlineStr" s="%d"><is><t xml:space="preserve">%s</t></is></c>`, ref, cell.style, xmlEscape(cell.value))
		}
		b.WriteString(`</row>`)
	}
	b.WriteString(`</sheetData>`)
	if len(sh.rows) > 0 && len(sh.rows[0]) > 0 {
		last := fmt.Sprintf("%s%d", colName(len(sh.rows[0])), len(sh.rows))
		fmt.Fprintf(&b, `<autoFilter ref="A1:%s"/>`, last)
	}
	b.WriteString(`</worksheet>`)
	return b.String()
}

func sheetName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "Sheet"
	}
	var b strings.Builder
	for _, r := range name {
		switch r {
		case ':', '\\', '/', '?', '*', '[', ']':
			continue
		default:
			b.WriteRune(r)
		}
	}
	out := b.String()
	if len(out) > 31 {
		out = out[:31]
	}
	if out == "" {
		return "Sheet"
	}
	return out
}

func colName(n int) string {
	s := ""
	for n > 0 {
		n--
		s = string(rune('A'+n%26)) + s
		n /= 26
	}
	return s
}
