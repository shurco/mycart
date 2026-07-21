package csvimport

// ImportResult contains the results of a CSV import operation
type ImportResult struct {
	TotalRows int     `json:"total_rows"`
	Imported  int     `json:"imported"`
	Updated   int     `json:"updated"`
	Skipped   int     `json:"skipped"`
	ToAdd     int     `json:"to_add"`    // Preview only
	ToUpdate  int     `json:"to_update"` // Preview only
	Errors    []Error `json:"errors"`
}

// Error represents an import error with line number
type Error struct {
	Line    int    `json:"line"`
	Message string `json:"message"`
}

// ImportMode specifies how to handle existing products
type ImportMode string

const (
	ModeUpsert ImportMode = "upsert"
)
