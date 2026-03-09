package libsql

import (
	_ "github.com/tursodatabase/go-libsql"

	"database/sql"
)

func OpenLibsqlDb(filePath string) (*sql.DB, error) {
	// Add busy timeout to handle concurrent access
	dsn := "file:" + filePath + "?_busy_timeout=5000"
	return sql.Open("libsql", dsn)
}
