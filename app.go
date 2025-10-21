package main

import (
	"context"
	_ "embed"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

//go:embed build.sql
var buildScriptContent string

type App struct {
	ctx      context.Context
	db       *sqlx.DB
	pkRegex  *regexp.Regexp
	logger   *slog.Logger
	unlocked bool
}

func NewApp(logger *slog.Logger, unlocked bool) *App {
	return &App{
		logger:   logger,
		unlocked: unlocked,
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	db, err := a.getDB()
	if err != nil {
		a.logger.Error(err.Error())
		return
	}
	if db == nil {
		a.logger.Error("failed to create db")
		return
	}
	a.db = db

	if _, err := a.db.Exec(buildScriptContent); err != nil {
		a.logger.Error("unable to run build script for db: %s", slog.Any("error", err))
		return
	}

	if err := a.attachDBs(); err != nil {
		a.logger.Error(fmt.Sprintf("unable to attach dbs: %s", err.Error()))
		return
	}

	a.pkRegex = regexp.MustCompile(`(?i)SELECT\s+.*?\s+FROM\s+(\w+)`)
	a.logger.Info("starting app")
}

func (a *App) attachDBs() error {
	rows, err := a.db.Query("SELECT name, path from dbs;")
	if err != nil {
		return err
	}
	for rows.Next() {
		var path string
		var name string
		if err := rows.Scan(&name, &path); err != nil {
			return err
		}
		attachQuery := fmt.Sprintf("ATTACH '%s' AS %s;", path, name)
		if _, err := a.db.Exec(attachQuery); err != nil {
			a.logger.Debug(err.Error())
			return err
		}
		a.logger.Debug(fmt.Sprintf("Attached %s successfully", name))
	}
	return nil
}

func (a *App) shutdown(ctx context.Context) {
	if a.db != nil {
		// This is the ONLY place db.Close() should be called.
		if err := a.db.Close(); err != nil {
			a.logger.Error(fmt.Sprintf("Failed to close database: %s", err.Error()))
			return
		}
	}
}

func (a *App) getDBPath(db_name string) string {
	dataDir, err := os.UserConfigDir()
	if err != nil {
		a.logger.Error(err.Error())
		return ""
	}
	return filepath.Join(dataDir, "sqlitegui", "dbs", fmt.Sprintf("%s.db", db_name))
}

func (a *App) getDB() (*sqlx.DB, error) {

	// 2. Combine with a subdirectory for your app and the db filename
	dbPath := a.getDBPath("main")
	// 3. Ensure the subdirectory exists (optional, but good practice)
	if err := os.MkdirAll(filepath.Dir(dbPath), SafePermissions); err != nil {
		return nil, err
	}
	db, err := sqlx.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("DB failed to open %s: %s", dbPath, err.Error())
	}
	if _, err := db.Exec("PRAGMA journal_mode=WAL;"); err != nil {
		return nil, fmt.Errorf("unable to configure db: %s", err.Error())
	}

	return db, nil

}

func (a *App) findPK(query string) []string {
	found := a.pkRegex.FindStringSubmatch(query)
	if len(found) <= 1 {
		return []string{}
	}
	tableName := found[1]
	a.logger.Debug("Table: %s", slog.String("tablename", tableName))

	// Step 1: Check for PRIMARY KEY columns using PRAGMA table_info
	var columnNames []string
	a.db.Select(&columnNames, fmt.Sprintf(`
		SELECT name FROM pragma_table_info('%s') WHERE pk > 0
	`, tableName))

	if len(columnNames) > 0 {
		return columnNames
	}

	// Step 2: If no PK columns were found, check for a primary key index (for composite PKs)
	var indexName string
	if err := a.db.Get(&indexName, fmt.Sprintf(`
		SELECT name FROM pragma_index_list('%s') WHERE origin='u' LIMIT 1
	`, tableName)); err != nil {
		return []string{}
	}
	if err := a.db.Select(&columnNames, fmt.Sprintf(`
		SELECT name FROM pragma_index_info('%s')
	`, indexName)); err != nil {
		return []string{}
	}
	return columnNames
}

func (a *App) newResult(err error, results any) Result {
	if err != nil {
		return Result{
			ErrStr:  err.Error(),
			Results: results,
		}
	}
	return Result{
		ErrStr:  "",
		Results: results,
	}
}
