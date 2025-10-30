package main

import (
	"embed"
	"log"
	"log/slog"
	"os"

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

func main() {
	// Create an instance of the app structure
	logger := NewSLogger()
	app := NewApp(logger, true)

	AppMenu := menu.NewMenu()
	isMac := runtime.GOOS == "darwin"
	// AppMenu.Append(menu.WindowMenu())
	if isMac {
		AppMenu.Append(menu.AppMenu())
		AppMenu.Append(menu.EditMenu())
	}
	FileMenu := AppMenu.AddSubmenu("File")
	FileMenu.AddText("New Window", keys.CmdOrCtrl("n"), func(cd *menu.CallbackData) {

	})
	FileMenu.AddText("Import", keys.CmdOrCtrl("o"), func(_ *menu.CallbackData) {
		app.importDB()
	})
	FileMenu.AddText("Open Folder...", keys.CmdOrCtrl("K"), func(_ *menu.CallbackData) {
		app.openFolder()
	})
	FileMenu.AddSeparator()
	FileMenu.AddText("Create new db from data file", keys.CmdOrCtrl("D"), func(_ *menu.CallbackData) {
		app.uploadDB()
	})
	FileMenu.AddSeparator()
	exportSubMenu := menu.NewMenuFromItems(
		menu.Text("Export to New DB File", keys.Combo("d", keys.CmdOrCtrlKey, keys.ShiftKey), func(_ *menu.CallbackData) {
			app.exportDB(".db")
		}),

		menu.Text("Export to CSV (Zip)", keys.Combo("c", keys.CmdOrCtrlKey, keys.ShiftKey), func(_ *menu.CallbackData) {
			app.exportDB(".csv")
		}),

		menu.Text("Export to JSON (Zip)", keys.Combo("j", keys.CmdOrCtrlKey, keys.ShiftKey), func(_ *menu.CallbackData) {
			app.exportDB(".json")
		}),

		menu.Text("Export to SQL (ZIP)", keys.Combo("s", keys.CmdOrCtrlKey, keys.ShiftKey), func(_ *menu.CallbackData) {
			app.exportDB(".sql")
		}),

		menu.Text("Export to DB (ZIP)", keys.Combo("b", keys.CmdOrCtrlKey, keys.ShiftKey), func(_ *menu.CallbackData) {
			app.exportDB("")
		}),
	)
	FileMenu.Merge(exportSubMenu)
	FileMenu.AddSeparator()
	FileMenu.AddText("Settings", keys.CmdOrCtrl("S"), func(_ *menu.CallbackData) {

	})
	FileMenu.AddSeparator()
	FileMenu.AddText("Quit", keys.CmdOrCtrl("q"), func(_ *menu.CallbackData) {
		rt.Quit(app.ctx)
	})

	// Create application with options
	err := wails.Run(&options.App{
		Title:  APP_NAME,
		Width:  SCREEN_WIDTH,
		Height: SCREEN_HEIGHT,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		// Frameless:        !isMac,
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
		Debug: options.Debug{
			OpenInspectorOnStartup: true,
		},
	})

	if err != nil {
		log.Fatal(err.Error())
	}
}
