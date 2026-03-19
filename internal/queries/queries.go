package queries

import (
	"database/sql"
	"embed"

	"github.com/shurco/litecart/internal/base"
	_ "modernc.org/sqlite"
)

var db *Base

// Define the structure 'Base' that aggregates various queries related to different modules like
// settings, authentication, installation, pages, products, and cart management.
type Base struct {
	SettingQueries
	AuthQueries
	InstallQueries
	PageQueries
	ProductQueries
	CartQueries
}

// New initializes the application's database and returns an error if any occurs during the process.
// It takes an 'embed.FS' which represents the file system intended to be used with embedded files.
func New(embed embed.FS) (err error) {
	var sqlite *sql.DB
	sqlite, err = base.New("./lc_base/data.db", embed)
	if err != nil {
		return
	}

	db = &Base{
		AuthQueries:    AuthQueries{DB: sqlite},
		InstallQueries: InstallQueries{DB: sqlite},
		SettingQueries: SettingQueries{DB: sqlite},
		PageQueries:    PageQueries{DB: sqlite},
		ProductQueries: ProductQueries{DB: sqlite},
		CartQueries:    CartQueries{DB: sqlite},
	}
	return
}

// NewFromDB initializes Base from an existing *sql.DB (e.g. in-memory SQLite for tests).
func NewFromDB(sqlite *sql.DB) {
	db = &Base{
		AuthQueries:    AuthQueries{DB: sqlite},
		InstallQueries: InstallQueries{DB: sqlite},
		SettingQueries: SettingQueries{DB: sqlite},
		PageQueries:    PageQueries{DB: sqlite},
		ProductQueries: ProductQueries{DB: sqlite},
		CartQueries:    CartQueries{DB: sqlite},
	}
}

// DB returns the Base instance. If the database is not initialized, returns nil.
// Use New() to initialize the database before calling DB().
func DB() *Base {
	return db
}
