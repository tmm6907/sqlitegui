package main

import (
	_ "embed"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
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

func cleanQuery(query string) string {
	trimmedQuery := strings.TrimSpace(query)

	// 2. Split by semicolon
	statements := strings.Split(trimmedQuery, ";")

	// 3. Trim inner whitespace of each fragment
	var cleanedStatements []string
	for _, stmt := range statements {
		// Trim each fragment individually before rejoining
		trimmedStmt := strings.TrimSpace(stmt)
		if trimmedStmt != "" {
			cleanedStatements = append(cleanedStatements, trimmedStmt)
		}
	}

	// 4. Join with the standardized separator
	// Note: Removed empty statements from the slice above.
	return strings.Join(cleanedStatements, "; ")
}

func containsAttachStatement(query string) bool {
	statements := strings.SplitSeq(strings.TrimSpace(query), ";")
	for statement := range statements {
		trimmedStatement := strings.ToUpper(strings.TrimSpace(statement))
		if strings.HasPrefix(trimmedStatement, "ATTACH") || strings.HasPrefix(trimmedStatement, "DETACH") {
			return true
		}
	}
	return false
}

// Returns safe DB name and file ext of the file
func parseFile(file string) (string, string) {
	if file == "" {
		return file, ""
	}
	baseName := filepath.Base(file)
	var fileExt string
	if !strings.HasPrefix(baseName, ".") {
		fileExt = filepath.Ext(baseName)
	}
	dbNameWithoutExt := strings.TrimSuffix(baseName, fileExt)

	sanitizedName := reDuplicate.ReplaceAllString(dbNameWithoutExt, "")
	dbName := illegalDBChars.ReplaceAllString(sanitizedName, "_")
	return cleanDBName(dbName), fileExt
}

func sqlSanitize(tblName string) string {
	if tblName == "" {
		return `""`
	}

	tblName = strings.TrimSpace(tblName)
	tblName = illegalDBChars.ReplaceAllString(tblName, "_")

	nameRunes := []rune(tblName)
	if len(nameRunes) == 0 {
		return `""`
	}
	startsWithDigit := unicode.IsDigit(nameRunes[0])

	if startsWithDigit {
		tblName = "_" + tblName
	}
	for strings.Contains(tblName, "__") {
		tblName = strings.ReplaceAll(tblName, "__", "_")
	}
	if !startsWithDigit {
		tblName = strings.Trim(tblName, "_")
	}

	return fmt.Sprintf(`"%s"`, strings.ToLower(tblName))
}

func cleanDBName(file string) string {
	// 1. Get the actual filename component
	baseName := filepath.Base(file)

	// 2. Get the extension from the filename
	fileExt := filepath.Ext(baseName)

	// 3. Trim the extension from the filename (baseName)
	trimmedName := strings.TrimSuffix(baseName, fileExt)

	// 4. Handle remaining parts of the path (if you need the path cleaned)
	// NOTE: This is complex and usually not necessary if you only want the name.

	// 5. Return the cleaned name (lowercase and trimmed)
	return strings.ToLower(strings.TrimSpace(trimmedName))
}
