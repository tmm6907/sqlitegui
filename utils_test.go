package main // Ensure this package matches your source code

import (
	"fmt"
	"testing"
)

// NOTE: I am assuming the return signature for containsAttachStatement is (string, bool)
// as per the previous context, but the test struct only provides 'want' (a bool).
// The test below has been adjusted to only check the bool result (got2).

// NOTE: This test assumes the loop is fixed to `for _, statement := range statements`
// and that `strings.SplitSeq` is a typo for `strings.Split`.

func Test_containsAttachStatement(t *testing.T) {
	tests := []struct {
		name  string
		query string
		want  bool
	}{
		{
			name:  "No illegal statement",
			query: "SELECT * FROM users; INSERT INTO logs (message) VALUES ('New user');",
			want:  false,
		},
		{
			name:  "Attach statement present",
			query: "SELECT id FROM data; ATTACH DATABASE 'test.db' AS test;",
			want:  true,
		},
		{
			name:  "Detach statement present, case insensitive",
			query: "DetAcH DATABASE test; SELECT 1;",
			want:  true,
		},
		{
			name:  "Illegal command on multi-line",
			query: "\n  ATTACH DATABASE 'db'\n; SELECT * FROM T",
			want:  true,
		},
		{
			name:  "Prefix match only, NOT just ATTACH anywhere",
			query: "SELECT 'ATTACHMENT' AS name;",
			want:  false,
		},
		{
			name:  "Empty query",
			query: "",
			want:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// **FIXME:** If the actual function has the loop bug (range over indices),
			// this test will fail until the function is corrected.
			got := containsAttachStatement(tt.query)
			if got != tt.want {
				t.Errorf("containsAttachStatement() got = %v, want %v", got, tt.want)
			}
		})
	}
}

// NOTE: Assuming cleanQuery removes leading/trailing whitespace and potentially comments/newlines.
func Test_cleanQuery(t *testing.T) {
	tests := []struct {
		name  string
		query string
		want  string
	}{
		// 1. Basic Functionality: Ensures standard queries are formatted correctly.
		{
			name:  "1. Two standard statements",
			query: "SELECT 1; UPDATE users SET name='A'",
			want:  "SELECT 1; UPDATE users SET name='A'",
		},
		// 2. Outer Whitespace: Ensures leading/trailing spaces/newlines are removed.
		{
			name:  "2. Leading and trailing whitespace",
			query: "\n  SELECT * FROM users\n; INSERT INTO logs; \t",
			want:  "SELECT * FROM users; INSERT INTO logs",
		},
		// 3. Inner Whitespace: Ensures uneven spacing around the semicolon is normalized.
		{
			name:  "3. Uneven spacing around delimiters",
			query: "SELECT name  ;  INSERT new values",
			want:  "SELECT name; INSERT new values",
		},
		// 4. Trailing Semicolons: Ensures empty fragments from trailing delimiters are filtered out.
		{
			name:  "4. Trailing semicolon",
			query: "DELETE FROM logs;",
			want:  "DELETE FROM logs", // The final "" fragment is filtered.
		},
		// 5. Leading Semicolons: Ensures empty fragments from leading delimiters are filtered out.
		{
			name:  "5. Leading semicolon",
			query: ";SELECT 1; UPDATE 2",
			want:  "SELECT 1; UPDATE 2", // The initial "" fragment is filtered.
		},
		// 6. Multiple Semicolons: Ensures consecutive delimiters result in only one separator.
		{
			name:  "6. Multiple consecutive semicolons",
			query: "START TRANSACTION;;INSERT data; ;COMMIT",
			want:  "START TRANSACTION; INSERT data; COMMIT", // All empty fragments are filtered.
		},
		// 7. Empty Input: Tests empty strings and strings containing only whitespace.
		{
			name:  "7. Empty query string",
			query: "",
			want:  "",
		},
		{
			name:  "8. Only whitespace and semicolons",
			query: " ; \t ; \n ; ",
			want:  "", // All fragments are filtered as empty.
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cleanQuery(tt.query)
			if got != tt.want {
				// Use %q to clearly show whitespace differences in the output.
				t.Errorf("cleanQuery()\nGOT:  %q\nWANT: %q", got, tt.want)
			}
		})
	}
}

