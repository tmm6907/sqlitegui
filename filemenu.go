package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
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
	dbFileTypes    = []string{".db", ".sqlite"}
)

type ColumnInfo struct {
	Name       string  `db:"name"`
	Type       string  `db:"type"`
	NotNull    bool    `db:"notnull"`
	PK         int     `db:"pk"`
	CID        string  `db:"cid"`
	DFLT_value *string `db:"dflt_value"`
}

func generatePlaceholders(count int) string {
	s := make([]string, count)
	for i := range s {
		s[i] = "?"
	}
	return strings.Join(s, ", ")
}

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

func (a *App) openFolder() {
	selection, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Open DB Folder",
	})
	if err != nil {
		a.logger.Error(err.Error())
		a.emit(OPEN_FOLDER_FAIL, err.Error())
		return
	}
	if selection == a.rootPath {
		a.emit(OPEN_FOLDER_FAIL, "folder already selected")
		return
	}
	a.logger.Debug(selection)
	if err = a.detachDBs(); err != nil {
		a.logger.Error(err.Error())
		a.emit(OPEN_FOLDER_FAIL, err.Error())
		return
	}
	a.rootPath = selection
	if err = a.attachMainDBs(); err != nil {
		a.logger.Error(err.Error())
		a.emit(OPEN_FOLDER_FAIL, err.Error())
		return
	}
	if err = a.attachDBsFromFolder(selection); err != nil {
		a.logger.Error(err.Error())
	}
	a.emit(OPEN_FOLDER_SUCCESS, "")
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
	if err := a.storeDB(dbName, selection, false); err != nil {
		a.logger.Error(err.Error())
		runtime.EventsEmit(a.ctx, IMPORT_DB_FAIL.String(), map[string]any{"error": err.Error()})
	}
	runtime.EventsEmit(a.ctx, IMPORT_DB_FAIL.String(), dbName)
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
		newDB, err := sqlx.Connect("sqlite3", filePath)
		if err != nil {
			a.logger.Error(err.Error())
			runtime.EventsEmit(a.ctx, "dbExportFailed", map[string]any{"error": err.Error()})
			return
		}
		defer newDB.Close()
		var dbs []pragmaResult

		if err := a.db.Select(&dbs, "PRAGMA database_list;"); err != nil {
			a.logger.Error(err.Error())
			runtime.EventsEmit(a.ctx, "dbExportFailed", map[string]any{"error": err.Error()})
			return
		}
		for _, db := range dbs {
			var tblNames []string
			query := fmt.Sprintf("SELECT name from %s.sqlite_master where type='table';", db.Name)
			if err := a.db.Select(&tblNames, query); err != nil {
				a.logger.Error(err.Error())
				runtime.EventsEmit(a.ctx, "dbExportFailed", map[string]any{"error": err.Error()})
				return
			}

			for _, tblName := range tblNames {
				var colInfos []ColumnInfo
				if tblName == "dbs" {
					continue
				}
				newName := fmt.Sprintf("%s_%s", db.Name, tblName)
				if err := a.db.Select(&colInfos, fmt.Sprintf("PRAGMA %s.table_info(%s);", db.Name, tblName)); err != nil {
					a.logger.Error(err.Error())
					runtime.EventsEmit(a.ctx, "dbExportFailed", map[string]any{"error": err.Error()})
					return
				}
				if len(colInfos) == 0 {
					a.logger.Info(fmt.Sprintf("Skipping table %s.%s: no column info.", db.Name, tblName))
					continue
				}

				var columnDefs []string
				var columnNames []string
				for _, info := range colInfos {
					columnNames = append(columnNames, info.Name)
					def := fmt.Sprintf("%s %s", info.Name, info.Type)
					if info.NotNull {
						def += " NOT NULL"
					}
					if info.PK > 0 {
						def += " PRIMARY KEY"
					}
					columnDefs = append(columnDefs, def)
				}
				createSQL := fmt.Sprintf("CREATE TABLE %s (%s);", newName, strings.Join(columnDefs, ", "))

				placeholders := generatePlaceholders(len(columnNames))
				insertSQL := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s);", newName, strings.Join(columnNames, ", "), placeholders)

				// C. Start Transaction on the NEW DB (Critical for Atomicity)
				tx, err := newDB.Begin()
				if err != nil {
					a.logger.Error(err.Error())
					runtime.EventsEmit(a.ctx, "dbExportFailed", map[string]any{"error": err.Error()})
					return
				}

				// D. Execute CREATE TABLE (DDL)
				if _, err = tx.Exec(createSQL); err != nil {
					tx.Rollback()
					a.logger.Error(err.Error())
					runtime.EventsEmit(a.ctx, "dbExportFailed", map[string]any{"error": err.Error()})
					return
				}

				// E. Prepare INSERT statement (DML)
				stmt, err := tx.Prepare(insertSQL)
				if err != nil {
					tx.Rollback()
					a.logger.Error(err.Error())
					runtime.EventsEmit(a.ctx, "dbExportFailed", map[string]any{"error": err.Error()})
					return
				}

				// F. Retrieve Data from Source DB
				rows, err := a.db.Queryx(fmt.Sprintf("SELECT * FROM %s.%s;", db.Name, tblName))
				if err != nil {
					tx.Rollback()
					stmt.Close()
					a.logger.Error(err.Error())
					runtime.EventsEmit(a.ctx, "dbExportFailed", map[string]any{"error": err.Error()})
					return
				}

				// G. Loop data, Marshal complex types, and Execute Inserts
				var rollbackErr error
				for rows.Next() {
					rowMap := make(map[string]any)
					if err := rows.MapScan(rowMap); err != nil {
						a.logger.Error(fmt.Sprintf("MapScan error on %s: %s", newName, err.Error()))
						continue // Skip row if map scan fails
					}

					values := make([]any, len(columnNames))

					for i, colName := range columnNames {
						value := rowMap[colName]
						v := reflect.ValueOf(value)

						// Check if the value is a complex type (slice or map) that needs JSON serialization
						if v.IsValid() && (v.Kind() == reflect.Slice || v.Kind() == reflect.Map) {
							jsonBytes, marshalErr := json.Marshal(value)
							if marshalErr != nil {
								rollbackErr = marshalErr
								break
							}
							values[i] = string(jsonBytes)
						} else {
							values[i] = value
						}
					}

					if rollbackErr != nil {
						break
					}

					// Execute the insert for the current row
					if _, err = stmt.Exec(values...); err != nil {
						rollbackErr = err
						break
					}
				}

				// H. Cleanup and Check for Errors
				rows.Close()
				stmt.Close()

				if rollbackErr != nil {
					tx.Rollback()
					a.logger.Error("transaction failed")
					runtime.EventsEmit(a.ctx, "dbExportFailed", map[string]any{"error": "transaction failed"})
					return
				}

				// I. Commit the Transaction for this table
				if err = tx.Commit(); err != nil {
					a.logger.Error(err.Error())
					runtime.EventsEmit(a.ctx, "dbExportFailed", map[string]any{"error": err.Error()})
					return
				}
				a.logger.Info(fmt.Sprintf("Successfully exported table: %s", newName))
			}
		}
		a.logger.Info("db successfully exported")
		runtime.EventsEmit(a.ctx, "dbExportSucceeded", map[string]any{"msg": "db successfully exported"})
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
		a.emit(DB_UPLOAD_FAIL, err.Error())
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
		a.emit(DB_UPLOAD_FAIL, err.Error())
		return
	}
	dbName = cleanDBName(dbName)
	dbPath := a.getDBPath(dbName)
	switch ext {
	case ".json":
		var fileData any
		if err = json.Unmarshal(file, &fileData); err != nil {
			a.logger.Error(err.Error())
			a.emit(DB_UPLOAD_FAIL, err.Error())
			return
		}
		switch data := fileData.(type) {
		case map[string]any:
			tables := []string{}
			for tblName, tblData := range data {
				tables = append(tables, cleanTableName(tblName))
				a.logger.Debug(fmt.Sprint(tblData))
				if sliceData, ok := tblData.([]any); ok {
					var finalDataframe []map[string]any
					for _, item := range sliceData {
						if rowMap, rowOk := item.(map[string]any); rowOk {
							finalDataframe = append(finalDataframe, rowMap)
						} else {
							a.logger.Error("malformed json: array element is not an object", slog.Any("element", item))
							err := errors.New("malformed json structure: expected array of objects")
							a.emit(DB_UPLOAD_FAIL, err.Error())
							return
						}
					}
					dfs = append(dfs, finalDataframe)

				} else {
					a.logger.Error("malformed json: expected array for table data", slog.Any("data_type", fmt.Sprintf("%T", tblData)))
					err := errors.New("malformed json structure: expected array of objects")
					a.emit(DB_UPLOAD_FAIL, err.Error())
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
			err := errors.New("malformed json")
			a.emit(DB_UPLOAD_FAIL, err.Error())
			return
		}

	case ".csv":
		csvReader := csv.NewReader(bytes.NewReader(file))
		if csvReader == nil {
			a.logger.Error("unable to read csv")
			err := errors.New("unable to read csv")
			a.emit(DB_UPLOAD_FAIL, err.Error())
			return
		}
		rows, err := csvReader.ReadAll()
		if err != nil {
			a.logger.Error("unable to read csv")
			err := errors.New("unable to read csv")
			a.emit(DB_UPLOAD_FAIL, err.Error())
			return
		}

		fieldNames := rows[0]
		data := make(Dataframe, 0, len(fieldNames)-1)
		for _, row := range rows[1:] {
			rowData := make(map[string]any)
			for i, name := range fieldNames {
				if i < len(row) {
					cleanedName := cleanTableName(name)
					rowData[cleanedName] = determineFieldType(row[i])
				}
			}
			data = append(data, rowData)
		}

		dfs = append(dfs, data)
		tableNames = append(tableNames, dbName)
	case ".sql":
		db, err := sqlx.Open(SQLITE_DRIVER, dbPath)
		if err != nil {
			a.logger.Error("Error converting to db: " + err.Error())
			err := errors.New("error converting to db")
			a.emit(DB_UPLOAD_FAIL, err.Error())
			return
		}
		defer db.Close()
		if _, err = db.Exec(string(file)); err != nil {
			a.logger.Error("Opening new db: " + err.Error())
			err := errors.New("error opening db")
			a.emit(DB_UPLOAD_FAIL, err.Error())
		}

		if err := a.storeDB(dbName, dbPath, true); err != nil {
			a.logger.Error(err.Error())
			a.emit(DB_UPLOAD_FAIL, err.Error())
			return
		}
		a.emit(DB_UPLOAD_SUCCESS, "DB uploaded successfully!")
		return
	}

	db, err := sqlx.Open(SQLITE_DRIVER, dbPath)
	if err != nil {
		a.logger.Error("Error converting to db: " + err.Error())
		err := errors.New("error converting to db")
		a.emit(DB_UPLOAD_FAIL, err.Error())
		return
	}
	for i, df := range dfs {
		if i < len(tableNames) {
			if err := df.convertToSQLite(db, tableNames[i]); err != nil {
				a.logger.Error("Error converting to db: " + err.Error())
				err := errors.New("error converting to db")
				a.emit(DB_UPLOAD_FAIL, err.Error())
				return
			}
		}
	}
	db.Close()
	if err := a.storeDB(dbName, dbPath, true); err != nil {
		a.logger.Error(err.Error())
		a.emit(DB_UPLOAD_FAIL, err.Error())
	}
	a.emit(DB_UPLOAD_SUCCESS, "DB uploaded successfully!")
}
