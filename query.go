package main

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strings"
)

func (a *App) handleSelectQueries(query string, editable bool) AppResult {
	rows, err := a.db.Query(query)
	if err != nil {
		a.logger.Error("failed to run query: %s", slog.Any("error", err.Error()))
		return a.newResult(
			fmt.Errorf("failed to run query: %s | %s", query, err.Error()),
			map[string]any{
				"error": BadRequestError,
			},
			nil,
		)
	}
	columns, _ := rows.Columns()

	var rowData [][]any
	for rows.Next() {
		values := make([]any, len(columns))
		valuePtrs := make([]any, len(columns))

		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			a.logger.Error("failed to run query: %s", slog.Any("error", err))
			return a.newResult(
				err,
				map[string]any{
					"error": InternalServerError,
				},
				nil,
			)
		}

		rowMap := make([]any, len(columns))
		for i := range columns {
			rowMap[i] = values[i]
		}
		rowData = append(rowData, rowMap)
	}
	data := map[string]any{
		"cols":     columns,
		"rows":     rowData,
		"editable": editable,
	}
	return a.newResult(nil, data, nil)
}

type QueryRequest struct {
	Query    string `json:"query"`
	Editable bool   `json:"editable"`
}

func (a *App) Query(q QueryRequest) AppResult {
	editable := q.Editable
	query := q.Query
	if query == "" {
		return a.newResult(
			errors.New(BadRequestError),
			map[string]any{
				"error": BadRequestError,
			},
			nil,
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
				nil,
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
				nil,
			)
		}
		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return a.newResult(
				errors.New(BadRequestError),
				map[string]any{
					"error": BadRequestError,
				},
				nil,
			)
		}
		a.logger.Debug("Rows affected: %s", slog.Int64("debug", rowsAffected))
		return a.newResult(
			nil,
			map[string]any{
				"rowsAffected": rowsAffected,
			},
			nil,
		)
	}
}

func (a *App) QueryAll(table string) AppResult {
	dbName, err := a.getCurrentDB()
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		a.logger.Error(err.Error())
		return a.newResult(err, nil, nil)
	}
	var query string
	if dbName != "" {
		query = fmt.Sprintf("SELECT * FROM %s.%s LIMIT 50;", dbName, table)
	} else {
		query = fmt.Sprintf("SELECT * FROM %s LIMIT 50;", table)
	}
	return a.handleSelectQueries(query, true)
}