// NOTE: Assuming parseFile reads a file path and returns two pieces of data, like DB name and table name.
func Test_parseFile(t *testing.T) {
	tests := []struct {
		name  string
		file  string
		want  string // Cleaned DB Name
		want2 string // File Extension
	}{
		{
			name:  "1. Standard path and extension",
			file:  "/path/to/users_data.sql",
			want:  "users_data", // Clean name, lowercased
			want2: ".sql",
		},
		{
			name:  "2. Removes Windows/system copy suffix ' (1)'",
			file:  "archive (1).zip",
			want:  "archive", // ' (1)' is removed by reDuplicate
			want2: ".zip",
		},
		{
			name:  "3. Removes copy suffix with extra spaces ' ( 5 )'",
			file:  "document ( 5 ).pdf",
			want:  "document", // ' ( 5 )' is removed by reDuplicate
			want2: ".pdf",
		},
		{
			name:  "4. Sanitizes illegal characters in file name",
			file:  "my-report!v1.0.xlsx", // '-' and '!' are illegal
			want:  "my_report_v1_0",      // Replaced with '_' and lowercased
			want2: ".xlsx",
		},
		{
			name:  "5. Combination: Copy suffix and illegal characters",
			file:  "old-data_v1-0 (2).db",
			want:  "old_data_v1_0", // '-data_v1-0' becomes 'old_data_v1_0', then ' (2)' is removed.
			want2: ".db",
		},
		{
			name:  "6. File with no extension, containing illegal chars and suffix",
			file:  "file & name (3)",
			want:  "file_name", // '&' is replaced by '_', ' (3)' is removed.
			want2: "",
		},
		{
			name:  "7. File that is hidden (starts with a dot)",
			file:  "/home/user/.bashrc",
			want:  "_bashrc", // '.' is an illegal character and replaced with '_'
			want2: "",
		},
		{
			name:  "8. Complex path and multiple extensions",
			file:  "folder/final_project.tar.gz",
			want:  "final_project_tar", // Only the last extension (.gz) is removed by filepath.Ext
			want2: ".gz",
		},
		{
			name:  "9. Empty input",
			file:  "",
			want:  "",
			want2: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got2 := parseFile(tt.file)
			if got != tt.want {
				fmt.Println()
				t.Errorf("parseFile() got DB Name = %q, want %q", got, tt.want)
			}
			if got2 != tt.want2 {
				t.Errorf("parseFile() got File Ext = %q, want %q", got2, tt.want2)
			}
		})
	}
}

func Test_cleanTableName(t *testing.T) {
	tests := []struct {
		name    string
		tblName string
		want    string
	}{
		{
			name:    "Simple clean name",
			tblName: "Users",
			want:    `"users"`,
		},
		{
			name:    "Starts with digit",
			tblName: "4_data",
			want:    `"_4_data"`, // _4_data -> __4_data -> _4_data -> " _4_data " -> "_4_data"
		},
		{
			name:    "Illegal chars and spacing",
			tblName: " My!Test!Table ", // Assuming '!' is illegal
			want:    `"my_test_table"`,
		},
		{
			name:    "Multiple underscores consolidation",
			tblName: "__table___name",
			want:    `"table_name"`,
		},
		{
			name:    "Trailing and leading underscores removal",
			tblName: "___users___",
			want:    `"users"`,
		},
		{
			name:    "Empty name",
			tblName: " ",
			want:    `""`, // strings.Trim("_") on "_" results in ""
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sqlSanitize(tt.tblName)
			if got != tt.want {
				t.Errorf("cleanTableName() = %q, want %q", got, tt.want)
			}
		})
	}
}

// NOTE: Assuming cleanDBName removes quotes, backticks, or other SQL delimiters, and perhaps file extensions.
func Test_cleanDBName(t *testing.T) {
	tests := []struct {
		name      string
		inputName string
		want      string
	}{
		{
			name:      "Simple name, no extension",
			inputName: "My_Database",
			want:      "my_database",
		},
		{
			name:      "Removes extension and lowercases",
			inputName: "prod.db.Sqlite",
			want:      "prod.db",
		},
		{
			name:      "Handles leading/trailing whitespace",
			inputName: "\t Dev_DB.db ",
			want:      "dev_db",
		},
		{
			name:      "No extension, just lower case",
			inputName: "RAW_Data",
			want:      "raw_data",
		},
		{
			name:      "Name ending in dot (no extension)",
			inputName: "data.",
			want:      "data", // filepath.Ext() returns "" for names ending in '.', so no suffix is trimmed.
		},
		{
			name:      "Empty string",
			inputName: " ",
			want:      "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cleanDBName(tt.inputName)
			if got != tt.want {
				t.Errorf("cleanDBName() = %q, want %q", got, tt.want)
			}
		})
	}
}
