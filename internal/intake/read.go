package intake

import (
	"archive/zip"
	"bytes"
	"encoding/csv"
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"path"
	"strings"
	"unicode/utf8"
)

// Read turns a CSV or XLSX upload into rows of cells. The first returned row is the header.
func Read(name string, data []byte) ([][]string, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("that file is empty")
	}
	kind := strings.ToLower(path.Ext(name))
	if len(data) >= 4 && string(data[:4]) == "PK\x03\x04" {
		kind = ".xlsx"
	}
	switch kind {
	case ".csv", ".txt", "":
		return readCSV(data)
	case ".xlsx":
		return readXLSX(data)
	case ".xls":
		return nil, fmt.Errorf("save the sheet as .xlsx or .csv — older Excel files are not read")
	default:
		return nil, fmt.Errorf("use a .csv or .xlsx file")
	}
}

func readCSV(data []byte) ([][]string, error) {
	data = bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})
	if !utf8.Valid(data) {
		return nil, fmt.Errorf("the CSV needs to be UTF-8 text")
	}
	comma := ','
	if bytes.Count(firstLine(data), []byte{';'}) > bytes.Count(firstLine(data), []byte{','}) {
		comma = ';'
	}
	r := csv.NewReader(bytes.NewReader(data))
	r.Comma = comma
	r.FieldsPerRecord = -1
	r.LazyQuotes = true
	rows, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("the CSV could not be read")
	}
	return trimGrid(rows), nil
}

func firstLine(data []byte) []byte {
	if i := bytes.IndexByte(data, '\n'); i >= 0 {
		return data[:i]
	}
	return data
}

func readXLSX(data []byte) ([][]string, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("that Excel file could not be opened")
	}
	files := map[string]*zip.File{}
	for _, f := range zr.File {
		files[f.Name] = f
	}
	shared, err := sharedStrings(files)
	if err != nil {
		return nil, err
	}
	sheetPath := firstSheet(files)
	if sheetPath == "" {
		return nil, fmt.Errorf("that Excel file has no sheet")
	}
	raw, err := unzip(files[sheetPath])
	if err != nil {
		return nil, fmt.Errorf("that Excel sheet could not be read")
	}
	grid, err := sheetGrid(raw, shared)
	if err != nil {
		return nil, err
	}
	if len(trimGrid(grid)) == 0 {
		return nil, fmt.Errorf("that Excel sheet is empty")
	}
	return trimGrid(grid), nil
}

func sharedStrings(files map[string]*zip.File) ([]string, error) {
	f := files["xl/sharedStrings.xml"]
	if f == nil {
		return nil, nil
	}
	raw, err := unzip(f)
	if err != nil {
		return nil, fmt.Errorf("that Excel file could not be read")
	}
	dec := xml.NewDecoder(bytes.NewReader(raw))
	var out []string
	var cur strings.Builder
	inSI, inT := false, false
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("that Excel file could not be read")
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if t.Name.Local == "si" {
				inSI = true
				cur.Reset()
			}
			if inSI && t.Name.Local == "t" {
				inT = true
			}
		case xml.EndElement:
			if t.Name.Local == "t" {
				inT = false
			}
			if t.Name.Local == "si" && inSI {
				out = append(out, strings.TrimSpace(cur.String()))
				inSI = false
			}
		case xml.CharData:
			if inT {
				cur.WriteString(string(t))
			}
		}
	}
	return out, nil
}

func firstSheet(files map[string]*zip.File) string {
	if f := files["xl/workbook.xml"]; f != nil {
		if raw, err := unzip(f); err == nil {
			dec := xml.NewDecoder(bytes.NewReader(raw))
			var ids []string
			for {
				tok, err := dec.Token()
				if err != nil {
					break
				}
				start, ok := tok.(xml.StartElement)
				if !ok || start.Name.Local != "sheet" {
					continue
				}
				for _, attr := range start.Attr {
					if attr.Name.Local == "id" {
						ids = append(ids, attr.Value)
					}
				}
			}
			if rel := files["xl/_rels/workbook.xml.rels"]; rel != nil && len(ids) > 0 {
				if raw, err := unzip(rel); err == nil {
					targets := relTargets(raw)
					for _, id := range ids {
						if target := targets[id]; target != "" {
							name := path.Clean("xl/" + strings.TrimPrefix(target, "/"))
							if strings.HasPrefix(target, "/") {
								name = path.Clean(strings.TrimPrefix(target, "/"))
							}
							if files[name] != nil {
								return name
							}
						}
					}
				}
			}
		}
	}
	for i := 1; i <= 8; i++ {
		name := fmt.Sprintf("xl/worksheets/sheet%d.xml", i)
		if files[name] != nil {
			return name
		}
	}
	return ""
}

func relTargets(raw []byte) map[string]string {
	out := map[string]string{}
	dec := xml.NewDecoder(bytes.NewReader(raw))
	for {
		tok, err := dec.Token()
		if err != nil {
			return out
		}
		start, ok := tok.(xml.StartElement)
		if !ok || start.Name.Local != "Relationship" {
			continue
		}
		var id, target, typ string
		for _, attr := range start.Attr {
			switch attr.Name.Local {
			case "Id":
				id = attr.Value
			case "Target":
				target = attr.Value
			case "Type":
				typ = attr.Value
			}
		}
		if strings.Contains(typ, "worksheet") && id != "" {
			out[id] = target
		}
	}
}

