package main

import (
	"errors"
	"fmt"
)

func (a *App) GetNavData() AppResult {
	type DBResult struct {
		Tables     []string `json:"tables"`
		AppCreated bool     `json:"appCreated"`
	}
	var mainTables []string

	if a.db == nil {
		a.logger.Error("FATAL: GetNavData called before database was initialized or after it failed to open.")
		return a.newResult(errors.New("database not available"), nil, nil)
	}

	data := make(map[string]DBResult)
	if err := a.db.Select(&mainTables, "SELECT name FROM main.sqlite_master WHERE type='table' AND name!='dbs';"); err != nil {

		a.logger.Error(fmt.Sprintf("Failed to fetch tables: %s", err.Error()))

		return a.newResult(err, nil, nil)
	}
	data["main"] = DBResult{mainTables, false}
	otherDBS, err := a.getStoredDBs()
	if err != nil {
		a.logger.Error(fmt.Sprintf("Failed to fetch tables: %s", err.Error()))
		return a.newResult(err, nil, nil)
	}
	for _, db := range otherDBS {
		dbName, err := a.getSQLiteDBName(db.Path)
		if err != nil {
			a.logger.Error(err.Error())
			return a.newResult(err, nil, nil)
		}
		tables, err := a.getTableList(dbName)
		if err != nil {
			a.logger.Error(fmt.Sprintf("Failed to fetch tables: %s", err.Error()))
			return a.newResult(err, nil, nil)
		}
		data[db.Name] = DBResult{tables, db.App_Created}
	}
	return a.newResult(
		nil,
		data,
		nil,
	)
}
