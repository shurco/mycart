package queries

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/shurco/mycart/internal/models"
	"github.com/shurco/mycart/pkg/errors"
	"github.com/shurco/mycart/pkg/security"
)

// ProductQueries is a struct that embeds a pointer to an sql.DB.
// This allows for direct access to the database methods via the ProductQueries struct,
// effectively extending it with all the functionality of *sql.DB.
type ProductQueries struct {
	*sql.DB
}

// ListProducts retrieves a list of products from the database.
// If cartID is provided, it will also include digital products that were purchased in that cart.
func (q *ProductQueries) ListProducts(ctx context.Context, private bool, limit, offset int, cartID string, idList ...models.CartProduct) (*models.Products, error) {
	currency, err := db.GetSettingByKey(ctx, "currency")
	if err != nil {
		return nil, err
	}

	products := &models.Products{
		Currency: currency["currency"].Value.(string),
	}

	query := `
			SELECT DISTINCT
			  product.id,
				product.name,
				product.brief,
				product.slug,
				product.amount,
				product.active,
				product.digital,
				EXISTS(SELECT 1 FROM digital_data WHERE digital_data.product_id = product.id AND digital_data.cart_id IS NULL) OR
				EXISTS(SELECT 1 FROM digital_file WHERE digital_file.product_id = product.id) AS digital_filled,
				(SELECT json_group_array(json_object('id', product_image.id, 'name', product_image.name, 'ext', product_image.ext)) as images FROM product_image WHERE product_id = product.id GROUP BY id LIMIT 1) as image,
				strftime('%s', created)
			FROM product
		`

	// For public queries, filter by available digital products
	// If cartID is provided, also include products purchased in that cart
	var queryPublic string
	var params []any
	var countParams []any

	if !private {
		if cartID != "" {
			// Include products with available digital data OR products purchased in this cart
			queryPublic = ` 
				LEFT JOIN digital_data ON digital_data.product_id = product.id
				LEFT JOIN digital_file ON digital_file.product_id = product.id
				WHERE (
					(digital_data.content IS NOT NULL AND digital_data.cart_id IS NULL) OR 
					(digital_data.content IS NOT NULL AND digital_data.cart_id = ?) OR
					digital_file.orig_name IS NOT NULL
				) 
				AND product.deleted = 0 AND product.active = 1
			`
			params = append(params, cartID)
			countParams = append(countParams, cartID)
		} else {
			// Only include products with available digital data
			queryPublic = ` 
				LEFT JOIN digital_data ON digital_data.product_id = product.id
				LEFT JOIN digital_file ON digital_file.product_id = product.id
				WHERE (
					(digital_data.content IS NOT NULL AND digital_data.cart_id IS NULL) OR
					digital_file.orig_name IS NOT NULL
				) 
				AND product.deleted = 0 AND product.active = 1
			`
		}
	}

	var queryAddon string

	if len(idList) > 0 {
		for _, item := range idList {
			params = append(params, item.ProductID)
			countParams = append(countParams, item.ProductID)
		}
		queryAddon = fmt.Sprintf("AND product.id IN (%s)", strings.Repeat("?, ", len(idList)-1)+"?")
	}

	if !private {
		query += queryPublic
	}

	// Add pagination
	if limit > 0 {
		query += " LIMIT ?"
		params = append(params, limit)
		if offset > 0 {
			query += " OFFSET ?"
			params = append(params, offset)
		}
	}

	rows, err := q.DB.QueryContext(ctx, query+queryAddon, params...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var image, digitalType sql.NullString
		var digitalFilled sql.NullBool
		product := models.Product{}
		err := rows.Scan(
			&product.ID,
			&product.Name,
			&product.Brief,
			&product.Slug,
			&product.Amount,
			&product.Active,
			&digitalType,
			&digitalFilled,
			&image,
			&product.Created,
		)
		if err != nil {
			return nil, err
		}

		if image.Valid && image.String != `[{"id":null,"name":null,"ext":null}]` {
			if err := json.Unmarshal([]byte(image.String), &product.Images); err != nil {
				return nil, err
			}
		}

		product.Digital.Type = digitalType.String
		if private && digitalType.Valid {
			if digitalFilled.Valid {
				product.Digital.Filled = digitalFilled.Bool
			} else {
				product.Digital.Filled = false
			}
		}

		products.Products = append(products.Products, product)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Count total records (without pagination params)
	countQuery := `SELECT COUNT(DISTINCT product.id) FROM product`
	if !private {
		countQuery += queryPublic
	}
	err = q.DB.QueryRowContext(ctx, countQuery+queryAddon, countParams...).Scan(&products.Total)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	return products, nil
}

// Product retrieves a product by its ID, with the option to fetch private or public data.
func (q *ProductQueries) Product(ctx context.Context, private bool, id string) (*models.Product, error) {
	product := &models.Product{}

	query := `
			SELECT DISTINCT
				product.id,
				product.name, 
				product.brief,
				product.desc, 
				product.slug, 
				product.amount,
				product.active,
				product.metadata, 
				product.attribute, 
				product.digital,
				product.seo, 
				json_group_array(json_object('id', pi.id, 'name', pi.name, 'ext', pi.ext)) as images,
				strftime('%s', product.created), 
				strftime('%s', product.updated)
	`

	// Добавляем вычисление digital_filled для приватных запросов
	if private {
		query += `, EXISTS(SELECT 1 FROM digital_data WHERE digital_data.product_id = product.id AND digital_data.cart_id IS NULL) OR
				EXISTS(SELECT 1 FROM digital_file WHERE digital_file.product_id = product.id) AS digital_filled
			FROM product 
			LEFT JOIN product_image pi ON product.id = pi.product_id
			WHERE product.id = ?`
	} else {
		query += `
			FROM product 
			LEFT JOIN product_image pi ON product.id = pi.product_id
			LEFT JOIN digital_data ON digital_data.product_id = product.id   
			LEFT JOIN digital_file ON digital_file.product_id = product.id 
			WHERE (digital_data.content IS NOT NULL AND digital_data.cart_id IS NULL OR digital_file.orig_name IS NOT NULL) AND
			product.slug = ? AND product.active = 1`
	}

	var images, metadata, attributes, digitalType, seo sql.NullString
	var updated sql.NullInt64
	var digitalFilled sql.NullBool

	scanArgs := []any{
		&product.ID,
		&product.Name,
		&product.Brief,
		&product.Description,
		&product.Slug,
		&product.Amount,
		&product.Active,
		&metadata,
		&attributes,
		&digitalType,
		&seo,
		&images,
		&product.Created,
		&updated,
	}

	// Добавляем digital_filled в scanArgs для приватных запросов
	if private {
		scanArgs = append(scanArgs, &digitalFilled)
	}

	err := q.DB.QueryRowContext(ctx, query, id).Scan(scanArgs...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.ErrProductNotFound
		}
		return nil, err
	}

	if updated.Valid {
		product.Updated = updated.Int64
	}

	if images.Valid && images.String != `[{"id":null,"name":null,"ext":null}]` {
		if err := json.Unmarshal([]byte(images.String), &product.Images); err != nil {
			return nil, err
		}
	}

	if attributes.Valid {
		if err := json.Unmarshal([]byte(attributes.String), &product.Attributes); err != nil {
			return nil, err
		}
	}

	if metadata.Valid {
		if err := json.Unmarshal([]byte(metadata.String), &product.Metadata); err != nil {
			return nil, err
		}
	}

	product.Digital.Type = digitalType.String

	// Устанавливаем digital.filled для приватных запросов
	if private && digitalType.Valid {
		if digitalFilled.Valid {
			product.Digital.Filled = digitalFilled.Bool
		} else {
			product.Digital.Filled = false
		}
	}

	if seo.Valid {
		if err := json.Unmarshal([]byte(seo.String), &product.Seo); err != nil {
			return nil, err
		}
	}

	return product, nil
}

