package main

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/jmoiron/sqlx"
)

func TestConvertToSQLite(t *testing.T) {
	tableName := "test_data"
	// 1. Setup in-memory SQLite database
	db, err := sqlx.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open in-memory database: %v", err)
	}
	defer db.Close()

	// 2. Define Test Cases
	tests := []struct {
		name             string
		df               *Dataframe
		tableName        string
		wantErr          bool
		expectedRowCount int
	}{
		{
			name:             "Empty Dataframe",
			df:               &Dataframe{},
			tableName:        tableName,
			wantErr:          true,
			expectedRowCount: 0,
		},
		{
			name:      "Valid Dataframe with Mixed Types",
			tableName: tableName,
			df: &Dataframe{
				{
					"ID":     int64(1),                                 // Should map to INTEGER
					"Name":   "Alice",                                  // Should map to TEXT
					"Score":  123.45,                                   // Should map to REAL
					"Active": true,                                     // Should map to INTEGER (0/1)
					"Data":   map[string]any{"key": "value", "id": 42}, // Should map to TEXT (JSON)
				},
				{
					"ID":     int64(2),
					"Name":   "Bob",
					"Score":  99.99,
					"Active": false,
					"Data":   []int{1, 2, 3}, // Should map to TEXT (JSON)
				},
			},
			wantErr:          false,
			expectedRowCount: 2,
		},
		{
			name: "Dataframe with Only Nil Values in First Row",
			df: &Dataframe{
				{"Col1": nil, "Col2": nil},
			},
			tableName:        tableName,
			wantErr:          true, // Should return "Dataframe has no valid columns..."
			expectedRowCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Execute the function
			err := convertToSQLite(tt.df, db, tt.tableName)

			if (err != nil) != tt.wantErr {
				t.Fatalf("convertToSQLite() error = %v, wantErr %v", err, tt.wantErr)
			}

			// If we expected an error and got it, move to next test case
			if tt.wantErr {
				return
			}

			// --- Post-Insertion Validation ---

			// 1. Check Row Count
			var count int
			countQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s", tt.tableName)
			if err := db.Get(&count, countQuery); err != nil {
				t.Fatalf("failed to query row count: %v", err)
			}
			if count != tt.expectedRowCount {
				t.Errorf("row count mismatch. Got %d, want %d", count, tt.expectedRowCount)
			}

			// 2. Check Data Integrity (fetch one row and verify content)
			var result struct {
				ID     int64   `db:"ID"`
				Name   string  `db:"Name"`
				Score  float64 `db:"Score"`
				Active int     `db:"Active"`
				Data   string  `db:"Data"`
			}
			dataQuery := fmt.Sprintf("SELECT * FROM %s WHERE ID = 1", tt.tableName)
			if err := db.Get(&result, dataQuery); err != nil {
				t.Fatalf("failed to fetch inserted row: %v", err)
			}

			// Verify values
			if result.Name != "Alice" {
				t.Errorf("Name mismatch. Got %s, want Alice", result.Name)
			}
			if result.Active != 1 {
				t.Errorf("Active mismatch. Got %d, want 1", result.Active)
			}
			dfData := *tt.df

			// Verify complex type (marshaled JSON string)
			expectedJSON, _ := json.Marshal(dfData[0]["Data"])
			if result.Data != string(expectedJSON) {
				t.Errorf("JSON Data mismatch.\nGot: %s\nWant: %s", result.Data, string(expectedJSON))
			}
		})
	}
}
