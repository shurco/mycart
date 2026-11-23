package csvimport

import (
	"context"
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"

	"github.com/shurco/mycart/internal/models"
)

// Querier is the database surface CSVImporter needs. It is declared here, on
// the consumer side, so this package does not depend on the storage layer and
// stays usable with any handle that rebinds placeholders for its dialect.
type Querier interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

// CSVImporter handles CSV import operations
type CSVImporter struct {
	db Querier
}

// NewCSVImporter creates a new CSV importer
func NewCSVImporter(db Querier) *CSVImporter {
	return &CSVImporter{db: db}
}

// Required CSV columns
var requiredColumns = []string{"name", "slug", "amount", "digital"}

// ValidateAndPreview parses CSV and returns preview without importing
func (c *CSVImporter) ValidateAndPreview(file io.Reader) (*ImportResult, []models.Product, error) {
	reader := csv.NewReader(file)
	reader.TrimLeadingSpace = true

	headerMap, err := readHeaderMap(reader)
	if err != nil {
		return nil, nil, err
	}

	result := &ImportResult{}
	products := []models.Product{}

	lineNum := 2 // Start from 2 (1 is header)
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			result.Errors = append(result.Errors, Error{
				Line:    lineNum,
				Message: fmt.Sprintf("CSV parse error: %v", err),
			})
			lineNum++
			continue
		}

		if product, ok := c.previewRecord(record, headerMap, lineNum, result); ok {
			products = append(products, product)
		}

		lineNum++
	}

	result.TotalRows = lineNum - 2
	return result, products, nil
}

// readHeaderMap reads the header row and maps column names to their index,
// refusing a file that lacks a required column.
func readHeaderMap(reader *csv.Reader) (map[string]int, error) {
	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("read header: %w", err)
	}

	headerMap := make(map[string]int)
	for i, col := range header {
		headerMap[col] = i
	}

	for _, req := range requiredColumns {
		if _, exists := headerMap[req]; !exists {
			return nil, fmt.Errorf("missing required column: %s", req)
		}
	}
	return headerMap, nil
}

// previewRecord parses one CSV row and counts it in result. It reports false when
// the row did not parse and so is not part of the preview.
func (c *CSVImporter) previewRecord(record []string, headerMap map[string]int, lineNum int, result *ImportResult) (models.Product, bool) {
	product, err := c.parseProduct(record, headerMap)
	if err != nil {
		result.Errors = append(result.Errors, Error{
			Line:    lineNum,
			Message: err.Error(),
		})
		result.Skipped++
		return product, false
	}

	// Check if product exists
	exists, err := c.productExists(context.Background(), product.Slug)
	if err != nil {
		result.Errors = append(result.Errors, Error{
			Line:    lineNum,
			Message: fmt.Sprintf("database error: %v", err),
		})
	} else if exists {
		result.ToUpdate++
	} else {
		result.ToAdd++
	}
	return product, true
}

// productExists checks if a product with the given slug exists
func (c *CSVImporter) productExists(ctx context.Context, slug string) (bool, error) {
	var count int
	err := c.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM product WHERE slug = ?", slug).Scan(&count)
	return count > 0, err
}

// Import executes the import operation
func (c *CSVImporter) Import(ctx context.Context, products []models.Product) (*ImportResult, error) {
	result := &ImportResult{}

	for _, product := range products {
		exists, err := c.productExists(ctx, product.Slug)
		if err != nil {
			result.Errors = append(result.Errors, Error{
				Message: fmt.Sprintf("Failed to check %s: %v", product.Slug, err),
			})
			continue
		}

		if exists {
			// Update logic would go here - for now, skip
			result.Skipped++
		} else {
			// Insert product (simplified - would use AddProductWithVariants in production)
			query := `INSERT INTO product (id, name, slug, "desc", amount, quantity, digital, active, deleted)
			          VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`
			_, err = c.db.ExecContext(ctx, query,
				product.ID, product.Name, product.Slug, product.Description,
				product.Amount, product.Quantity, product.Digital.Type, product.Active, false)

			if err != nil {
				result.Errors = append(result.Errors, Error{
					Message: fmt.Sprintf("Failed to insert %s: %v", product.Slug, err),
				})
			} else {
				result.Imported++
			}
		}
	}

	result.TotalRows = len(products)
	return result, nil
}
