package main

import (
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
)

func (a *App) handleSelectQueries(query string, editable bool) Result {
	a.logger.Debug(query)
	rows, err := a.db.Query(query)
	if err != nil {
		a.logger.Error("failed to run query: %s", slog.Any("error", err.Error()))
		return a.newResult(
			fmt.Errorf("failed to run query: %s | %s", query, err.Error()),
			map[string]any{
				"error": BadRequestError,
			},
		)
	}
	columns, _ := rows.Columns()

	var rowData [][]any
	for rows.Next() {
		values := make([]any, len(columns))
		valuePtrs := make([]any, len(columns))

		// Assign each pointer to the corresponding interface
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		// Scan the row into the value pointers
		if err := rows.Scan(valuePtrs...); err != nil {
			a.logger.Error("failed to run query: %s", slog.Any("error", err))
			return a.newResult(
				err,
				map[string]any{
					"error": InternalServerError,
				},
			)
		}

		// Create a map for the current row
		rowMap := make([]any, len(columns))
		for i := range columns {
			// Dereference the value
			rowMap[i] = values[i]
		}

		rowData = append(rowData, rowMap)
	}
	indexes := a.findPK(query)
	includedIndexes := []string{}
	if strings.Contains(query, "*") {
		includedIndexes = indexes
	} else {
		for _, i := range indexes {
			if strings.Contains(query, i) {
				includedIndexes = append(includedIndexes, i)
			}
		}
	}

	cols := includedIndexes
	for _, col := range columns {
		if !slices.Contains(cols, col) {
			cols = append(cols, col)
		}
	}

	return a.newResult(
		nil,
		map[string]any{
			"pk":       len(indexes) > 0,
			"cols":     columns,
			"rows":     rowData,
			"editable": editable,
		})
}

type QueryRequest struct {
	Query    string `json:"query"`
	Editable bool   `json:"editable"`
}

func (a *App) Query(q QueryRequest) Result {
	editable := q.Editable
	query := q.Query
	if query == "" {
		return a.newResult(
			errors.New(BadRequestError),
			map[string]any{
				"error": BadRequestError,
			},
		)
	}
	if !a.unlocked {
		q, found := ContainsAttachStatement(query)
		if found {
			return a.newResult(
				errors.New(BadRequestError),
				map[string]any{
					"error": BadRequestError,
				},
			)
		}
		query = q
	}
	if strings.HasPrefix(strings.ToUpper(query), "SELECT") || strings.HasPrefix(strings.ToUpper(query), "PRAGMA") {
		return a.handleSelectQueries(query, editable)
	} else {
		result, err := a.db.Exec(query)
		if err != nil {
			a.logger.Error("query failed to execute", slog.Any("error", err))
			return a.newResult(
				err,
				map[string]any{
					"error": BadRequestError,
				},
			)
		}
		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return a.newResult(
				errors.New(BadRequestError),
				map[string]any{
					"error": BadRequestError,
				},
			)
		}
		a.logger.Debug("Rows affected: %s", slog.Int64("debug", rowsAffected))
		return a.newResult(
			nil,
			map[string]any{
				"rowsAffected": rowsAffected,
			},
		)
	}
}

func (a *App) QueryAll(table string) Result {
	dbName, err := a.getCurrentDB()
	if err != nil {
		a.logger.Error(err.Error())
		return a.newResult(err, nil)
	}
	if dbName == "" {
		return a.newResult(errors.New("unable to determine current db"), nil)
	}

	query := fmt.Sprintf("SELECT * FROM %s.%s LIMIT 50;", dbName, table)
	return a.handleSelectQueries(query, true)
}
