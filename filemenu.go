package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/jmoiron/sqlx"
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

func determineFieldType(value string) any {
	lowerValue := strings.ToLower(value)
	if lowerValue == "true" {
		return true
	}
	if lowerValue == "false" {
		return false
	}
	if i, err := strconv.ParseInt(value, 10, 64); err == nil {
		return i
	}
	if f, err := strconv.ParseFloat(value, 64); err == nil {
		return f
	}
	return value
}

func parseFile(selection string) (string, string) {
	baseName := filepath.Base(selection)
	fileExt := filepath.Ext(baseName)
	dbNameWithExt := strings.TrimSuffix(baseName, fileExt)

	sanitizedName := reDuplicate.ReplaceAllString(dbNameWithExt, "")
	dbName := reInvalidChars.ReplaceAllString(sanitizedName, "_")
	dbName = strings.ToLower(dbName)
	return dbName, fileExt
}

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
	dbName, _ := parseFile(selection)
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
	selection, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select data file to upload.",
		Filters: []runtime.FileFilter{
			{DisplayName: "SQLite Data Files (*.csv, *.json, *.sql)", Pattern: "*.csv;*.json;*.sql;"},
		}})
	if err != nil {
		a.logger.Error(err.Error())
		runtime.EventsEmit(a.ctx, "dbUploadFailed", map[string]any{"error": err.Error()})
		return
	}
	if selection == "" {
		return
	}

	a.logger.Debug("Selected file: %s", slog.String("debug", selection))
	dbName, ext := parseFile(selection)
	a.logger.Debug(fmt.Sprintf("%s %s", dbName, ext))
	var df *Dataframe
	file, err := os.ReadFile(selection)
	if err != nil {
		a.logger.Error(err.Error())
		runtime.EventsEmit(a.ctx, "dbUploadFailed", map[string]any{"error": err.Error()})
		return
	}
	switch ext {
	case ".json":
		if err = json.Unmarshal(file, &df); err != nil {
			a.logger.Error(err.Error())
			runtime.EventsEmit(a.ctx, "dbUploadFailed", map[string]any{"error": err.Error()})
			return
		}
		a.logger.Debug("converting df", slog.Any("dataframe", df))
		if err = df.convertToDB(a.db, dbName); err != nil {
			a.logger.Error("Error converting to db: " + err.Error())
			runtime.EventsEmit(a.ctx, "dbUploadFailed", map[string]any{"error": "Error converting to db."})
			return
		}
		a.logger.Debug("Data frame: %s", slog.Any("dataframe", df))
		runtime.EventsEmit(a.ctx, "dbUploadSucceeded", map[string]any{})

	case ".csv":
		csvReader := csv.NewReader(bytes.NewReader(file))
		if csvReader == nil {
			a.logger.Error("unable to read csv")
			runtime.EventsEmit(a.ctx, "dbUploadFailed", map[string]any{"error": "unable to read csv"})
			return
		}
		rows, err := csvReader.ReadAll()
		if err != nil {
			a.logger.Error("unable to read csv")
			runtime.EventsEmit(a.ctx, "dbUploadFailed", map[string]any{"error": "unable to read csv"})
			return
		}

		fieldNames := rows[0]
		data := make(Dataframe, 0, len(fieldNames)-1)
		for _, row := range rows[1:] {
			rowData := make(map[string]any)
			for i, name := range fieldNames {
				if i < len(row) {
					rowData[name] = determineFieldType(row[i])
				}
			}
			data = append(data, rowData)
		}

		df = &data
		if err = df.convertToDB(a.db, dbName); err != nil {
			a.logger.Error("Error converting to db: " + err.Error())
			runtime.EventsEmit(a.ctx, "dbUploadFailed", map[string]any{"error": "Error converting to db."})
			return
		}
		a.logger.Debug("Data frame: %s", slog.Any("dataframe", df))
		runtime.EventsEmit(a.ctx, "dbUploadSucceeded", map[string]any{})
	case ".sql":
		dbPath := a.getDBPath(dbName)
		newDB, err := sqlx.Open("sqlite3", dbPath)
		if err != nil {
			a.logger.Error("Opening new db: " + err.Error())
			runtime.EventsEmit(a.ctx, "dbUploadFailed", map[string]any{"error": "Opening new db."})
			return
		}
		defer newDB.Close()
		if _, err = newDB.Exec(string(file)); err != nil {
			a.logger.Error("Opening new db: " + err.Error())
			runtime.EventsEmit(a.ctx, "dbUploadFailed", map[string]any{"error": "Opening new db."})
			return
		}
		attachQuery := fmt.Sprintf("ATTACH DATABASE '%s' AS %s;", dbPath, dbName)
		_, err = a.db.Exec(attachQuery)
		if err != nil {
			a.logger.Error("Executing build script: " + err.Error())
			runtime.EventsEmit(a.ctx, "dbUploadFailed", map[string]any{"error": "Executing build script."})
			return
		}

		_, err = a.db.Exec("INSERT into dbs (name, path) VALUES (?,?);", dbName, dbPath)
		if err != nil {
			a.logger.Error("adding to main db: " + err.Error())
			runtime.EventsEmit(a.ctx, "dbUploadFailed", map[string]any{"error": "adding to main db."})
			return
		}
	}
}
