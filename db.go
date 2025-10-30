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

	if _, err := a.db.Exec("INSERT INTO dbs (name, path, root, app_created) VALUES (?,?,?,?);", dbForm.Name, dbPath, a.rootPath, true); err != nil {
		a.logger.Error(err.Error())
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
