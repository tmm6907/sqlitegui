package main

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

var (
	// Regex 1: Finds common Windows/system file copy suffixes like " (1)", "( 2)", or " ( 3 )"
	// at the end of a string ($). These are replaced with an empty string.
	reDuplicate = regexp.MustCompile(`\s*\(\s*\d+\s*\)\s*$`)

	// Regex 2: Finds any character that is NOT an alphanumeric character (a-z, A-Z, 0-9) or an underscore (_).
	// These invalid characters are replaced with a single underscore.
	reInvalidChars = regexp.MustCompile(`[^a-zA-Z0-9_]+`)
)

func (a *App) importDB() {
	selection, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select SQLite Database to Import",
		Filters: []runtime.FileFilter{
			{DisplayName: "SQLite Files (*.db, *.sqlite)", Pattern: "*.db;*.sqlite"},
		}})

	if err != nil {
		a.logger.Error(err.Error())
		runtime.EventsEmit(a.ctx, "dbAttachFailed", map[string]any{"error": err.Error()})
		return
	}
	if selection == "" {
		return
	}

	a.logger.Debug("Selected file: %s", slog.String("debug", selection))

	baseName := filepath.Base(selection)
	dbNameWithExt := strings.TrimSuffix(baseName, filepath.Ext(baseName))

	sanitizedName := reDuplicate.ReplaceAllString(dbNameWithExt, "")
	dbName := reInvalidChars.ReplaceAllString(sanitizedName, "_")
	dbName = strings.ToLower(dbName)
	attachQuery := fmt.Sprintf("ATTACH '%s' AS %s;", selection, dbName)
	if _, err = a.db.Exec(attachQuery); err != nil {
		a.logger.Error(fmt.Sprintf("Failed to ATTACH database %s: %v", dbName, err))
		runtime.EventsEmit(a.ctx, "dbAttachFailed", map[string]any{"error": err.Error()})
	}

	if _, err = a.db.Exec("INSERT into dbs (name, path) VALUES (?,?);", dbName, selection); err != nil {
		a.logger.Error(fmt.Sprintf("Failed to add database %s: %v", dbName, err))
		runtime.EventsEmit(a.ctx, "dbAttachFailed", map[string]any{"error": err.Error()})
	}
	runtime.EventsEmit(a.ctx, "dbAttached", dbName)
}

func (a *App) exportDB() {

	return
}
func (a *App) uploadDB() {

	return
}
