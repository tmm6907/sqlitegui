package main

import (
	_ "embed"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

//go:embed build.sql
var buildScriptContent string

const (
	APP_NAME      = "SQLite GUI"
	SQLITE_DRIVER = "sqlite3"
	SCREEN_WIDTH  = 1920
	SCREEN_HEIGHT = 1080

	SafePermissions     = 0755
	InternalServerError = "internal server error"
	BadRequestError     = "bad request"

	LINUX   TargetOS = "linux"
	MAC_OS  TargetOS = "darwin"
	WINDOWS TargetOS = "windows"

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

var (
	illegalDBChars = regexp.MustCompile(`[^a-zA-Z0-9_]+`)

	// Finds common Windows/system file copy suffixes like " (1)", "( 2)", or " ( 3 )"
	// at the end of a string ($). These are replaced with an empty string.
	reDuplicate = regexp.MustCompile(`\s*\(\s*\d+\s*\)\s*$`)

	pkRegex = regexp.MustCompile(`(?i)SELECT\s+.*?\s+FROM\s+(\w+)`)

	dbFileTypes   = [2]string{".db", ".sqlite"}
	SYSTEM_TABLES = [2]string{"dbs", "current_db"}
)

type TargetOS string

func (t TargetOS) String() string {
	return string(t)
}

type WailsEmitType string

func (w WailsEmitType) String() string {
	return string(w)
}

type DBFileType string

func (t DBFileType) String() string {
	return string(t)
}

func containsAttachStatement(query string) (string, bool) {
	illegalStatement := false
	statements := strings.Split(query, "\n")

	for _, statement := range statements {

		if strings.HasPrefix(strings.ToUpper(statement), "ATTACH") {
			illegalStatement = true
		}
		if strings.HasPrefix(strings.ToUpper(statement), "DETACH") {
			illegalStatement = true
		}
	}

	return strings.Join(statements, " "), illegalStatement
}

func parseFile(selection string) (string, string) {
	baseName := filepath.Base(selection)
	fileExt := filepath.Ext(baseName)
	dbNameWithExt := strings.TrimSuffix(baseName, fileExt)

	sanitizedName := reDuplicate.ReplaceAllString(dbNameWithExt, "")
	dbName := illegalDBChars.ReplaceAllString(sanitizedName, "_")
	return cleanDBName(dbName), fileExt
}

func cleanTableName(name string) string {
	name = strings.TrimSpace(name)
	name = illegalDBChars.ReplaceAllString(name, "_")
	if name != "" && name[0] >= '0' && name[0] <= '9' {
		name = "_" + name
	}
	name = strings.ReplaceAll(name, "__", "_")
	for strings.Contains(name, "__") {
		name = strings.ReplaceAll(name, "__", "_")
	}
	name = strings.ToLower(strings.Trim(name, "_"))
	return fmt.Sprintf(`"%s"`, name)
}

func cleanDBName(inputName string) string {
	inputName = strings.TrimSpace(inputName)
	return strings.ToLower(
		strings.TrimSuffix(
			inputName,
			filepath.Ext(inputName),
		),
	)
}
