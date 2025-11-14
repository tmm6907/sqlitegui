package main

import (
	"testing"
)

func TestCurrentDB(t *testing.T) {
	tests := []struct {
		dbName  string
		wantErr bool
	}{
		{"mustard", false},
		{"2", false},
		{"mustard6", false},
		{"", false},
	}

	for _, tt := range tests {
		res := test_app.SetCurrentDB(tt.dbName)
		if (res.Err != nil) != tt.wantErr {
			t.Errorf("error expected %v setting current db: %s", tt.wantErr, res.Err.Error())
		} else {
			if name, _ := test_app.getCurrentDB(); tt.dbName != "" && name != tt.dbName {
				t.Errorf("got current db as '%s', expected db as '%s'", name, tt.dbName)
			}
		}
	}
	// res := test_app.SetCurrentDB("")
	// if res.Err != nil {
	// 	t.Errorf("error: %s", res.Err.Error())
	// }
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
		res := test_app.CreateDB(CreateDBRequest{tt.name, tt.cache, tt.journal, tt.sync, tt.lock})
		if (res.Err != nil) != tt.wantErr {
			t.Errorf("DB: %s Expected error %v got: %v", tt.name, tt.wantErr, res.Err)
		}

		test_app.db.MustExec(query)

		res = test_app.UpdateDB(UpdateRequest{tt.update.db, tt.update.table, tt.update.row, tt.update.column, tt.update.value})
		if (res.Err != nil) != tt.wantErr {
			t.Errorf("Update DB: %s Expected error %v got: %v", tt.name, tt.wantErr, res.Err)
		}
		res = test_app.RemoveDB(tt.name)
		if (res.Err != nil) != tt.wantErr {
			t.Errorf("DB: %s Expected error %v got: %v", tt.name, tt.wantErr, res.Err)
		}
	}
}
