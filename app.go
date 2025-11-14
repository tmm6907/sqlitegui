package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"sqlitegui/models"
	"strings"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type DialogService interface {
	OpenDirectory(ctx context.Context, opts runtime.OpenDialogOptions) (string, error)
	OpenFile(ctx context.Context, opts runtime.OpenDialogOptions) (string, error)
}

type WailsDialogService struct{}

func (w *WailsDialogService) OpenDirectory(ctx context.Context, opts runtime.OpenDialogOptions) (string, error) {
	return runtime.OpenDirectoryDialog(ctx, opts)
}

func (w *WailsDialogService) OpenFile(ctx context.Context, opts runtime.OpenDialogOptions) (string, error) {
	return runtime.OpenFileDialog(ctx, opts)
}

type App struct {
	ctx        context.Context
	db         *sqlx.DB
	pkRegex    *regexp.Regexp
	logger     *slog.Logger
	unlocked   bool
	rootDBName string
	rootPath   string
	dialog     DialogService
}

type CustomAppConfig struct {
	RootDBName          string
	Logger              *slog.Logger
	AttachDetachEnabled bool
	DialogService
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
		dialog:     cfg.DialogService,
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	dbPath := a.getNewDBPath(a.rootDBName)
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

func (a *App) shutdown(ctx context.Context) {
	if a.db != nil {
		// This is the ONLY place db.Close() should be called.
		if err := a.db.Close(); err != nil {
			panic(err)
		}
	}
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

func (a *App) getNewDBPath(dbName string) string {
	dataDir, err := os.UserConfigDir()
	if err != nil {
		a.logger.Error(err.Error())
		return ""
	}
	return filepath.Join(dataDir, "sqlitegui", "dbs", fmt.Sprintf("%s.db", cleanDBName(dbName)))
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

func (a *App) storeDB(name string, path string, appCreated bool) error {
	attachQuery := fmt.Sprintf("ATTACH '%s' AS %s;", path, name)
	if _, err := a.db.Exec(attachQuery); err != nil {
		if strings.Contains(err.Error(), "already in use") {
			return nil
		}
		if appCreated {
			os.Remove(path)
		} // Clean up the created file
		return err
	}
	if _, err := a.db.Exec("INSERT OR IGNORE INTO main.dbs (name, path, root, app_created) VALUES (?,?,?,?);", name, path, a.rootPath, appCreated); err != nil {
		a.db.Exec(fmt.Sprintf("DETACH DATABASE '%s';", name))
		if appCreated {
			os.Remove(path)
		}
		return err
	}
	return nil
}

func (a *App) GetRootPath() AppResult {
	return a.newResult(nil, map[string]any{"root": a.rootPath}, nil)
}

func (a *App) getSQLiteDBName(path string) (string, error) {
	var internalDBName string
	if path == "" {
		return "", errors.New("path cannot be empty")
	}
	query := fmt.Sprintf("SELECT name FROM pragma_database_list WHERE file = '%s' LIMIT 1;", path)
	if err := a.db.Get(&internalDBName, query); err != nil {
		return "", err
	}
	return internalDBName, nil
}

// gets list of dbs from main.dbs
func (a *App) getStoredDBs() ([]models.DB, error) {
	var mainDbs []models.DB
	if err := a.db.Select(&mainDbs, "SELECT * from main.dbs WHERE root = ?", a.rootPath); err != nil {
		return mainDbs, err
	}
	return mainDbs, nil
}

func (a *App) getSQLiteDBNames() ([]string, error) {
	var names []string
	dbs, err := a.getStoredDBs()
	if err != nil {
		return []string{}, err
	}
	for _, db := range dbs {
		name, err := a.getSQLiteDBName(db.Path)
		if err != nil {
			return []string{}, err
		}
		names = append(names, name)
	}
	return names, nil
}

// returns the list of all tables for given sqlite db name
func (a *App) getTableList(dbName string) ([]string, error) {
	var tables []string
	query := fmt.Sprintf("SELECT name FROM %s.sqlite_master WHERE type='table';", dbName)
	if err := a.db.Select(&tables, query); err != nil {
		return tables, nil
	}
	return tables, nil
}

func (a *App) getTableData(tableName string) (Dataframe, error) {
	var res Dataframe
	query := fmt.Sprintf("SELECT * FROM %s;", tableName)
	rows, err := a.db.Queryx(query)
	if err != nil {
		return res, err
	}
	defer rows.Close()
	for rows.Next() {
		row := make(map[string]any)
		if err := rows.MapScan(row); err != nil {
			return res, err
		}
		res = append(res, row)
	}
	return res, nil
}

func (a *App) getColumns(tblName string) ([]string, error) {
	var names []string
	query := fmt.Sprintf("SELECT name FROM pragma_table_info('%s')", tblName)
	if err := a.db.Select(&names, query); err != nil {
		return names, err
	}
	return names, nil
}
