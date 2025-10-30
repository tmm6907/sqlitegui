package main

import (
	"errors"
	"fmt"
	"sqlitegui/models"
)

type pragmaResult struct {
	Seq  int    `db:"seq"`
	Name string `db:"name"`
	File string `db:"file"`
}

func (a *App) GetNavData() Result {
	if a.db == nil {
		a.logger.Error("FATAL: GetNavData called before database was initialized or after it failed to open.")
		return a.newResult(errors.New("database not available"), nil)
	}

	data := make(map[string]DBResult)
	var mainTables []string

	if err := a.db.Select(&mainTables, "SELECT name FROM main.sqlite_master WHERE type='table' AND name!='dbs';"); err != nil {

		a.logger.Error(fmt.Sprintf("Failed to fetch tables: %s", err.Error()))

		return a.newResult(err, nil)
	}
	data["main"] = DBResult{mainTables, false}
	a.logger.Debug(fmt.Sprintf("Main Data:, %v", data))
	var otherDBS []models.DB
	if err := a.db.Select(&otherDBS, "SELECT * from main.dbs WHERE root = ?", a.rootPath); err != nil {
		a.logger.Error(fmt.Sprintf("Failed to fetch tables: %s", err.Error()))
		return a.newResult(err, nil)
	}
	dbNames := make([]string, len(otherDBS))
	a.logger.Debug(fmt.Sprint(otherDBS))
	for i, db := range otherDBS {
		var dbName string
		query := fmt.Sprintf("SELECT name FROM pragma_database_list WHERE file = '%s' LIMIT 1;", db.Path)
		a.logger.Debug(query)
		a.db.Get(&dbName, query)
		if dbName != "" {
			dbNames[i] = dbName
		}
	}
	a.logger.Debug(fmt.Sprint(dbNames))

	for i, db := range dbNames {
		var tables []string
		if db == "" {
			continue
		}
		query := fmt.Sprintf("SELECT name FROM %s.sqlite_master WHERE type='table';", db)
		a.logger.Debug(query)
		if err := a.db.Select(&tables, query); err != nil {
			a.logger.Error(fmt.Sprintf("Failed to fetch tables: %s", err.Error()))
			return a.newResult(err, nil)
		}
		data[otherDBS[i].Name] = DBResult{tables, otherDBS[i].App_Created}
	}

	a.logger.Debug(fmt.Sprintf("All Data:, %v", data))

	// FIX 4: Return the successful result payload directly
	return a.newResult(
		nil,
		data,
	)
}