// AddProduct inserts a new product into the database and returns the product with the created timestamp.
func (q *ProductQueries) AddProduct(ctx context.Context, product *models.Product) (*models.Product, error) {
	product.ID = security.RandomString()

	metadata, err := json.Marshal(product.Metadata)
	if err != nil {
		return nil, err
	}

	attributes, err := json.Marshal(product.Attributes)
	if err != nil {
		return nil, err
	}

	query := `
			INSERT INTO product (
					id, name, amount, slug, metadata, attribute, brief, desc, digital, active
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, FALSE)
			RETURNING strftime('%s', created)
	`
	stmt, err := q.DB.PrepareContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer func() { _ = stmt.Close() }()

	err = stmt.QueryRowContext(ctx,
		product.ID, product.Name, product.Amount, product.Slug,
		metadata, attributes, product.Brief, product.Description, product.Digital.Type,
	).Scan(&product.Created)
	if err != nil {
		return nil, err
	}

	return product, nil
}

// UpdateProduct updates an existing product in the database with new values.
func (q *ProductQueries) UpdateProduct(ctx context.Context, product *models.Product) error {
	metadata, err := json.Marshal(product.Metadata)
	if err != nil {
		return err
	}

	attributes, err := json.Marshal(product.Attributes)
	if err != nil {
		return err
	}

	seo, err := json.Marshal(product.Seo)
	if err != nil {
		return err
	}

	stmt, err := q.DB.PrepareContext(ctx, `
			UPDATE product SET 
				name = ?, 
				brief = ?, 
				desc = ?, 
				slug = ?, 
				amount = ?, 
				metadata = ?, 
				attribute = ?, 
				seo = ?, 
				updated = datetime('now') 
			WHERE id = ?
		`)
	if err != nil {
		return err
	}
	defer func() { _ = stmt.Close() }()

	_, err = stmt.ExecContext(ctx,
		product.Name,
		product.Brief,
		product.Description,
		product.Slug,
		product.Amount,
		metadata,
		attributes,
		seo,
		product.ID,
	)
	return err
}

