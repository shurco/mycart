package csvimport

import (
	"database/sql"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Create minimal schema
	schema := `
		CREATE TABLE product (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			slug TEXT UNIQUE NOT NULL,
			desc TEXT,
			amount NUMERIC,
			quantity INTEGER DEFAULT 0,
			digital TEXT,
			active BOOLEAN DEFAULT TRUE,
			deleted BOOLEAN DEFAULT FALSE
		);
	`
	_, err = db.Exec(schema)
	if err != nil {
		t.Fatalf("Failed to create test schema: %v", err)
	}

	return db
}

func TestCSVValidation(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	importer := NewCSVImporter(db)

	tests := []struct {
		name       string
		csv        string
		wantErrors int
		wantRows   int
	}{
		{
			name: "valid CSV",
			csv: `name,slug,amount,digital
Product 1,product-1,1000,file
Product 2,product-2,2000,file`,
			wantErrors: 0,
			wantRows:   2,
		},
		{
			name: "missing required field value",
			csv: `name,slug,amount,digital
Product 1,product-1,1000,`,
			wantErrors: 1,
			wantRows:   1,
		},
		{
			name: "invalid amount",
			csv: `name,slug,amount,digital
Product 1,product-1,invalid,file`,
			wantErrors: 1,
			wantRows:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.csv)
			result, _, err := importer.ValidateAndPreview(reader)
			if err != nil {
				t.Fatalf("ValidateAndPreview() error = %v", err)
			}

			if len(result.Errors) != tt.wantErrors {
				t.Errorf("Expected %d errors, got %d", tt.wantErrors, len(result.Errors))
			}

			if result.TotalRows != tt.wantRows {
				t.Errorf("Expected %d rows, got %d", tt.wantRows, result.TotalRows)
			}
		})
	}
}
