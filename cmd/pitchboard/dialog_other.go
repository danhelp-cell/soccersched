//go:build !windows

package main

import "log"

func showError(title, text string) {
	log.Printf("%s: %s", title, text)
}
