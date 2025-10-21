package main

import (
	"errors"
	"fmt"
	"sqlitegui/models"
)

func (a *App) GetNavData() Result {
	if a.db == nil {
		a.logger.Error("FATAL: GetNavData called before database was initialized or after it failed to open.")
		return a.newResult(errors.New("database not available"), nil)
	}
	a.attachDBs()

	data := make(map[string][]string)
	var mainTables []string

	if err := a.db.Select(&mainTables, "SELECT name FROM sqlite_master WHERE type='table' AND name!='dbs';"); err != nil {

		a.logger.Error(fmt.Sprintf("Failed to fetch tables: %s", err.Error()))

		return a.newResult(err, nil)
	}
	data["main"] = mainTables
	var otherDBS []models.DB
	if err := a.db.Select(&otherDBS, "SELECT * FROM dbs;"); err != nil {
		a.logger.Error(fmt.Sprintf("Failed to fetch tables: %s", err.Error()))
		return a.newResult(err, nil)
	}

	for _, db := range otherDBS {
		var tables []string
		query := fmt.Sprintf("SELECT name FROM %s.sqlite_master WHERE type='table';", db.Name)
		if err := a.db.Select(&tables, query); err != nil {
			a.logger.Error(fmt.Sprintf("Failed to fetch tables: %s", err.Error()))
			return a.newResult(err, nil)
		}
		data[db.Name] = tables
	}

	// FIX 4: Return the successful result payload directly
	return a.newResult(
		nil,
		data,
	)
}
