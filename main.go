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
	FileMenu.AddText("Import", keys.CmdOrCtrl("I"), func(_ *menu.CallbackData) {
		app.importDB()
	})
	FileMenu.AddSeparator()
	FileMenu.AddText("Export as ...", keys.CmdOrCtrl("E"), func(_ *menu.CallbackData) {
		app.exportDB()
	})
	FileMenu.AddSeparator()
	FileMenu.AddText("Create new db from file", keys.CmdOrCtrl("D"), func(_ *menu.CallbackData) {
		app.uploadDB()
	})
	FileMenu.AddSeparator()
	FileMenu.AddText("Quit", keys.CmdOrCtrl("q"), func(_ *menu.CallbackData) {
		rt.Quit(app.ctx)
	})
	programName := "SQLite GUI"

	// Create application with options
	err := wails.Run(&options.App{
		Title:  programName,
		Width:  1920,
		Height: 1080,
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
			ProgramName: programName,
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
