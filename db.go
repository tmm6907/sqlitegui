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
	if _, err := a.db.Exec("UPDATE current_db SET current_db = ? WHERE id = 1;", name); err != nil {
		a.logger.Error(err.Error())
		return a.newResult(
			err,
			nil,
		)
	}
	return a.newResult(
		nil,
		nil,
	)
}

func (a *App) GetCurrentDB() Result {
	var dbName string
	if err := a.db.Get(&dbName, "SELECT current_db from current_db where id=1;"); err != nil {
		a.logger.Error(err.Error())
		return a.newResult(
			err,
			nil,
		)
	}
	return a.newResult(
		nil,
		dbName,
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
	var count int

	if dbForm.Cache == "" || dbForm.Journal == "" || dbForm.Lock == "" || dbForm.Name == "" || dbForm.Sync == "" {
		err := errors.New("invalid request. all fields are required")
		a.logger.Error(err.Error())
		return a.newResult(err, map[string]any{"error": BadRequestError})
	}
	dbForm.Name = cleanDBName(dbForm.Name)

	if err := a.db.Get(&count, "SELECT COUNT(*) FROM dbs WHERE name = ?;", dbForm.Name); err != nil {
		a.logger.Error(err.Error())
		return a.newResult(err, map[string]string{
			"error": InternalServerError,
		})
	}

	if count > 0 {
		err := fmt.Errorf("database by that name already exists")
		a.logger.Error(err.Error())
		return a.newResult(err, map[string]string{
			"error": "database by that name already exists",
		})
	}

	dbPath := a.getDBPath(dbForm.Name)
	if err := os.WriteFile(dbPath, []byte{}, SafePermissions); err != nil {
		a.logger.Error(err.Error())
		return a.newResult(
			err,
			map[string]string{
				"error": InternalServerError,
			},
		)
	}

	if _, err := a.db.Exec("INSERT INTO dbs (name, path) VALUES (?,?);", dbForm.Name, dbPath); err != nil {
		a.logger.Error(err.Error())
		return a.newResult(
			err,
			map[string]any{
				"error": InternalServerError,
			})
	}
	a.logger.Debug(fmt.Sprintf("path: %s", dbPath))

	if strings.ToLower(dbForm.Journal) == "wal" {
		newDB, err := sqlx.Open("sqlite3", dbPath)
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

	attachQuery := fmt.Sprintf("ATTACH '%s' AS %s;", dbPath, dbForm.Name)
	if _, err := a.db.Exec(attachQuery); err != nil {
		a.logger.Error(err.Error())
		return a.newResult(
			err,
			map[string]any{
				"error": InternalServerError,
			})
	}

	return a.newResult(
		nil,
		map[string]any{
			"message": fmt.Sprintf("%s has been completed successfully", dbForm.Name),
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
