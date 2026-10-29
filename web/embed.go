// Package web is the calendar interface, stored inside the program.
package web

import "embed"

//go:embed index.html css js vendor
var Files embed.FS
