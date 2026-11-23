package queries

import (
	"context"

	"github.com/shurco/mycart/internal/database"
	"github.com/shurco/mycart/pkg/errors"
	"github.com/shurco/mycart/pkg/slugify"
)

// ProductQueries is a struct that holds a dialect-aware database handle.
// This allows for direct access to the database methods via the ProductQueries struct,
// calls on it are automatically rebound for the configured dialect.
type ProductQueries struct {
	DB *database.Conn
}

// DeleteProduct removes a product from the database based on its ID.
func (q *ProductQueries) DeleteProduct(ctx context.Context, id string) error {
	// Guard: a hard delete cascades into digital_file/digital_data, which
	// would revoke downloads already purchased by customers (rows assigned
	// to a paid cart). Refuse such deletes; admins should deactivate the
	// product instead.
	var sold int
	err := q.DB.QueryRowContext(ctx,
		`SELECT COUNT(1) FROM digital_data WHERE product_id = ? AND cart_id IS NOT NULL`, id).Scan(&sold)
	if err != nil {
		return err
	}
	if sold > 0 {
		return errors.ErrProductSold
	}

	_, err = q.DB.ExecContext(ctx, `DELETE FROM product WHERE id = ?`, id)
	return err
}

// IsProduct checks if a product with the given slug exists and is active.
func (q *ProductQueries) IsProduct(ctx context.Context, slug string) bool {
	var exists bool
	query := `
			SELECT EXISTS (
				SELECT 1 FROM product
				WHERE product.slug = ? AND product.deleted = FALSE AND product.active = TRUE
			)
	`
	err := q.DB.QueryRowContext(ctx, query, slug).Scan(&exists)
	return err == nil && exists
}

// UpdateActive toggles the 'active' status of a product and updates its 'updated' timestamp.
// It takes a context and an ID as arguments, and returns an error if the operation fails.
func (q *ProductQueries) UpdateActive(ctx context.Context, id string) error {
	query := `UPDATE product SET active = NOT active, updated = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := q.DB.ExecContext(ctx, query, id)
	return err
}

// GenerateUniqueSlug generates a unique URL-friendly slug from product name
func (q *ProductQueries) GenerateUniqueSlug(ctx context.Context, name string, excludeProductID string) (string, error) {
	service := slugify.NewSlugService(q.DB)
	return service.Generate(ctx, name, excludeProductID)
}
