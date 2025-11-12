package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/jmoiron/sqlx"
)

func (a *App) SetCurrentDB(name string) AppResult {
	// if name == "" {
	// 	err := errors.New("invalid name")
	// 	a.logger.Error(err.Error())
	// 	return a.newResult(
	// 		err,
	// 		nil,
	// 		nil,
	// 	)
	// }
	if _, err := a.db.Exec(
		`INSERT INTO main.current_db (id, current_db)
		VALUES (1, ?)
		ON CONFLICT (id) DO UPDATE SET current_db = excluded.current_db;`,
		name,
	); err != nil {
		a.logger.Error(err.Error())
		return a.newResult(
			err,
			nil,
			nil,
		)
	}
	return a.newResult(
		nil,
		map[string]string{"name": name},
		nil,
	)
}

func (a *App) getCurrentDB() (string, error) {
	var dbName string
	if a.db == nil {
		return "", errors.New("now db found")
	}
	if err := a.db.Get(&dbName, "SELECT current_db from main.current_db where id=1;"); err != nil {
		return "", err
	}
	return dbName, nil
}

func (a *App) GetCurrentDB() AppResult {
	name, err := a.getCurrentDB()
	return a.newResult(
		err,
		name,
		nil,
	)
}

func cleanDBName(inputName string) string {
	// 1. Get the file extension. If there is no extension, this returns an empty string "".
	ext := filepath.Ext(inputName)

	// 2. Trim the extension.
	// If ext is an empty string, strings.TrimSuffix does nothing, which is exactly what we want.
	baseName := strings.TrimSuffix(inputName, ext)

	// Optional: Add further sanitization (like to lowercase or replacing invalid chars) here
	return strings.ToLower(baseName)
}

func (a *App) storeDB(name string, path string, appCreated bool) error {
	attachQuery := fmt.Sprintf("ATTACH '%s' AS %s;", path, name)
	if _, err := a.db.Exec(attachQuery); err != nil {
		if strings.Contains(err.Error(), "already in use") {
			return nil
		}
		if appCreated {
			os.Remove(path)
		} // Clean up the created file
		return err
	}
	if _, err := a.db.Exec("INSERT OR IGNORE INTO main.dbs (name, path, root, app_created) VALUES (?,?,?,?);", name, path, a.rootPath, appCreated); err != nil {
		a.db.Exec(fmt.Sprintf("DETACH DATABASE '%s';", name))
		if appCreated {
			os.Remove(path)
		}
		return err
	}
	return nil
}
func (a *App) CreateDB(dbForm CreateDBRequest) AppResult {
	a.logger.Debug(fmt.Sprint(dbForm))
	if dbForm.Name == "" {
		err := errors.New("invalid request. db name is required")
		a.logger.Error(err.Error())
		return a.newResult(err, map[string]any{"error": BadRequestError}, nil)
	}
	dbForm.Name = cleanDBName(dbForm.Name)
	// 1. MANDATORY: Check for DB Name Uniqueness in Application Metadata
	// The name used in the app/ATTACH command must be unique for the user.
	var count int
	if err := a.db.Get(&count, "SELECT COUNT(*) FROM main.dbs WHERE name = ? AND root = ?;", dbForm.Name, a.rootPath); err != nil {
		a.logger.Error(err.Error())
		return a.newResult(err, nil, nil)
	}

	if count > 0 {
		// If the name is already used in the app, we must fail.
		err := fmt.Errorf("database by that name already exists")
		a.logger.Error(err.Error())
		return a.newResult(err, nil, nil)
	}

	// --- MODIFICATION START: Ensure unique PHYSICAL file path ---
	basePath := a.getDBPath(dbForm.Name) // e.g., /path/to/my_db.db
	dbPath := basePath
	suffix := 0
	const maxAttempts = 100 // Safety break

	// Split the name from the extension for easy suffixing (requires "path/filepath")
	ext := filepath.Ext(basePath)
	nameWithoutExt := basePath[:len(basePath)-len(ext)]

	for i := range maxAttempts {
		// Check if the file at the current path exists on disk
		if _, err := os.Stat(dbPath); errors.Is(err, os.ErrNotExist) {
			// File does not exist, path is unique. Break and proceed.
			break
		}

		// File exists, so generate the next suffixed path (e.g., /path/to/my_db_1.db)
		suffix++
		dbPath = fmt.Sprintf("%s_%d%s", nameWithoutExt, suffix, ext)
		a.logger.Debug(fmt.Sprintf("File collision detected. Trying new path: %s", dbPath))

		if i == maxAttempts-1 {
			err := fmt.Errorf("failed to find a unique file path for database: %s after 100 attempts", dbForm.Name)
			a.logger.Error(err.Error())
			return a.newResult(err, nil, nil)
		}
	}

	// Create the file using the unique path (dbPath)
	if err := os.WriteFile(dbPath, []byte{}, SafePermissions); err != nil {
		a.logger.Error(err.Error())
		return a.newResult(err, nil, nil)
	}

	// Attach using the unique file path (dbPath) and the user's ORIGINAL name (dbForm.Name)
	if err := a.storeDB(dbForm.Name, dbPath, true); err != nil {
		a.logger.Error(err.Error())
		return a.newResult(err, nil, nil)
	}

	if strings.ToLower(dbForm.Journal) == "wal" {
		newDB, err := sqlx.Open(SQLITE_DRIVER, dbPath)
		if err != nil {
			a.logger.Error(err.Error())
			return a.newResult(err, nil, nil)
		}
		defer newDB.Close()
		_, err = newDB.Exec("PRAGMA journal_mode=WAL;")
		if err != nil {
			a.logger.Error(err.Error())
			return a.newResult(err, nil, nil)
		}
	}

	// Return the original user-provided name
	return a.newResult(
		nil,
		map[string]any{
			"message": fmt.Sprintf("%s has been completed successfully", dbForm.Name),
			"name":    dbForm.Name, // This is the user-facing name
		},
		nil,
	)
}

