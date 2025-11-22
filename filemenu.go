package main

import (
	"archive/zip"
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"sqlitegui/models"
	"strconv"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func generateINSERTPlaceholders(count int) string {
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

func (a *App) openFolder() {
	selection, err := a.dialog.OpenDirectory(a.ctx, runtime.OpenDialogOptions{
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
	selection, err := a.dialog.OpenFile(a.ctx, runtime.OpenDialogOptions{
		Title: "Select SQLite Database to Import",
		Filters: []runtime.FileFilter{
			{DisplayName: "SQLite Files (*.db, *.sqlite)", Pattern: "*.db;*.sqlite"},
		}})

	if err != nil {
		a.logger.Error(err.Error())
		a.emit(IMPORT_DB_FAIL, err.Error())
		return
	}
	if selection == "" {
		a.emit(IMPORT_DB_FAIL, "selection cannot be empty")
		return
	}
	dbName, _ := parseFile(selection)
	if err := a.storeDB(dbName, selection, false); err != nil {
		a.logger.Error(err.Error())
		a.emit(IMPORT_DB_FAIL, err.Error())
		return
	}
	a.emit(IMPORT_DB_SUCCESS, dbName)
}

func (a *App) exportToZip(path string, ext string) error {
	switch ext {
	case ".csv":
		zipFile, err := os.Create(path)
		if err != nil {
			return err
		}
		defer zipFile.Close()

		zipWriter := zip.NewWriter(zipFile)
		defer zipWriter.Close()

		dbs, err := a.getSQLiteDBNames()
		if err != nil {
			return err
		}
		for _, dbName := range dbs {
			tableNames, err := a.getTableList(dbName)
			if err != nil {
				return err
			}
			for _, tblName := range tableNames {
				tblData, err := a.getTableData(fmt.Sprintf("%s.%s", dbName, tblName))
				if err != nil {
					return err
				}
				headers, err := a.getColumns(tblName)
				if err != nil {
					return err
				}
				zipPath := filepath.Join(dbName, tblName) + ext
				file, err := zipWriter.Create(zipPath)
				if err != nil {
					return err
				}

				// 2. Initialize the CSV Writer
				csvWriter := csv.NewWriter(file)
				defer csvWriter.Flush()

				if err := csvWriter.Write(headers); err != nil {
					return err
				}
				for _, s := range tblData {
					var record []string

					// Ensure the order of fields in the row matches the order of the headers
					for _, header := range headers {
						record = append(record, fmt.Sprint(s[header]))
					}

					// Write the record to the CSV file
					if err := csvWriter.Write(record); err != nil {
						return err
					}
				}
				csvWriter.Flush()
				if err := csvWriter.Error(); err != nil {
					return err
				}
			}
		}
	case ".json":
		zipFile, err := os.Create(path)
		if err != nil {
			return err
		}
		defer zipFile.Close()

		zipWriter := zip.NewWriter(zipFile)
		defer zipWriter.Close()

		dbs, err := a.getSQLiteDBNames()
		if err != nil {
			return err
		}
		for _, dbName := range dbs {
			tableNames, err := a.getTableList(dbName)
			if err != nil {
				return err
			}
			for _, tblName := range tableNames {
				tblData, err := a.getTableData(fmt.Sprintf("%s.%s", dbName, tblName))
				if err != nil {
					return err
				}
				zipPath := filepath.Join(dbName, tblName) + ext
				file, err := zipWriter.Create(zipPath)
				if err != nil {
					return err
				}

				// 2. Initialize the CSV Writer
				body, err := json.Marshal(tblData)
				if err != nil {
					return err
				}
				_, err = file.Write(body)
				if err != nil {
					return err
				}
			}
		}
	default:
		return errors.New("invalid format")
	}
	return nil
}

func (a *App) exportDB(format string) {

	switch format {
	case ".db":
		selection, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
			Title:           "Save Exported Data",
			DefaultFilename: "export.db",
			Filters: []runtime.FileFilter{
				{DisplayName: "DB Export file (*.db)", Pattern: "*.db;"},
			},
		})
		a.logger.Debug(fmt.Sprint(selection, err))
		// PRAGMA database_list; select * from each table and create new tables in main and export
		// copy all tables into main db and export as .db
		newDB, err := sqlx.Connect("sqlite3", selection)
		if err != nil {
			a.logger.Error(err.Error())
			a.emit(DB_EXPORT_FAIL, err.Error())
			return
		}
		defer newDB.Close()
		dbs, err := a.getStoredDBs()
		if err != nil {
			a.logger.Error(err.Error())
			a.emit(DB_EXPORT_FAIL, err.Error())
			return
		}
		for _, db := range dbs {
			tblNames, err := a.getTableList(db.Path)
			if err != nil {
				a.logger.Error(err.Error())
				a.emit(DB_EXPORT_FAIL, err.Error())
				return
			}
			for _, tblName := range tblNames {
				var colInfos []models.ColumnInfo
				if tblName == "dbs" {
					continue
				}

				if err := a.db.Select(&colInfos, fmt.Sprintf("PRAGMA %s.table_info(%s);", db.Name, tblName)); err != nil {
					a.logger.Error(err.Error())
					a.emit(DB_EXPORT_FAIL, err.Error())
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
				newName := fmt.Sprintf("%s_%s", db.Name, tblName)
				createSQL := fmt.Sprintf("CREATE TABLE %s (%s);", newName, strings.Join(columnDefs, ", "))

				placeholders := generateINSERTPlaceholders(len(columnNames))
				insertSQL := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s);", newName, strings.Join(columnNames, ", "), placeholders)

				// C. Start Transaction on the NEW DB (Critical for Atomicity)
				tx, err := newDB.Begin()
				if err != nil {
					a.logger.Error(err.Error())
					a.emit(DB_EXPORT_FAIL, err.Error())
					return
				}

				// D. Execute CREATE TABLE (DDL)
				if _, err = tx.Exec(createSQL); err != nil {
					tx.Rollback()
					a.logger.Error(err.Error())
					a.emit(DB_EXPORT_FAIL, err.Error())
					return
				}

				// E. Prepare INSERT statement (DML)
				stmt, err := tx.Prepare(insertSQL)
				if err != nil {
					tx.Rollback()
					a.logger.Error(err.Error())
					a.emit(DB_EXPORT_FAIL, err.Error())
					return
				}

				// F. Retrieve Data from Source DB
				rows, err := a.db.Queryx(fmt.Sprintf("SELECT * FROM %s.%s;", db.Name, tblName))
				if err != nil {
					tx.Rollback()
					stmt.Close()
					a.logger.Error(err.Error())
					a.emit(DB_EXPORT_FAIL, err.Error())
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
					a.emit(DB_EXPORT_FAIL, "transaction failed")
					return
				}

				// I. Commit the Transaction for this table
				if err = tx.Commit(); err != nil {
					a.logger.Error(err.Error())
					a.emit(DB_EXPORT_FAIL, err.Error())
					return
				}
				a.logger.Info(fmt.Sprintf("Successfully exported table: %s", newName))
			}
		}
		a.logger.Info("db successfully exported")
		a.emit(DB_EXPORT_SUCCESS, "Exported Successfully!")
	case ".csv":
		selection, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
			Title:           "Save Exported Data",
			DefaultFilename: "export.zip",
			Filters: []runtime.FileFilter{
				{DisplayName: "DB Export CSV file (*.zip)", Pattern: "*.zip;"},
			},
		})
		if err != nil {
			a.logger.Error(err.Error())
			a.emit(DB_EXPORT_FAIL, "db failed to export")
			return
		}

		if err = a.exportToZip(selection, format); err != nil {
			a.logger.Error(err.Error())
			a.emit(DB_EXPORT_FAIL, "db failed to export")
			return
		}

	case ".json":
		selection, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
			Title:           "Save Exported Data",
			DefaultFilename: "export.zip",
			Filters: []runtime.FileFilter{
				{DisplayName: "DB Export JSON file (*.zip)", Pattern: "*.zip;"},
			},
		})
		if err != nil {
			a.logger.Error(err.Error())
			a.emit(DB_EXPORT_FAIL, "db failed to export")
			return
		}

		if err = a.exportToZip(selection, format); err != nil {
			a.logger.Error(err.Error())
			a.emit(DB_EXPORT_FAIL, "db failed to export")
			return
		}

		// PRAGMA database_list; new folders for each db, each table is a file
	default:
		a.emit(DB_EXPORT_FAIL, "invalid")
		// PRAGMA database_list; new folders for each db, each table is a file
		// export all dbs as zip
	}
}

