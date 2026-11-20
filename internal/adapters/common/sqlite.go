package common

import (
	"net/url"
	"path/filepath"
)

// SQLiteURI builds a file: URI so paths with spaces or special characters are
// escaped correctly before the query parameters are appended. modernc.org/sqlite
// only honors query parameters such as mode=ro for file: DSNs; for plain paths
// it silently drops them.
func SQLiteURI(path, query string) string {
	return (&url.URL{Scheme: "file", Path: filepath.ToSlash(path), RawQuery: query}).String()
}
