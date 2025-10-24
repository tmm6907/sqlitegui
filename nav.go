package main

import (
	"errors"
	"fmt"
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

	data := make(map[string][]string)
	var mainTables []string

	if err := a.db.Select(&mainTables, "SELECT name FROM sqlite_master WHERE type='table' AND name!='dbs';"); err != nil {

		a.logger.Error(fmt.Sprintf("Failed to fetch tables: %s", err.Error()))

		return a.newResult(err, nil)
	}
	data["main"] = mainTables

	var otherDBS []pragmaResult
	if err := a.db.Select(&otherDBS, "PRAGMA database_list;"); err != nil {
		a.logger.Error(fmt.Sprintf("Failed to fetch tables: %s", err.Error()))
		return a.newResult(err, nil)
	}

	if len(otherDBS) == 0 {
		return a.newResult(errors.New("no dbs attached"), nil)
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
