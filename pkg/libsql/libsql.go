package libsql

import (
	_ "github.com/tursodatabase/go-libsql"

	"database/sql"
)

func OpenLibsqlDb(filePath string) (*sql.DB, error) {
	return sql.Open("libsql", "file:"+filePath)
}
