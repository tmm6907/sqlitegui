package main

import (
	"context"
	_ "embed"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed build.sql
var buildScriptContent string

var (
	pkRegex = regexp.MustCompile(`(?i)SELECT\s+.*?\s+FROM\s+(\w+)`)
)

type App struct {
	ctx        context.Context
	db         *sqlx.DB
	pkRegex    *regexp.Regexp
	logger     *slog.Logger
	unlocked   bool
	rootDBName string
	rootPath   string
}

type CustomAppConfig struct {
	RootDBName          string
	Logger              *slog.Logger
	AttachDetachEnabled bool
}

func NewApp(cfg *CustomAppConfig) *App {
	if cfg.RootDBName == "" {
		cfg.RootDBName = "main"
	}
	return &App{
		rootDBName: cfg.RootDBName,
		logger:     cfg.Logger,
		unlocked:   cfg.AttachDetachEnabled,
		pkRegex:    pkRegex,
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	dbPath := a.getDBPath(a.rootDBName)
	// 3. Ensure the subdirectory exists (optional, but good practice)
	if err := os.MkdirAll(filepath.Dir(dbPath), SafePermissions); err != nil {
		panic(err)
	}
	db := sqlx.MustOpen(SQLITE_DRIVER, dbPath)
	db.MustExec("PRAGMA journal_mode=WAL;")

	db.MustExec(buildScriptContent)
	a.db = db
	a.logger.Info("starting app")
}

func (a *App) attachMainDBs() error {
	type dbInfo struct {
		Name string `db:"name"`
		Path string `db:"path"`
	}
	var rows []dbInfo
	if err := a.db.Select(&rows, "SELECT name, path from main.dbs WHERE root = ?;", a.rootPath); err != nil {
		a.logger.Error(err.Error())
		return err
	}
	for _, row := range rows {
		attachQuery := fmt.Sprintf("ATTACH '%s' AS %s;", row.Path, row.Name)
		if _, err := a.db.Exec(attachQuery); err != nil {
			a.logger.Error(err.Error(), slog.String("filename", row.Path))
			return err
		}
	}
	return nil
}

func (a *App) shutdown(ctx context.Context) {
	if a.db != nil {
		// This is the ONLY place db.Close() should be called.
		if err := a.db.Close(); err != nil {
			panic(err)
		}
	}
}

func (a *App) getDBPath(db_name string) string {
	dataDir, err := os.UserConfigDir()
	if err != nil {
		a.logger.Error(err.Error())
		return ""
	}
	return filepath.Join(dataDir, "sqlitegui", "dbs", fmt.Sprintf("%s.db", cleanDBName(db_name)))
}

// func (a *App) findPK(query string) []string {
// 	found := a.pkRegex.FindStringSubmatch(query)
// 	if len(found) <= 1 {
// 		return []string{}
// 	}
// 	tableName := found[1]
// 	a.logger.Debug("Table: %s", slog.String("tablename", tableName))

// 	// Step 1: Check for PRIMARY KEY columns using PRAGMA table_info
// 	var columnNames []string
// 	a.db.Select(&columnNames, fmt.Sprintf(`
// 		SELECT name FROM pragma_table_info('%s') WHERE pk > 0
// 	`, tableName))

// 	if len(columnNames) > 0 {
// 		return columnNames
// 	}

// 	// Step 2: If no PK columns were found, check for a primary key index (for composite PKs)
// 	var indexName string
// 	if err := a.db.Get(&indexName, fmt.Sprintf(`
// 		SELECT name FROM pragma_index_list('%s') WHERE origin='u' LIMIT 1
// 	`, tableName)); err != nil {
// 		return []string{}
// 	}
// 	if err := a.db.Select(&columnNames, fmt.Sprintf(`
// 		SELECT name FROM pragma_index_info('%s')
// 	`, indexName)); err != nil {
// 		return []string{}
// 	}
// 	return columnNames
// }

type EmitEvent struct {
	Type WailsEmitType
	Msg  string
}

func (a *App) newResult(err error, results any, emit *EmitEvent) AppResult {
	if emit != nil {
		runtime.EventsEmit(
			a.ctx,
			emit.Type.String(),
			map[string]string{"msg": emit.Msg},
		)
	}
	return AppResult{
		Err:     err,
		Results: results,
	}

}

func (a *App) emit(emitType WailsEmitType, emitMsg string) {
	runtime.EventsEmit(
		a.ctx,
		emitType.String(),
		map[string]string{"msg": emitMsg},
	)
}

func (a *App) GetRootPath() AppResult {
	return a.newResult(nil, map[string]any{"root": a.rootPath}, nil)
}

func (a *App) detachDBs() error {
	var dbNames []string
	if err := a.db.Select(&dbNames, "SELECT name FROM pragma_database_list;"); err != nil {
		return err
	}
	var detachQueries []string
	for _, dbName := range dbNames {
		if dbName != "main" {
			detachQueries = append(detachQueries, fmt.Sprintf("DETACH DATABASE '%s'", dbName))
		}
	}
	query := strings.Join(detachQueries, "; ")
	if _, err := a.db.Exec(query); err != nil {
		return err
	}
	return nil
}
