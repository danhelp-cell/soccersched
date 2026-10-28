package export

import "strings"

func xmlEscape(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '&':
			b.WriteString("&amp;")
		case '<':
			b.WriteString("&lt;")
		case '>':
			b.WriteString("&gt;")
		case '"':
			b.WriteString("&quot;")
		default:
			if r == '\t' || r == '\n' || r == '\r' || r >= 32 {
				b.WriteRune(r)
			}
		}
	}
	return b.String()
}
