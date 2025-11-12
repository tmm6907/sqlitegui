package main

import "testing"

func TestQuery(t *testing.T) {
	tests := []struct {
		query    string
		editable bool
		wantErr  bool
	}{
		{"", false, true},
		{"mustard", false, true},
		{"SELECT * FROM pragma_database_list;", false, false},
		{"PRAGMA database_list;", true, false},
	}

	for _, tt := range tests {
		res := test_app.Query(QueryRequest{tt.query, tt.editable})
		if (res.Err != nil) != tt.wantErr {
			t.Errorf("Expected: %v Got: %s Query: '%s' ", tt.wantErr, res.Err, tt.query)
		}
	}
}

func TestQueryAll(t *testing.T) {
	tests := []struct {
		table   string
		wantErr bool
	}{
		{"", true},
		{"mustard", true},
		{"pragma_database_list", false},
	}

	for _, tt := range tests {
		res := test_app.QueryAll(tt.table)
		if (res.Err != nil) != tt.wantErr {
			t.Errorf("Expected: %v Got: %s Table: '%s' ", tt.wantErr, res.Err, tt.table)
		}
	}
}
