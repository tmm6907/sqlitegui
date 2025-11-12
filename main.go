package main

import (
	"embed"
	"log/slog"
	"os"
	"os/exec"

	"runtime"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/menu/keys"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
	rt "github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:frontend/dist
var assets embed.FS

const (
	APP_NAME      = "SQLite GUI"
	SQLITE_DRIVER = "sqlite3"
	SCREEN_WIDTH  = 1920
	SCREEN_HEIGHT = 1080
)

type TargetOS string

func (t TargetOS) String() string {
	return string(t)
}

const (
	LINUX   TargetOS = "linux"
	MAC_OS  TargetOS = "darwin"
	WINDOWS TargetOS = "windows"
)

type WailsEmitType string

func (w WailsEmitType) String() string {
	return string(w)
}

const (
	DB_UPLOAD_SUCCESS   WailsEmitType = "dbUploadSucceeded"
	DB_UPLOAD_FAIL      WailsEmitType = "dbUploadFailed"
	DB_EXPORT_SUCCESS   WailsEmitType = "dbExportSucceeded"
	DB_EXPORT_FAIL      WailsEmitType = "dbExportFailed"
	NEW_WINDOW_SUCCESS  WailsEmitType = "newWindowSucceeded"
	NEW_WINDOW_FAIL     WailsEmitType = "newWindowFailed"
	OPEN_FOLDER_SUCCESS WailsEmitType = "openFolderSucceeded"
	OPEN_FOLDER_FAIL    WailsEmitType = "openFolderFailed"
	IMPORT_DB_SUCCESS   WailsEmitType = "importDBSucceeded"
	IMPORT_DB_FAIL      WailsEmitType = "importDBFailed"
)

func NewSLogger() *slog.Logger {
	// Configure the handler options
	opts := &slog.HandlerOptions{
		Level:     slog.LevelDebug, // Set your desired minimum log level
		AddSource: true,            // This includes the file and line number
	}

	// Create a TextHandler writing to Stderr
	handler := slog.NewTextHandler(os.Stderr, opts)

	// Create and return the logger
	return slog.New(handler)
}

func (a *App) newAppMenu() *menu.Menu {
	AppMenu := menu.NewMenu()
	isMac := runtime.GOOS == MAC_OS.String()
	if isMac {
		AppMenu.Append(menu.AppMenu())
		AppMenu.Append(menu.EditMenu())
	}

	FileMenu := AppMenu.AddSubmenu("File")
	FileMenu.AddText("New Window", keys.CmdOrCtrl("n"), func(cd *menu.CallbackData) {
		appExecutable, err := os.Executable()
		if err != nil {
			a.logger.Error(err.Error())
			a.emit(NEW_WINDOW_FAIL, err.Error())
			return
		}
		cmd := exec.Command(appExecutable)
		if err = cmd.Start(); err != nil {
			a.logger.Error(err.Error())
			a.emit(NEW_WINDOW_FAIL, err.Error())
			return
		}
		a.emit(NEW_WINDOW_SUCCESS, "")
	})
	FileMenu.AddText("Import", keys.CmdOrCtrl("o"), func(_ *menu.CallbackData) {
		a.importDB()
	})
	FileMenu.AddText("Open Folder...", keys.CmdOrCtrl("K"), func(_ *menu.CallbackData) {
		a.openFolder()
	})
	FileMenu.AddSeparator()
	FileMenu.AddText("Create new db from data file", keys.CmdOrCtrl("D"), func(_ *menu.CallbackData) {
		a.uploadDB()
	})
	FileMenu.AddSeparator()
	exportSubMenu := menu.NewMenuFromItems(
		menu.Text("Export to New DB File", keys.Combo("d", keys.CmdOrCtrlKey, keys.ShiftKey), func(_ *menu.CallbackData) {
			a.exportDB(".db")
		}),

		menu.Text("Export to CSV (Zip)", keys.Combo("c", keys.CmdOrCtrlKey, keys.ShiftKey), func(_ *menu.CallbackData) {
			a.exportDB(".csv")
		}),

		menu.Text("Export to JSON (Zip)", keys.Combo("j", keys.CmdOrCtrlKey, keys.ShiftKey), func(_ *menu.CallbackData) {
			a.exportDB(".json")
		}),

		menu.Text("Export to SQL (ZIP)", keys.Combo("s", keys.CmdOrCtrlKey, keys.ShiftKey), func(_ *menu.CallbackData) {
			a.exportDB(".sql")
		}),

		menu.Text("Export to DB (ZIP)", keys.Combo("b", keys.CmdOrCtrlKey, keys.ShiftKey), func(_ *menu.CallbackData) {
			a.exportDB("")
		}),
	)
	FileMenu.Merge(exportSubMenu)
	FileMenu.AddSeparator()
	FileMenu.AddText("Settings", keys.CmdOrCtrl("S"), func(_ *menu.CallbackData) {

	})
	FileMenu.AddSeparator()
	FileMenu.AddText("Quit", keys.CmdOrCtrl("q"), func(_ *menu.CallbackData) {
		rt.Quit(a.ctx)
	})
	return AppMenu
}

func main() {
	// Create an instance of the app structure
	logger := NewSLogger()
	app := NewApp(&CustomAppConfig{
		Logger: logger,
	})
	AppMenu := app.newAppMenu()
	appInstance := options.App{
		Title:  APP_NAME,
		Width:  SCREEN_WIDTH,
		Height: SCREEN_HEIGHT,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup:  app.startup,
		OnShutdown: app.shutdown,
		Bind: []any{
			app,
		},
		Menu: AppMenu,
		Linux: &linux.Options{
			ProgramName: APP_NAME,
		},
		Windows: &windows.Options{
			Theme: windows.Dark,
		},
		// EnableDefaultContextMenu: true,
		// Debug: options.Debug{
		// 	OpenInspectorOnStartup: true,
		// },
	}
	err := wails.Run(&appInstance)

	if err != nil {
		panic(err)
	}
}
