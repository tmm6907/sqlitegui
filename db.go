package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jmoiron/sqlx"
)

func (a *App) SetCurrentDB(name string) Result {
	if name == "" {
		err := errors.New("invalid name")
		a.logger.Error(err.Error())
		return a.newResult(
			err,
			nil,
		)
	}
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
		)
	}
	a.logger.Debug(fmt.Sprintf("Set db to %s", name))
	return a.newResult(
		nil,
		map[string]string{"name": name},
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

func (a *App) GetCurrentDB() Result {
	name, err := a.getCurrentDB()
	return a.newResult(
		err,
		name,
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
func (a *App) CreateDB(dbForm CreateDBRequest) Result {
	a.logger.Debug(fmt.Sprint(dbForm))

	if dbForm.Name == "" {
		err := errors.New("invalid request. db name is required")
		a.logger.Error(err.Error())
		return a.newResult(err, map[string]any{"error": BadRequestError})
	}
	dbForm.Name = cleanDBName(dbForm.Name)

	// 1. MANDATORY: Check for DB Name Uniqueness in Application Metadata
	// The name used in the app/ATTACH command must be unique for the user.
	var count int
	if err := a.db.Get(&count, "SELECT COUNT(*) FROM main.dbs WHERE name = ? AND root = ?;", dbForm.Name, a.rootPath); err != nil {
		a.logger.Error(err.Error())
		return a.newResult(err, map[string]string{
			"error": InternalServerError,
		})
	}

	if count > 0 {
		// If the name is already used in the app, we must fail.
		err := fmt.Errorf("database by that name already exists")
		a.logger.Error(err.Error())
		return a.newResult(err, map[string]string{
			"error": "Database by that name already exists. Please choose a different application name.",
		})
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
			return a.newResult(err, map[string]string{
				"error": "Failed to create unique physical file path.",
			})
		}
	}
	// After the loop, dbPath holds the unique physical file path.
	// dbForm.Name holds the original user-provided application name.
	// --- MODIFICATION END ---

	// Create the file using the unique path (dbPath)
	if err := os.WriteFile(dbPath, []byte{}, SafePermissions); err != nil {
		a.logger.Error(err.Error())
		return a.newResult(
			err,
			map[string]string{
				"error": InternalServerError,
			},
		)
	}

	// Attach using the unique file path (dbPath) and the user's ORIGINAL name (dbForm.Name)
	attachQuery := fmt.Sprintf("ATTACH '%s' AS %s;", dbPath, dbForm.Name)
	if _, err := a.db.Exec(attachQuery); err != nil {
		a.logger.Error(err.Error())
		os.Remove(dbPath) // Clean up the created file
		return a.newResult(
			err,
			map[string]any{
				"error": InternalServerError,
			})
	}

	// Insert into main.dbs using the original name (dbForm.Name) and the unique path (dbPath)
	if _, err := a.db.Exec("INSERT INTO main.dbs (name, path, root, app_created) VALUES (?,?,?,?);", dbForm.Name, dbPath, a.rootPath, true); err != nil {
		a.logger.Error(err.Error())
		a.db.Exec(fmt.Sprintf("DETACH DATABASE %s;", dbForm.Name))
		os.Remove(dbPath) // Clean up the created file
		return a.newResult(
			err,
			map[string]any{
				"error": InternalServerError,
			})
	}
	a.logger.Debug(fmt.Sprintf("path: %s", dbPath))

	if strings.ToLower(dbForm.Journal) == "wal" {
		newDB, err := sqlx.Open(SQLITE_DRIVER, dbPath)
		if err != nil {
			a.logger.Error(err.Error())
			return a.newResult(
				err,
				map[string]any{
					"error": InternalServerError,
				})
		}
		defer newDB.Close()
		_, err = newDB.Exec("PRAGMA journal_mode=WAL;")
		if err != nil {
			a.logger.Error(err.Error())
			return a.newResult(
				err,
				map[string]any{
					"error": err.Error(),
				},
			)
		}
	}

	// Return the original user-provided name
	return a.newResult(
		nil,
		map[string]any{
			"message": fmt.Sprintf("%s has been completed successfully", dbForm.Name),
			"name":    dbForm.Name, // This is the user-facing name
		},
	)
}

type UpdateRequest struct {
	ID    any    `json:"id"`
	Query string `json:"query"`
	Value string `json:"value"`
}

func (a *App) UpdateDB(req UpdateRequest) Result {
	escapedValue := strings.ReplaceAll(req.Value, "'", "''")

	var idValue string
	var escapedID string

	switch v := req.ID.(type) {
	case float64:
		idValue = fmt.Sprintf("%v", int64(v))
		escapedID = idValue
	case string:
		idValue = v
		escapedID = "'" + strings.ReplaceAll(idValue, "'", "''") + "'"
	default:
		idValue = fmt.Sprintf("%v", v)
		escapedID = "'" + strings.ReplaceAll(idValue, "'", "''") + "'"
	}
	query := fmt.Sprintf(req.Query, escapedValue, escapedID)

	a.logger.Debug(query)

	if _, err := a.db.Exec(query); err != nil {
		a.logger.Error(fmt.Sprintf("Update failed on table: %s", err.Error()))
		return a.newResult(
			err,
			map[string]any{
				"error": err.Error(),
				"query": query,
			},
		)
	}
	return a.newResult(nil, nil)
}

func (a *App) RemoveDB(dbName string) Result {
	if dbName == "" {
		a.logger.Error("invalid db name")
		return a.newResult(errors.New("invalid db name"), map[string]any{"error": "invalid db name"})
	}
	type SQLResult struct {
		Name string `db:"name"`
		File string `db:"file"`
	}
	var sqlResult SQLResult
	if err := a.db.Get(&sqlResult, "SELECT dbs.name, dbs.file FROM pragma_database_list dbs JOIN main.dbs mdbs ON dbs.file = mdbs.path WHERE mdbs.name = ?", dbName); err != nil {
		a.logger.Error(err.Error())
		return a.newResult(err, nil)
	}
	if _, err := a.db.Exec("DELETE FROM main.dbs where name = ? ;", dbName); err != nil {
		a.logger.Error(err.Error())
		return a.newResult(err, map[string]any{"error": err.Error()})
	}
	query := fmt.Sprintf("DETACH DATABASE \"%s\";", sqlResult.Name)
	if _, err := a.db.Exec(query); err != nil {
		a.logger.Error(err.Error())
		return a.newResult(err, map[string]any{"error": err.Error()})
	}
	if err := os.Remove(sqlResult.File); err != nil {
		a.logger.Error(err.Error())
		return a.newResult(err, nil)
	}
	return a.newResult(nil, nil)
}