type UpdateRequest struct {
	DB     string  `json:"db"`
	Table  string  `json:"table"`
	Row    [][]any `json:"row"`
	Column string  `json:"column"`
	Value  string  `json:"value"`
}

func (a *App) UpdateDB(req UpdateRequest) AppResult {
	var pks []string
	a.logger.Debug(fmt.Sprint(req))
	var tableName string
	if req.DB == "" {
		tableName = req.Table
	} else {
		tableName = fmt.Sprintf("%s.%s", req.DB, req.Table)
	}

	pkQuery := fmt.Sprintf("SELECT name FROM pragma_table_info('%s') WHERE pk <> 0;", tableName)
	if err := a.db.Select(&pks, pkQuery); err != nil {
		a.logger.Error(err.Error())
		return a.newResult(err, nil, nil)
	}
	var pk string
	var pkVal any

	if len(req.Row) < 2 {
		return a.newResult(errors.New("invalid row data"), nil, nil)
	}

	for i, col := range req.Row[0] {
		a.logger.Debug(fmt.Sprintf("%s: %v", col, req.Row[1][i]))
		colName := col.(string)
		if slices.Contains(pks, colName) {
			pk = colName
			pkVal = req.Row[1][i]
			break
		}
	}
	if pk == "" || pkVal == nil {
		if req.Row[0][0] != req.Column {
			pk = req.Row[0][0].(string)
			pkVal = req.Row[1][0]
		} else {
			pk = req.Row[0][1].(string)
			pkVal = req.Row[1][1]
		}
	}
	query := fmt.Sprintf(
		"UPDATE %s SET %s='%v' WHERE %s='%v';",
		tableName,
		req.Column,
		req.Value,
		pk,
		pkVal,
	)
	a.logger.Debug(query)
	if _, err := a.db.Exec(query); err != nil {
		a.logger.Error(err.Error())
		return a.newResult(err, nil, nil)
	}
	return a.newResult(nil, nil, nil)
}

func (a *App) RemoveDB(dbName string) AppResult {
	if dbName == "" {
		a.logger.Error("invalid db name")
		return a.newResult(errors.New("invalid db name"), map[string]any{"error": "invalid db name"}, nil)
	}
	type SQLResult struct {
		Name string `db:"name"`
		File string `db:"file"`
	}
	var sqlResult SQLResult
	if err := a.db.Get(&sqlResult, "SELECT dbs.name, dbs.file FROM pragma_database_list dbs JOIN main.dbs mdbs ON dbs.file = mdbs.path WHERE mdbs.name = ?", dbName); err != nil {
		a.logger.Error(err.Error())
		return a.newResult(err, nil, nil)
	}
	if _, err := a.db.Exec("DELETE FROM main.dbs where name = ? ;", dbName); err != nil {
		a.logger.Error(err.Error())
		return a.newResult(err, map[string]any{"error": err.Error()}, nil)
	}
	query := fmt.Sprintf("DETACH DATABASE \"%s\";", sqlResult.Name)
	if _, err := a.db.Exec(query); err != nil {
		a.logger.Error(err.Error())
		return a.newResult(err, map[string]any{"error": err.Error()}, nil)
	}
	if err := os.Remove(sqlResult.File); err != nil {
		a.logger.Error(err.Error())
		return a.newResult(err, nil, nil)
	}
	return a.newResult(nil, nil, nil)
}