func (a *App) uploadDB() {
	selection, err := a.dialog.OpenFile(a.ctx, runtime.OpenDialogOptions{
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
		a.emit(DB_UPLOAD_FAIL, "selection cannot be empty")
		return
	}

	dbName, ext := parseFile(selection)
	a.logger.Debug(fmt.Sprintf("%s %s", dbName, ext))
	var dfs []*Dataframe
	var tableNames []string
	file, err := os.ReadFile(selection)
	if err != nil {
		a.logger.Error(err.Error())
		a.emit(DB_UPLOAD_FAIL, err.Error())
		return
	}
	dbPath := a.getNewDBPath(dbName)
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

			for tblName, tblData := range data {
				tableNames = append(tableNames, sqlSanitize(tblName))
				a.logger.Debug(fmt.Sprint(tblData))
				if sliceData, ok := tblData.([]any); ok {
					finalDataframe := make(Dataframe, 0, len(sliceData))
					for _, item := range sliceData {
						if rowMap, rowOk := item.(map[string]any); rowOk {
							finalDataframe = append(finalDataframe, (Series)(rowMap))
						} else {
							a.logger.Error("malformed json: array element is not an object", slog.Any("element", item))
							err := errors.New("malformed json structure: expected array of objects")
							a.emit(DB_UPLOAD_FAIL, err.Error())
							return
						}
					}
					dfs = append(dfs, &finalDataframe)

				} else {
					a.logger.Error("malformed json: expected array for table data", slog.Any("data_type", fmt.Sprintf("%T", tblData)))
					err := errors.New("malformed json structure: expected array of objects")
					a.emit(DB_UPLOAD_FAIL, err.Error())
					return
				}
			}

		case []any:
			df := Dataframe{}
			for _, el := range data {
				if dfData, ok := el.(map[string]any); ok {
					df = append(df, (Series)(dfData))
				}
			}
			dfs = append(dfs, &df)
			tableNames = append(tableNames, dbName)
			a.logger.Debug(fmt.Sprint(df, dfs, tableNames))
		default:
			a.logger.Error("malformed json")
			a.emit(DB_UPLOAD_FAIL, "malformed json")
			return
		}

	case ".csv":
		csvReader := csv.NewReader(bytes.NewReader(file))
		if csvReader == nil {
			a.logger.Error("unable to read csv")
			a.emit(DB_UPLOAD_FAIL, "unable to read csv")
			return
		}
		rows, err := csvReader.ReadAll()
		if err != nil {
			a.logger.Error("unable to read csv")
			a.emit(DB_UPLOAD_FAIL, "unable to read csv")
			return
		}

		fieldNames := rows[0]
		data := make(Dataframe, 0, len(fieldNames))
		for _, row := range rows[1:] {
			rowData := make(Series)
			for i, name := range fieldNames {
				if i < len(row) {
					cleanedName := sqlSanitize(name)
					rowData[cleanedName] = determineFieldType(row[i])
				}
			}
			data = append(data, rowData)
		}

		dfs = append(dfs, &data)
		tableNames = append(tableNames, dbName)
	case ".sql":
		db, err := sqlx.Open(SQLITE_DRIVER, dbPath)
		if err != nil {
			a.logger.Error("Error converting to db: " + err.Error())
			a.emit(DB_UPLOAD_FAIL, "error converting to db")
			return
		}
		defer db.Close()
		if _, err = db.Exec(string(file)); err != nil {
			a.logger.Error("Opening new db: " + err.Error())
			a.emit(DB_UPLOAD_FAIL, "error converting to db")
			return
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
		a.emit(DB_UPLOAD_FAIL, "error converting to db")
		return
	}
	for i, df := range dfs {
		if i < len(tableNames) {
			if err := convertToSQLite(df, db, tableNames[i]); err != nil {
				a.logger.Error("Error converting to db: " + err.Error())
				a.emit(DB_UPLOAD_FAIL, "error converting to db")
				return
			}
		}
	}
	db.Close()
	if err := a.storeDB(dbName, dbPath, true); err != nil {
		a.logger.Error(err.Error())
		a.emit(DB_UPLOAD_FAIL, err.Error())
		return
	}
	a.emit(DB_UPLOAD_SUCCESS, "DB uploaded successfully!")
}
