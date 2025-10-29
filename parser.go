package main

import (
	"regexp"
	"strings"
)

var attachDetachRegex = regexp.MustCompile(`(?i)\s*(ATTACH|DETACH)\s+DATABASE`)

func (a *App) splitQueries(input string) []string {
	// 1. Clean up and normalize whitespace
	trimmedInput := strings.TrimSpace(input)

	// 2. Split by semicolon, but ensure we don't split within quoted strings.
	// A simple split by ';' is often sufficient for a GUI, but more complex parsers exist.
	statements := strings.Split(trimmedInput, ";")

	// 3. Trim whitespace from each statement and filter out empty strings
	cleanedStatements := make([]string, 0)
	for _, s := range statements {
		s = strings.TrimSpace(s)
		if s != "" {
			cleanedStatements = append(cleanedStatements, s)
		}
	}
	return cleanedStatements
}

func (a *App) containsAttachOrDetach(query string) bool {
	return attachDetachRegex.MatchString(query)
}
