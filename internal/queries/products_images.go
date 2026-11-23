package queries

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"

	"github.com/shurco/mycart/internal/models"
	"github.com/shurco/mycart/pkg/errors"
	"github.com/shurco/mycart/pkg/security"
)

// productImage represents the database schema for product images.
// It serves as a data transfer object between the database and domain layer,
// keeping database concerns separate from the domain model.
type productImage struct {
	ID   string
	Name string
	Ext  string
}

// ProductImages orchestrates overall image retrieval process
func (q *ProductQueries) ProductImages(ctx context.Context, id string) (*[]models.File, error) {
	images, err := q.fetchProductImages(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("fetching product images: %w", err)
	}
	return q.mapToModelFiles(images), nil
}

// fetchProductImages performs database query to get image(s) for a product
func (q *ProductQueries) fetchProductImages(ctx context.Context, id string) ([]productImage, error) {
	query := `SELECT id, name, ext FROM product_image WHERE product_id = ?`
	rows, err := q.DB.QueryContext(ctx, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.ErrProductNotFound
		}
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var images []productImage
	for rows.Next() {
		var img productImage
		if err := rows.Scan(&img.ID, &img.Name, &img.Ext); err != nil {
			return nil, err
		}
		images = append(images, img)
	}
	return images, rows.Err()
}

// mapToModelFiles creates a representation of the database results as a domain object
func (q *ProductQueries) mapToModelFiles(images []productImage) *[]models.File {
	result := make([]models.File, len(images))
	for i, img := range images {
		result[i] = models.File{
			ID:   img.ID,
			Name: img.Name,
			Ext:  img.Ext,
		}
	}
	return &result
}

// AddImage attaches an image to a product by inserting a new record in the product_image table.
func (q *ProductQueries) AddImage(ctx context.Context, productID, fileUUID, fileExt, origName string) (*models.File, error) {
	file := &models.File{
		ID:   security.RandomString(),
		Name: fileUUID,
		Ext:  fileExt,
	}

	query := `INSERT INTO product_image (id, product_id, name, ext, orig_name) VALUES (?, ?, ?, ?, ?)`
	_, err := q.DB.ExecContext(ctx, query, file.ID, productID, file.Name, file.Ext, origName)
	if err != nil {
		return nil, err
	}

	return file, nil
}

// DeleteImage handles the deletion of a product image.
func (q *ProductQueries) DeleteImage(ctx context.Context, productID, imageID string) error {
	// Get image info and delete from database
	imageInfo, err := q.deleteImageRecord(ctx, productID, imageID)
	if err != nil {
		return fmt.Errorf("deleting image record: %w", err)
	}

	// Delete physical files
	if err := q.deleteImageFiles(imageInfo.name, imageInfo.ext); err != nil {
		return fmt.Errorf("deleting image files: %w", err)
	}

	return nil
}

type imageInfo struct {
	name string
	ext  string
}

func (q *ProductQueries) deleteImageRecord(ctx context.Context, productID, imageID string) (*imageInfo, error) {
	tx, err := q.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	// Get image info before deletion
	info := &imageInfo{}
	err = tx.QueryRowContext(ctx,
		`SELECT name, ext FROM product_image WHERE id = ?`,
		imageID).Scan(&info.name, &info.ext)
	if err != nil {
		return nil, err
	}

	// Delete the database record
	if _, err := tx.ExecContext(ctx,
		`DELETE FROM product_image WHERE id = ? AND product_id = ?`,
		imageID, productID); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return info, nil
}

func (q *ProductQueries) deleteImageFiles(name, ext string) error {
	filePaths := []string{
		fmt.Sprintf("./lc_uploads/%s.%s", name, ext),
		fmt.Sprintf("./lc_uploads/%s_sm.%s", name, ext),
	}

	var removeErrors []string
	for _, filePath := range filePaths {
		if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
			removeErrors = append(removeErrors,
				fmt.Sprintf("failed to remove %s: %v", filePath, err))
		}
	}

	if len(removeErrors) > 0 {
		return fmt.Errorf("file deletion errors: %s",
			strings.Join(removeErrors, "; "))
	}

	return nil
}