// DeleteProduct removes a product from the database based on its ID.
func (q *ProductQueries) DeleteProduct(ctx context.Context, id string) error {
	_, err := q.DB.ExecContext(ctx, `DELETE FROM product WHERE id = ?`, id)
	return err
}

// IsProduct checks if a product with the given slug exists and is active,
// and also has associated digital data or file that meets certain conditions.
func (q *ProductQueries) IsProduct(ctx context.Context, slug string) bool {
	var exists bool
	query := `
			SELECT EXISTS (
				SELECT 1 FROM product 
				WHERE product.slug = ? AND product.active = 1 AND (
					EXISTS (
						SELECT 1 FROM digital_data 
						WHERE digital_data.product_id = product.id 
						AND digital_data.content IS NOT NULL 
						AND digital_data.cart_id IS NULL
					) OR EXISTS (
						SELECT 1 FROM digital_file 
						WHERE digital_file.product_id = product.id 
						AND digital_file.orig_name IS NOT NULL
					)
				)
			)
	`
	err := q.DB.QueryRowContext(ctx, query, slug).Scan(&exists)
	return err == nil && exists
}

// UpdateActive toggles the 'active' status of a product and updates its 'updated' timestamp.
// It takes a context and an ID as arguments, and returns an error if the operation fails.
func (q *ProductQueries) UpdateActive(ctx context.Context, id string) error {
	query := `UPDATE product SET active = NOT active, updated = datetime('now') WHERE id = ?`
	_, err := q.DB.ExecContext(ctx, query, id)
	return err
}

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

	var digitalType sql.NullString
	for rows.Next() {
		var fileID, fileName, fileExt sql.NullString
		var dataID, dataContent, cartID sql.NullString

		err := rows.Scan(
			&digitalType,
			&fileID, &fileName, &fileExt,
			&dataID, &dataContent, &cartID,
		)
		if err != nil {
			return nil, err
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
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return digital, nil
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
	case "file":
		query = `DELETE FROM digital_file WHERE id = ? AND product_id = ?`
		if _, err := q.DB.ExecContext(ctx, query, digitalID, productID); err != nil {
			return fmt.Errorf("deleting from digital_file: %w", err)
		}

		filePath := fmt.Sprintf("./lc_digitals/%s.%s", name.String, ext.String)
		if err := os.Remove(filePath); err != nil {
			return fmt.Errorf("failed to remove file %s: %w", filePath, err)
		}

	case "data":
		query = `DELETE FROM digital_data WHERE id = ? AND product_id = ?`
		if _, err := q.DB.ExecContext(ctx, query, digitalID, productID); err != nil {
			return fmt.Errorf("deleting from digital_data: %w", err)
		}
	}

	return nil
}
