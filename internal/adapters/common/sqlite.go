package common

import (
	"database/sql"
	"net/url"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// SQLiteURI builds a file: URI so paths with spaces or special characters are
// escaped correctly before the query parameters are appended. modernc.org/sqlite
// only honors query parameters such as mode=ro for file: DSNs; for plain paths
// it silently drops them.
func SQLiteURI(path, query string) string {
	return (&url.URL{Scheme: "file", Path: filepath.ToSlash(path), RawQuery: query}).String()
}

// OpenSQLiteRO opens a SQLite database read-only, waiting briefly on a busy database.
func OpenSQLiteRO(path string) (*sql.DB, error) {
	return sql.Open("sqlite", SQLiteURI(path, "mode=ro&_pragma=busy_timeout(2000)"))
}
