package main

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"
)

var app *App

func TestMain(m *testing.M) {
	uniqueDBIdentifier := fmt.Sprintf("test_%d", time.Now().UnixNano())
	app = NewApp(&CustomAppConfig{Logger: NewSLogger(), RootDBName: uniqueDBIdentifier})
	app.startup(context.Background())

	exitCode := m.Run()

	app.shutdown(app.ctx)

	os.Exit(exitCode)
}

func TestCurrentDB(t *testing.T) {
	tests := []struct {
		dbName  string
		wantErr bool
	}{
		{"mustard", false},
		{"", true},
		{"2", false},
		{"mustard6", false},
	}

	for _, tt := range tests {
		res := app.SetCurrentDB(tt.dbName)
		if (res.Err != nil) != tt.wantErr {
			t.Errorf("error expected %v setting current db: %s", tt.wantErr, res.Err.Error())
		} else {
			if name, _ := app.getCurrentDB(); tt.dbName != "" && name != tt.dbName {
				t.Errorf("got current db as '%s', expected db as '%s'", name, tt.dbName)
			}
		}
	}
}

func TestCRUDDB(t *testing.T) {
	goodDb := "test_db"
	createTests := []struct {
		name    string
		cache   string
		journal string
		sync    string
		lock    string
		update  struct {
			db     string
			table  string
			row    [][]any
			column string
			value  string
		}
		wantErr bool
	}{
		{"", "", "", "", "", struct {
			db     string
			table  string
			row    [][]any
			column string
			value  string
		}{}, true},
		{"", "wal", "private", "", "", struct {
			db     string
			table  string
			row    [][]any
			column string
			value  string
		}{}, true},
		{goodDb, "wal", "private", "", "", struct {
			db     string
			table  string
			row    [][]any
			column string
			value  string
		}{"", "test", [][]any{
			{"id", "name"},
			{1, "test1"},
		}, "name", "test2"}, false},
	}
	query := `
	CREATE TABLE IF NOT EXISTS test (
	id PRIMARY_KEY,
	name TEXT
	);

	INSERT OR IGNORE INTO test (name) VALUES ("test1");
	`

	for _, tt := range createTests {
		res := app.CreateDB(CreateDBRequest{tt.name, tt.cache, tt.journal, tt.sync, tt.lock})
		if (res.Err != nil) != tt.wantErr {
			t.Errorf("DB: %s Expected error %v got: %v", tt.name, tt.wantErr, res.Err)
		}

		app.db.MustExec(query)

		res = app.UpdateDB(UpdateRequest{tt.update.db, tt.update.table, tt.update.row, tt.update.column, tt.update.value})
		if (res.Err != nil) != tt.wantErr {
			t.Errorf("Update DB: %s Expected error %v got: %v", tt.name, tt.wantErr, res.Err)
		}
		res = app.RemoveDB(tt.name)
		if (res.Err != nil) != tt.wantErr {
			t.Errorf("DB: %s Expected error %v got: %v", tt.name, tt.wantErr, res.Err)
		}
	}
}
