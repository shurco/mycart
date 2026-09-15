package queries

import (
	"context"
	"database/sql"
	"fmt"
)

// ImagePositionUpdate contains data for updating a product image position.
type ImagePositionUpdate struct {
	ImageID  string
	Position int
}

// UpdateProductImagePositions updates positions for multiple product images in a single transaction.
// All updates succeed or all fail (atomic operation).
func (q *ProductQueries) UpdateProductImagePositions(ctx context.Context, updates []ImagePositionUpdate) error {
	if len(updates) == 0 {
		return nil
	}

	tx, err := q.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Update each image position
	for _, update := range updates {
		result, err := tx.ExecContext(ctx,
			`UPDATE product_image SET position = ? WHERE id = ?`,
			update.Position,
			update.ImageID,
		)
		if err != nil {
			return fmt.Errorf("update image position: %w", err)
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("get rows affected: %w", err)
		}

		if rowsAffected == 0 {
			return fmt.Errorf("image not found: %s", update.ImageID)
		}
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// SetRepresentativeImage sets one image as representative and unmarks all others for the same product.
// Performs both operations atomically in a single transaction.
func (q *ProductQueries) SetRepresentativeImage(ctx context.Context, productID string, imageID string) error {
	tx, err := q.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// First, verify the image exists and belongs to this product
	var existingProductID string
	err = tx.QueryRowContext(ctx,
		`SELECT product_id FROM product_image WHERE id = ?`,
		imageID,
	).Scan(&existingProductID)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("image not found: %s", imageID)
		}
		return fmt.Errorf("query image: %w", err)
	}

	if existingProductID != productID {
		return fmt.Errorf("image does not belong to product: image_id=%s, product_id=%s", imageID, productID)
	}

	// Unmark all images for this product
	_, err = tx.ExecContext(ctx,
		`UPDATE product_image SET is_representative = FALSE WHERE product_id = ?`,
		productID,
	)
	if err != nil {
		return fmt.Errorf("unmark images: %w", err)
	}

	// Mark the specified image as representative
	_, err = tx.ExecContext(ctx,
		`UPDATE product_image SET is_representative = TRUE WHERE id = ?`,
		imageID,
	)
	if err != nil {
		return fmt.Errorf("mark representative image: %w", err)
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}
