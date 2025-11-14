package main

import (
	"encoding/json"
	"fmt"
	"log"
	"reflect"
	"strings"

	"github.com/jmoiron/sqlx"
)

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

type Series = map[string]any
type Dataframe = []Series

func convertToSQLite(d *Dataframe, db *sqlx.DB, tableName string) error {
	if len(*d) == 0 {
		return fmt.Errorf("cannot convert empty Dataframe to database")
	}
	df := *d

	firstRow := df[0]
	columnDefinitions := ""
	columnNames := []string{}

	for colName, value := range firstRow {
		valueType := reflect.TypeOf(value)
		if valueType == nil {
			continue
		}
		sqliteType := castToSQLiteType(valueType)
		// name := cleanTableName(colName)
		name := colName
		columnDefinitions += fmt.Sprintf("%s %s, ", name, sqliteType)
		columnNames = append(columnNames, name)
	}

	if len(columnNames) == 0 {
		return fmt.Errorf("Dataframe has no valid columns to infer schema")
	}

	columnDefinitions = strings.TrimSuffix(columnDefinitions, ", ")
	createTableSQL := fmt.Sprintf("CREATE TABLE %s (%s)", tableName, columnDefinitions)

	if _, err := db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s;", tableName)); err != nil {
		return fmt.Errorf("error dropping existing table %s: %w", tableName, err)
	}

	if _, err := db.Exec(createTableSQL); err != nil {
		return fmt.Errorf("error creating table %s: %w | %s", tableName, err, createTableSQL)
	}
	log.Println(createTableSQL)
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

	for _, row := range df {
		values := make([]any, len(columnNames))
		// log.Println(row)
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

// map{dbName: map{ tableName : [map {columnName: value}]}}

func convertFromSQLite(db *sqlx.DB) (*Dataframe, error) {
	return nil, nil
}