func sheetGrid(raw []byte, shared []string) ([][]string, error) {
	dec := xml.NewDecoder(bytes.NewReader(raw))
	type cell struct {
		col, row int
		value    string
	}
	var cells []cell
	maxR, maxC := 0, 0
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("that Excel sheet could not be read")
		}
		start, ok := tok.(xml.StartElement)
		if !ok || start.Name.Local != "c" {
			continue
		}
		var ref, typ string
		for _, attr := range start.Attr {
			switch attr.Name.Local {
			case "r":
				ref = attr.Value
			case "t":
				typ = attr.Value
			}
		}
		col, row := cellRef(ref)
		inner, err := innerXML(dec, start.Name.Local)
		if err != nil {
			return nil, fmt.Errorf("that Excel sheet could not be read")
		}
		value := cellText(typ, inner, shared)
		if strings.TrimSpace(value) == "" || row <= 0 || col < 0 {
			continue
		}
		cells = append(cells, cell{col: col, row: row, value: strings.TrimSpace(value)})
		if row > maxR {
			maxR = row
		}
		if col > maxC {
			maxC = col
		}
	}
	if maxR == 0 {
		return nil, nil
	}
	if maxR > 500 {
		return nil, fmt.Errorf("a timetable can have at most 500 rows")
	}
	grid := make([][]string, maxR)
	for i := range grid {
		grid[i] = make([]string, maxC+1)
	}
	for _, c := range cells {
		grid[c.row-1][c.col] = c.value
	}
	return grid, nil
}

func innerXML(dec *xml.Decoder, local string) (string, error) {
	var buf bytes.Buffer
	enc := xml.NewEncoder(&buf)
	depth := 1
	for depth > 0 {
		tok, err := dec.Token()
		if err != nil {
			return "", err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if t.Name.Local == local {
				depth++
			}
		case xml.EndElement:
			if t.Name.Local == local {
				depth--
				if depth == 0 {
					continue
				}
			}
		}
		if depth > 0 {
			if err := enc.EncodeToken(tok); err != nil {
				return "", err
			}
		}
	}
	if err := enc.Flush(); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func cellText(typ, inner string, shared []string) string {
	switch typ {
	case "s":
		n := atoi(plainValue(inner))
		if n >= 0 && n < len(shared) {
			return shared[n]
		}
		return ""
	case "inlineStr":
		return html.UnescapeString(joinTags(inner))
	default:
		if text := joinTags(inner); typ == "str" || text != "" && plainValue(inner) == "" {
			return html.UnescapeString(text)
		}
		return html.UnescapeString(plainValue(inner))
	}
}

func plainValue(inner string) string {
	dec := xml.NewDecoder(strings.NewReader("<c>" + inner + "</c>"))
	for {
		tok, err := dec.Token()
		if err != nil {
			return ""
		}
		start, ok := tok.(xml.StartElement)
		if !ok || start.Name.Local != "v" {
			continue
		}
		var text string
		if err := dec.DecodeElement(&text, &start); err != nil {
			return ""
		}
		return strings.TrimSpace(text)
	}
}

func joinTags(inner string) string {
	dec := xml.NewDecoder(strings.NewReader("<c>" + inner + "</c>"))
	var b strings.Builder
	capture := false
	for {
		tok, err := dec.Token()
		if err != nil {
			return strings.TrimSpace(b.String())
		}
		switch t := tok.(type) {
		case xml.StartElement:
			capture = t.Name.Local == "t"
		case xml.EndElement:
			if t.Name.Local == "t" {
				capture = false
			}
		case xml.CharData:
			if capture {
				b.WriteString(string(t))
			}
		}
	}
}

func cellRef(ref string) (col, row int) {
	col, row = -1, 0
	for _, r := range ref {
		switch {
		case r >= 'A' && r <= 'Z':
			if col < 0 {
				col = 0
			}
			col = col*26 + int(r-'A'+1)
		case r >= 'a' && r <= 'z':
			if col < 0 {
				col = 0
			}
			col = col*26 + int(r-'a'+1)
		case r >= '0' && r <= '9':
			row = row*10 + int(r-'0')
		}
	}
	if col > 0 {
		col--
	}
	return col, row
}

func atoi(s string) int {
	n := 0
	for _, r := range strings.TrimSpace(s) {
		if r < '0' || r > '9' {
			return -1
		}
		n = n*10 + int(r-'0')
	}
	return n
}

func unzip(f *zip.File) ([]byte, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return io.ReadAll(io.LimitReader(rc, 8<<20))
}

func trimGrid(rows [][]string) [][]string {
	var out [][]string
	for _, row := range rows {
		blank := true
		for _, cell := range row {
			if strings.TrimSpace(cell) != "" {
				blank = false
				break
			}
		}
		if !blank {
			out = append(out, row)
		}
	}
	return out
}
