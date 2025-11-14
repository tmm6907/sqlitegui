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

func (a *App) SetupMain() AppResult {
	a.rootPath = "main"
	if err := a.attachMainDBs(); err != nil {
		a.logger.Error(err.Error())
		return a.newResult(err, nil, nil)
	}
	return a.newResult(nil, nil, nil)
}

func (a *App) OpenFolderOnStart() AppResult {
	selection, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Open DB Folder",
	})
	if err != nil {
		a.logger.Error(err.Error())
		return a.newResult(err, nil, nil)
	}
	if selection == "" {
		err := errors.New("selection cannot be empty")
		return a.newResult(err, nil, nil)
	}
	a.rootPath = selection

	if err = a.attachMainDBs(); err != nil {
		a.logger.Error(err.Error())
		return a.newResult(err, nil, nil)
	}
	if err = a.attachDBsFromFolder(selection); err != nil {
		a.logger.Error(err.Error())
		return a.newResult(err, nil, nil)
	}
	return a.newResult(nil, map[string]any{"root": selection}, nil)
}
func (a *App) attachDBsFromFolder(targetPath string) error {
	err := filepath.WalkDir(targetPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			a.logger.Error(fmt.Sprintf("Error accessing path %q: %v", path, err))
			return err // Stop the walk on critical error
		}

		if d.IsDir() {
			return nil
		}

		baseName, ext := parseFile(path)
		if slices.Contains(dbFileTypes[:], ext) {
			safeAlias := strings.ReplaceAll(baseName, string(filepath.Separator), "_")
			safeAlias = strings.ReplaceAll(safeAlias, ".", "_")

			if safeAlias == "main" {
				safeAlias = "main_db"
			}
			a.logger.Debug(fmt.Sprintf("Attaching DB: %s as Alias: %s", path, safeAlias))

			if err := a.storeDB(safeAlias, path, false); err != nil {
				a.logger.Error(fmt.Sprintf("Failed to attach and persist DB %s: %v", path, err))
			}
		}
		return nil
	})
	return err
}
