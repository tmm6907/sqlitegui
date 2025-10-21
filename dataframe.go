package main

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/jmoiron/sqlx"
)

type Dataframe []map[string]any

func castToSQLiteType(t reflect.Type) string {
	switch t.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return "INTEGER"
	case reflect.Float32, reflect.Float64:
		return "REAL" // REAL is for floating point numbers
	case reflect.Bool:
		return "INTEGER" // SQLite often stores booleans as 0 or 1
	case reflect.String:
		return "TEXT"

	// Add explicit checks for complex types that need serialization
	case reflect.Slice, reflect.Array, reflect.Map, reflect.Struct:
		return "TEXT"

	default:
		// Default to TEXT for any other unknown Go type
		return "TEXT"
	}
}

func (d Dataframe) convertToSQLite(db *sqlx.DB, tableName string) error {
	if len(d) == 0 {
		return fmt.Errorf("cannot convert empty Dataframe to database")
	}

	firstRow := d[0]
	columnDefinitions := ""
	columnNames := []string{}

	for colName, value := range firstRow {
		valueType := reflect.TypeOf(value)
		if valueType == nil {
			continue
		}
		sqliteType := castToSQLiteType(valueType)
		columnDefinitions += fmt.Sprintf("%s %s, ", colName, sqliteType)
		columnNames = append(columnNames, colName)
	}

	if len(columnNames) == 0 {
		return fmt.Errorf("Dataframe has no valid columns to infer schema")
	}

	columnDefinitions = strings.TrimSuffix(columnDefinitions, ", ")
	createTableSQL := fmt.Sprintf("CREATE TABLE %s (%s)", tableName, columnDefinitions)

	if _, err := db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s;", tableName)); err != nil {
		return fmt.Errorf("error dropping existing table: %w", err)
	}

	if _, err := db.Exec(createTableSQL); err != nil {
		return fmt.Errorf("error creating table: %w", err)
	}

	placeholders := strings.TrimSuffix(strings.Repeat("?, ", len(columnNames)), ", ")
	insertSQL := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		tableName,
		strings.Join(columnNames, ", "),
		placeholders)

	tx, err := db.Begin() // Start transaction on the NEW database
	if err != nil {
		return err
	}

	stmt, err := tx.Prepare(insertSQL)
	if err != nil {
		tx.Rollback()
		return err
	}
	defer stmt.Close()

	for _, row := range d {
		values := make([]any, len(columnNames))

		for i, colName := range columnNames {
			value := row[colName]
			v := reflect.ValueOf(value)

			if v.IsValid() && (v.Kind() == reflect.Slice || v.Kind() == reflect.Map || v.Kind() == reflect.Struct || v.Kind() == reflect.Array) {
				jsonBytes, marshalErr := json.Marshal(value)
				if marshalErr != nil {
					tx.Rollback()
					return fmt.Errorf("failed to marshal complex type for column %s: %w", colName, marshalErr)
				}
				values[i] = string(jsonBytes)
			} else {
				values[i] = value
			}
		}

		_, err = stmt.Exec(values...)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("error inserting row into %s: %w", tableName, err)
		}
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("error committing transaction: %w", err)
	}

	return nil
}
