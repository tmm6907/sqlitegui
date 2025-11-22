package main

import (
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
					"id":     int64(1),                                 // Should map to INTEGER
					"name":   "Alice",                                  // Should map to TEXT
					"score":  123.45,                                   // Should map to REAL
					"active": true,                                     // Should map to INTEGER (0/1)
					"data":   map[string]any{"key": "value", "id": 42}, // Should map to TEXT (JSON)
				},
				{
					"id":     int64(2),
					"name":   "Bob",
					"score":  99.99,
					"active": false,
					"data":   []int{1, 2, 3}, // Should map to TEXT (JSON)
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
			wantErr:          false, // Should return "Dataframe has no valid columns..."
			expectedRowCount: 1,
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

			res := make(map[string]any)

			rows, err := db.Queryx(fmt.Sprintf("SELECT * FROM %s", tt.tableName))
			if err != nil {
				t.Fatalf("no rows selected %v", err)
			}
			for rows.Next() {
				if err := rows.MapScan(res); err != nil {
					t.Fatalf("no rows selected %v", err)
				}
			}
		})
	}
}
