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

func (a *App) exportDB(format string) {
	switch format {
	case ".db":
		filePath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
			Title:           "Save Exported Data",
			DefaultFilename: "export.db",
			Filters: []runtime.FileFilter{
				{DisplayName: "DB Export file (*.db)", Pattern: "*.db;"},
			},
		})
		a.logger.Debug(fmt.Sprint(filePath, err))
		// PRAGMA database_list; select * from each table and create new tables in main and export
		// copy all tables into main db and export as .db
	case ".csv":
		filePath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
			Title:           "Save Exported Data",
			DefaultFilename: "export.zip",
			Filters: []runtime.FileFilter{
				{DisplayName: "CSV Export file (*.zip)", Pattern: "*.zip;"},
			},
		})
		a.logger.Debug(fmt.Sprint(filePath, err))
		// PRAGMA database_list; new folders for each db, each table is a file

	case ".json":
		filePath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
			Title:           "Save Exported Data",
			DefaultFilename: "export.zip",
			Filters: []runtime.FileFilter{
				{DisplayName: "JSON Export file (*.zip)", Pattern: "*.zip;"},
			},
		})
		a.logger.Debug(fmt.Sprint(filePath, err))
		// PRAGMA database_list; new folders for each db, each table is a file
	case "":
		filePath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
			Title:           "Save Exported Data",
			DefaultFilename: "export.zip",
			Filters: []runtime.FileFilter{
				{DisplayName: "DB Export file (*.zip)", Pattern: "*.zip;"},
			},
		})
		a.logger.Debug(fmt.Sprint(filePath, err))
		// PRAGMA database_list; new folders for each db, each table is a file
		// export all dbs as zip
	}
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
	dfs := []Dataframe{}
	tableNames := []string{}
	file, err := os.ReadFile(selection)
	if err != nil {
		a.logger.Error(err.Error())
		runtime.EventsEmit(a.ctx, "dbUploadFailed", map[string]any{"error": err.Error()})
		return
	}
	switch ext {
	case ".json":
		var fileData any
		if err = json.Unmarshal(file, &fileData); err != nil {
			a.logger.Error(err.Error())
			runtime.EventsEmit(a.ctx, "dbUploadFailed", map[string]any{"error": err.Error()})
			return
		}
		switch data := fileData.(type) {
		case map[string]any:
			tables := []string{}
			for tblName, tblData := range data {
				tables = append(tables, tblName)
				a.logger.Debug(fmt.Sprint(tblData))
				if sliceData, ok := tblData.([]any); ok {
					var finalDataframe []map[string]any
					for _, item := range sliceData {
						if rowMap, rowOk := item.(map[string]any); rowOk {
							finalDataframe = append(finalDataframe, rowMap)
						} else {
							a.logger.Error("malformed json: array element is not an object", slog.Any("element", item))
							runtime.EventsEmit(a.ctx, "dbUploadFailed", map[string]any{"error": "malformed json structure: expected array of objects"})
							return
						}
					}
					dfs = append(dfs, finalDataframe)

				} else {
					a.logger.Error("malformed json: expected array for table data", slog.Any("data_type", fmt.Sprintf("%T", tblData)))
					runtime.EventsEmit(a.ctx, "dbUploadFailed", map[string]any{"error": "malformed json: table data not an array"})
					return
				}
			}
			tableNames = tables

		case []any:
			df := Dataframe{}
			for _, el := range data {
				if dfData, ok := el.(map[string]any); ok {
					df = append(df, dfData)
				}
				dfs = append(dfs, df)
				tableNames = append(tableNames, dbName)
			}
		default:
			a.logger.Error("malformed json")
			runtime.EventsEmit(a.ctx, "dbUploadFailed", map[string]any{"error": "malformed json"})
			return
		}

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

		dfs = append(dfs, data)
		tableNames = append(tableNames, dbName)
	case ".sql":
		dbPath := a.getDBPath(dbName)
		db, err := sqlx.Open(SQLITE_DRIVER, dbPath)
		if err != nil {
			a.logger.Error("Error converting to db: " + err.Error())
			runtime.EventsEmit(a.ctx, "dbUploadFailed", map[string]any{"error": "Error converting to db."})
			return
		}
		defer db.Close()
		if _, err = db.Exec(string(file)); err != nil {
			a.logger.Error("Opening new db: " + err.Error())
			runtime.EventsEmit(a.ctx, "dbUploadFailed", map[string]any{"error": "Opening new db."})
			return
		}
	}

	dbPath := a.getDBPath(dbName)
	db, err := sqlx.Open(SQLITE_DRIVER, dbPath)
	if err != nil {
		a.logger.Error("Error converting to db: " + err.Error())
		runtime.EventsEmit(a.ctx, "dbUploadFailed", map[string]any{"error": "Error converting to db."})
		return
	}
	for i, df := range dfs {
		if i < len(tableNames) {
			if err := df.convertToSQLite(db, tableNames[i]); err != nil {
				a.logger.Error("Error converting to db: " + err.Error())
				runtime.EventsEmit(a.ctx, "dbUploadFailed", map[string]any{"error": "Error converting to db."})
				return
			}
		}
	}
	db.Close()
	attachQuery := fmt.Sprintf("ATTACH DATABASE '%s' AS %s;", dbPath, dbName)
	_, err = a.db.Exec(attachQuery)
	if err != nil {
		a.logger.Error("Error converting to db: " + err.Error())
		runtime.EventsEmit(a.ctx, "dbUploadFailed", map[string]any{"error": "Error converting to db."})
		return
	}

	_, err = a.db.Exec("INSERT into dbs (name, path) VALUES (?,?);", dbName, dbPath)
	if err != nil {
		a.logger.Error("Error converting to db: " + err.Error())
		runtime.EventsEmit(a.ctx, "dbUploadFailed", map[string]any{"error": "Error converting to db."})
		return
	}
	a.emit("dbUploadSucceeded", "DB uploaded successfully!")
}
