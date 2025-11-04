package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func (a *App) SetupMain() Result {
	a.rootPath = "main"
	err := a.attachMainDBs()
	if err != nil {
		a.logger.Error(err.Error())
		return a.newResult(err, nil)
	}
	return a.newResult(nil, nil)
}

func (a *App) OpenFolderOnStart() Result {
	selection, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Open DB Folder",
	})
	if err != nil {
		a.logger.Error(err.Error())
		return a.newResult(err, nil)
	}
	a.rootPath = selection
	err = a.attachMainDBs()
	if err != nil {
		a.logger.Error(err.Error())
		return a.newResult(err, nil)
	}
	err = a.attachDBsFromFolder(selection)
	if err != nil {
		a.logger.Error(err.Error())
		return a.newResult(err, nil)
	}
	return a.newResult(nil, map[string]any{"root": selection})
}
func (a *App) attachDBsFromFolder(rootPath string) error {
	a.rootPath = rootPath

	err := filepath.WalkDir(rootPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			a.logger.Error(fmt.Sprintf("Error accessing path %q: %v", path, err))
			return err // Stop the walk on critical error
		}

		if d.IsDir() {
			return nil
		}

		baseName, ext := parseFile(path)
		if slices.Contains(dbFileTypes, ext) {

			// relPath, relErr := filepath.Rel(rootPath, path)
			// if relErr != nil {
			// 	a.logger.Error(fmt.Sprintf("Failed to get relative path for %s: %v", path, relErr))
			// 	return nil
			// }

			// baseName := relPath[:len(relPath)-len(ext)]

			safeAlias := strings.ReplaceAll(baseName, string(filepath.Separator), "_")
			safeAlias = strings.ReplaceAll(safeAlias, ".", "_")

			if safeAlias == "main" {
				safeAlias = "main_db"
				a.logger.Debug("Alias 'main' conflicted, renamed to 'main_db'.")
			}
			a.logger.Debug(fmt.Sprintf("Attaching DB: %s as Alias: %s", path, safeAlias))

			if err := a.attachFolderDB(path, safeAlias, rootPath); err != nil {
				a.logger.Error(fmt.Sprintf("Failed to attach and persist DB %s: %v", path, err))
				return nil
			}
		}

		return nil
	})

	return err
}

func (a *App) attachFolderDB(path string, name string, rootPath string) error {
	const mainSchema = "main"

	query := fmt.Sprintf("ATTACH '%s' AS %s;", path, name)
	a.logger.Debug(fmt.Sprintf("query: %s", query))
	if a.db == nil {
		return errors.New("db is not initialized")
	}
	if _, err := a.db.Exec(query); err != nil {
		a.logger.Error(err.Error())
		return err
	}

	insertQuery := fmt.Sprintf("INSERT OR IGNORE INTO %s.dbs (path, name, root) VALUES (?,?,?)", mainSchema)
	if _, err := a.db.Exec(insertQuery, path, name, rootPath); err != nil {
		a.logger.Error(err.Error())

		a.db.Exec(fmt.Sprintf("DETACH DATABASE %s;", name))
		return err
	}

	return nil
}
