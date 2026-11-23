package queries

import (
	"context"
	"database/sql"
	stderrors "errors"
	"fmt"
	"os"

	"github.com/shurco/mycart/internal/digitalfiles"
	"github.com/shurco/mycart/internal/models"
	"github.com/shurco/mycart/pkg/errors"
	"github.com/shurco/mycart/pkg/security"
)

// ProductDigital retrieves the digital content associated with a given product ID.
func (q *ProductQueries) ProductDigital(ctx context.Context, productID string) (*models.Digital, error) {
	digital := &models.Digital{}

	query := `
			SELECT 
					p.digital,
					df.id, df.name, df.ext,
					dd.id, dd.content, dd.cart_id
			FROM product p
			LEFT JOIN digital_file df ON p.id = df.product_id
			LEFT JOIN digital_data dd ON p.id = dd.product_id
			WHERE p.id = ?
	`

	rows, err := q.DB.QueryContext(ctx, query, productID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		if err := scanDigitalRow(rows, digital); err != nil {
			return nil, err
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return digital, nil
}

// scanDigitalRow adds the file and the data record of one ProductDigital row to digital.
func scanDigitalRow(rows *sql.Rows, digital *models.Digital) error {
	var digitalType sql.NullString
	var fileID, fileName, fileExt sql.NullString
	var dataID, dataContent, cartID sql.NullString

	err := rows.Scan(
		&digitalType,
		&fileID, &fileName, &fileExt,
		&dataID, &dataContent, &cartID,
	)
	if err != nil {
		return err
	}

	if digital.Type == "" {
		digital.Type = digitalType.String
	}

	if fileID.Valid {
		file := models.File{
			ID:   fileID.String,
			Name: fileName.String,
			Ext:  fileExt.String,
		}
		digital.Files = append(digital.Files, file)
	}
	if dataID.Valid {
		data := models.Data{
			ID:      dataID.String,
			Content: dataContent.String,
			CartID:  cartID.String,
		}
		digital.Data = append(digital.Data, data)
	}

	return nil
}

// DigitalFile returns a single digital file record for a product.
// Used by the authenticated admin download endpoint.
func (q *ProductQueries) DigitalFile(ctx context.Context, productID, fileID string) (*models.File, error) {
	file := &models.File{}
	query := `SELECT id, name, ext, orig_name FROM digital_file WHERE id = ? AND product_id = ?`
	err := q.DB.QueryRowContext(ctx, query, fileID, productID).
		Scan(&file.ID, &file.Name, &file.Ext, &file.OrigName)
	if err != nil {
		if stderrors.Is(err, sql.ErrNoRows) {
			return nil, errors.ErrProductNotFound
		}
		return nil, err
	}
	return file, nil
}

// AddDigitalFile associates a digital file with a product in the database.
func (q *ProductQueries) AddDigitalFile(ctx context.Context, productID, fileUUID, fileExt, origName string) (*models.File, error) {
	file := &models.File{
		ID:   security.RandomString(),
		Name: fileUUID,
		Ext:  fileExt,
	}

	query := `INSERT INTO digital_file (id, product_id, name, ext, orig_name) VALUES (?, ?, ?, ?, ?)`
	_, err := q.DB.ExecContext(ctx, query, file.ID, productID, file.Name, file.Ext, origName)
	if err != nil {
		return nil, err
	}

	return file, nil
}

// AddDigitalData adds a new digital data record associated with a product.
func (q *ProductQueries) AddDigitalData(ctx context.Context, productID, content string) (*models.Data, error) {
	file := &models.Data{
		ID:      security.RandomString(),
		Content: content,
	}

	query := `INSERT INTO digital_data (id, product_id, content) VALUES (?, ?, ?)`
	_, err := q.DB.ExecContext(ctx, query, file.ID, productID, file.Content)
	if err != nil {
		return nil, err
	}

	return file, nil
}

// UpdateDigital updates the content of a digital data record in the database.
func (q *ProductQueries) UpdateDigital(ctx context.Context, digital *models.Data) error {
	query := `UPDATE digital_data SET content = ? WHERE id = ?`
	_, err := q.DB.ExecContext(ctx, query, digital.Content, digital.ID)
	return err
}

func (q *ProductQueries) DeleteDigital(ctx context.Context, productID, digitalID string) error {
	var digitalType string
	var name, ext sql.NullString

	query := `
				SELECT p.digital, df.name, df.ext
				FROM product p
				LEFT JOIN digital_file df ON df.id = ? AND df.product_id = p.id
				WHERE p.id = ?
		`

	err := q.DB.QueryRowContext(ctx, query, digitalID, productID).Scan(&digitalType, &name, &ext)
	if err != nil {
		return err
	}

	switch digitalType {
	case models.DigitalFile:
		query = `DELETE FROM digital_file WHERE id = ? AND product_id = ?`
		if _, err := q.DB.ExecContext(ctx, query, digitalID, productID); err != nil {
			return fmt.Errorf("deleting from digital_file: %w", err)
		}

		filePath := digitalfiles.Path(name.String, ext.String)
		if err := os.Remove(filePath); err != nil {
			return fmt.Errorf("failed to remove file %s: %w", filePath, err)
		}

	case models.DigitalData:
		query = `DELETE FROM digital_data WHERE id = ? AND product_id = ?`
		if _, err := q.DB.ExecContext(ctx, query, digitalID, productID); err != nil {
			return fmt.Errorf("deleting from digital_data: %w", err)
		}
	}

	return nil
}
