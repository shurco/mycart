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
	"github.com/shurco/mycart/pkg/slugify"
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
				product.has_variants,
				product.active,
				product.digital,
				EXISTS(SELECT 1 FROM digital_data WHERE digital_data.product_id = product.id AND digital_data.cart_id IS NULL) OR
				EXISTS(SELECT 1 FROM digital_file WHERE digital_file.product_id = product.id) AS digital_filled,
				(SELECT json_group_array(json_object('id', product_image.id, 'name', product_image.name, 'ext', product_image.ext)) as images FROM product_image WHERE product_id = product.id GROUP BY id LIMIT 1) as image,
				(SELECT json_group_array(json_object('id', product_variant.id, 'sku', product_variant.sku, 'option_values', json(product_variant.option_values))) FROM product_variant WHERE product_id = product.id AND active = 1) as variants,
				strftime('%s', created)
			FROM product
		`

	var queryPublic string
	var params []any
	var countParams []any

	if !private {
		queryPublic, params = publicProductFilter(cartID)
		countParams = append(countParams, params...)
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
		var image, variants, digitalType sql.NullString
		var digitalFilled, hasVariants sql.NullBool
		product := models.Product{}
		err := rows.Scan(
			&product.ID,
			&product.Name,
			&product.Brief,
			&product.Slug,
			&product.Amount,
			&hasVariants,
			&product.Active,
			&digitalType,
			&digitalFilled,
			&image,
			&variants,
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

		if hasVariants.Valid {
			product.HasVariants = hasVariants.Bool
		}

		if variants.Valid && variants.String != `[{"id":null}]` && variants.String != "[]" {
			if err := json.Unmarshal([]byte(variants.String), &product.Variants); err != nil {
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
				product.quantity,
				product.sku,
				product.has_variants,
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
			WHERE product.id = ?
			GROUP BY product.id`
	} else {
		query += `
			FROM product
			LEFT JOIN product_image pi ON product.id = pi.product_id
			WHERE product.slug = ? AND product.deleted = 0 AND product.active = 1
			GROUP BY product.id`
	}

	var images, metadata, attributes, digitalType, seo, sku sql.NullString
	var updated sql.NullInt64
	var digitalFilled, hasVariants sql.NullBool
	var quantity sql.NullInt64

	scanArgs := []any{
		&product.ID,
		&product.Name,
		&product.Brief,
		&product.Description,
		&product.Slug,
		&product.Amount,
		&quantity,
		&sku,
		&hasVariants,
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

	if quantity.Valid {
		product.Quantity = int(quantity.Int64)
	}

	if sku.Valid {
		product.SKU = sku.String
	}

	if hasVariants.Valid {
		product.HasVariants = hasVariants.Bool
	}

	// Load options and variants if product has variants
	if product.HasVariants {
		// Load all options first
		optionsQuery := `
			SELECT id, name, position
			FROM product_option
			WHERE product_id = ?
			ORDER BY position`

		optionRows, err := q.DB.QueryContext(ctx, optionsQuery, product.ID)
		if err != nil {
			return nil, err
		}

		optionIDs := []string{}
		optionMap := make(map[string]*models.ProductOption)

		for optionRows.Next() {
			option := models.ProductOption{ProductID: product.ID}
			if err := optionRows.Scan(&option.ID, &option.Name, &option.Position); err != nil {
				optionRows.Close()
				return nil, err
			}
			optionIDs = append(optionIDs, option.ID)
			optionMap[option.ID] = &option
			product.Options = append(product.Options, option)
		}
		optionRows.Close()

		// Load all option values
		if len(optionIDs) > 0 {
			valuesQuery := `
				SELECT id, option_id, value, position
				FROM product_option_value
				WHERE option_id IN (SELECT id FROM product_option WHERE product_id = ?)
				ORDER BY option_id, position`

			valuesRows, err := q.DB.QueryContext(ctx, valuesQuery, product.ID)
			if err != nil {
				return nil, err
			}

			for valuesRows.Next() {
				value := models.ProductOptionValue{}
				if err := valuesRows.Scan(&value.ID, &value.OptionID, &value.Value, &value.Position); err != nil {
					valuesRows.Close()
					return nil, err
				}

				// Find the option and append the value
				for i := range product.Options {
					if product.Options[i].ID == value.OptionID {
						product.Options[i].Values = append(product.Options[i].Values, value)
						break
					}
				}
			}
			valuesRows.Close()
		}

		// Load variants
		variantsQuery := `
			SELECT id, sku, price_surcharge, quantity, option_values, active
			FROM product_variant
			WHERE product_id = ? AND deleted = 0`

		variantRows, err := q.DB.QueryContext(ctx, variantsQuery, product.ID)
		if err != nil {
			return nil, err
		}

		for variantRows.Next() {
			variant := models.ProductVariant{ProductID: product.ID}
			var optionValues string
			var sku sql.NullString

			if err := variantRows.Scan(&variant.ID, &sku, &variant.PriceSurcharge, &variant.Quantity, &optionValues, &variant.Active); err != nil {
				variantRows.Close()
				return nil, err
			}

			if sku.Valid {
				variant.SKU = sku.String
			}

			if err := json.Unmarshal([]byte(optionValues), &variant.OptionValues); err != nil {
				variantRows.Close()
				return nil, err
			}

			product.Variants = append(product.Variants, variant)
		}
		variantRows.Close()
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
	// Marshal JSON fields
	metadata, attributes, seo, err := q.marshalProductJSON(product)
	if err != nil {
		return err
	}

	// Start transaction for atomic updates
	tx, err := q.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	// Update main product fields
	if err = q.updateProductMainFields(ctx, tx, product, metadata, attributes, seo); err != nil {
		return err
	}

	// Sync variant data
	if err = q.syncProductVariants(ctx, tx, product); err != nil {
		return err
	}

	return tx.Commit()
}

// marshalProductJSON marshals product metadata, attributes, and SEO to JSON
func (q *ProductQueries) marshalProductJSON(product *models.Product) (metadata, attributes, seo []byte, err error) {
	metadata, err = json.Marshal(product.Metadata)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("marshal metadata: %w", err)
	}

	attributes, err = json.Marshal(product.Attributes)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("marshal attributes: %w", err)
	}

	seo, err = json.Marshal(product.Seo)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("marshal seo: %w", err)
	}

	return metadata, attributes, seo, nil
}

// updateProductMainFields executes the UPDATE statement for main product fields
func (q *ProductQueries) updateProductMainFields(ctx context.Context, tx *sql.Tx, product *models.Product, metadata, attributes, seo []byte) error {
	stmt, err := tx.PrepareContext(ctx, `
		UPDATE product SET
			name = ?,
			brief = ?,
			desc = ?,
			slug = ?,
			amount = ?,
			quantity = ?,
			sku = ?,
			has_variants = ?,
			metadata = ?,
			attribute = ?,
			seo = ?,
			updated = datetime('now')
		WHERE id = ?
	`)
	if err != nil {
		return fmt.Errorf("prepare update statement: %w", err)
	}
	defer func() { _ = stmt.Close() }()

	// Handle empty SKU as NULL
	var productSKU sql.NullString
	if product.SKU != "" {
		productSKU = sql.NullString{String: product.SKU, Valid: true}
	}

	_, err = stmt.ExecContext(ctx,
		product.Name,
		product.Brief,
		product.Description,
		product.Slug,
		product.Amount,
		product.Quantity,
		productSKU,
		product.HasVariants,
		metadata,
		attributes,
		seo,
		product.ID,
	)
	if err != nil {
		return fmt.Errorf("execute update: %w", err)
	}

	return nil
}

// syncProductVariants handles all variant-related CRUD operations
func (q *ProductQueries) syncProductVariants(ctx context.Context, tx *sql.Tx, product *models.Product) error {
	if product.HasVariants {
		// Delete existing options (cascades to option values)
		_, err := tx.ExecContext(ctx, `DELETE FROM product_option WHERE product_id = ?`, product.ID)
		if err != nil {
			return fmt.Errorf("delete options: %w", err)
		}

		// Delete existing variants
		_, err = tx.ExecContext(ctx, `DELETE FROM product_variant WHERE product_id = ?`, product.ID)
		if err != nil {
			return fmt.Errorf("delete variants: %w", err)
		}

		// Insert new options and their values
		for _, option := range product.Options {
			optionID := security.RandomString()

			_, err = tx.ExecContext(ctx,
				`INSERT INTO product_option (id, product_id, name, position) VALUES (?, ?, ?, ?)`,
				optionID, product.ID, option.Name, option.Position,
			)
			if err != nil {
				return fmt.Errorf("insert option: %w", err)
			}

			// Insert option values
			for _, value := range option.Values {
				valueID := security.RandomString()
				_, err = tx.ExecContext(ctx,
					`INSERT INTO product_option_value (id, option_id, value, position) VALUES (?, ?, ?, ?)`,
					valueID, optionID, value.Value, value.Position,
				)
				if err != nil {
					return fmt.Errorf("insert option value: %w", err)
				}
			}
		}

		// Insert new variants
		for _, variant := range product.Variants {
			variantID := security.RandomString()

			// Marshal option_values map to JSON
			optionValuesJSON, err := json.Marshal(variant.OptionValues)
			if err != nil {
				return fmt.Errorf("marshal option values: %w", err)
			}

			// Handle empty SKU as NULL to avoid unique constraint violations
			var skuValue sql.NullString
			if variant.SKU != "" {
				skuValue = sql.NullString{String: variant.SKU, Valid: true}
			}

			_, err = tx.ExecContext(ctx,
				`INSERT INTO product_variant (id, product_id, sku, price_surcharge, quantity, option_values, active) VALUES (?, ?, ?, ?, ?, ?, ?)`,
				variantID, product.ID, skuValue, variant.PriceSurcharge, variant.Quantity, string(optionValuesJSON), variant.Active,
			)
			if err != nil {
				return fmt.Errorf("insert variant: %w", err)
			}
		}
	} else {
		// If has_variants is false, clean up any existing variant data
		_, err := tx.ExecContext(ctx, `DELETE FROM product_option WHERE product_id = ?`, product.ID)
		if err != nil {
			return fmt.Errorf("cleanup options: %w", err)
		}
		_, err = tx.ExecContext(ctx, `DELETE FROM product_variant WHERE product_id = ?`, product.ID)
		if err != nil {
			return fmt.Errorf("cleanup variants: %w", err)
		}
	}

	return nil
}

// DeleteProduct removes a product from the database based on its ID.
func (q *ProductQueries) DeleteProduct(ctx context.Context, id string) error {
	_, err := q.DB.ExecContext(ctx, `DELETE FROM product WHERE id = ?`, id)
	return err
}

// publicProductFilter builds the WHERE clause for storefront product queries.
// When cartID is set, purchased items remain visible even if the product was deactivated.
func publicProductFilter(cartID string) (string, []any) {
	if cartID != "" {
		return `
			WHERE product.deleted = 0 AND (
				product.active = 1
				OR EXISTS (
					SELECT 1 FROM digital_data
					WHERE digital_data.product_id = product.id
					AND digital_data.cart_id = ?
				)
			)
		`, []any{cartID}
	}
	return ` WHERE product.deleted = 0 AND product.active = 1 `, nil
}

// IsProduct checks if a product with the given slug exists and is active.
func (q *ProductQueries) IsProduct(ctx context.Context, slug string) bool {
	var exists bool
	query := `
			SELECT EXISTS (
				SELECT 1 FROM product
				WHERE product.slug = ? AND product.deleted = 0 AND product.active = 1
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

// AddProductWithVariants adds a product with its options and variants in a transaction
func (q *ProductQueries) AddProductWithVariants(ctx context.Context, product *models.Product) (*models.Product, error) {
	tx, err := q.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	// 1. Insert product
	query := `
		INSERT INTO product (
			id, name, brief, desc, slug, amount, quantity, sku,
			has_variants, metadata, attribute, digital, active
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	metadata, err := json.Marshal(product.Metadata)
	if err != nil {
		return nil, fmt.Errorf("marshal metadata: %w", err)
	}

	attributes, err := json.Marshal(product.Attributes)
	if err != nil {
		return nil, fmt.Errorf("marshal attributes: %w", err)
	}

	_, err = tx.ExecContext(ctx, query,
		product.ID,
		product.Name,
		product.Brief,
		product.Description,
		product.Slug,
		product.Amount,
		product.Quantity,
		product.SKU,
		product.HasVariants,
		string(metadata),
		string(attributes),
		product.Digital.Type,
		product.Active,
	)
	if err != nil {
		return nil, fmt.Errorf("insert product: %w", err)
	}

	// 2. Insert product images
	if err = q.insertProductImages(ctx, tx, product.ID, product.Images); err != nil {
		return nil, err
	}

	// 3. Insert options and option values
	if err = q.insertProductOptions(ctx, tx, product.ID, product.Options); err != nil {
		return nil, err
	}

	// 4. Insert variants with relationships and images
	if err = q.insertProductVariants(ctx, tx, product); err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	return product, nil
}

// insertProductImages inserts all product images within a transaction
func (q *ProductQueries) insertProductImages(ctx context.Context, tx *sql.Tx, productID string, images []models.File) error {
	for i, img := range images {
		_, err := tx.ExecContext(ctx, `
			INSERT INTO product_image (id, product_id, name, ext, orig_name, position)
			VALUES (?, ?, ?, ?, ?, ?)
		`, img.ID, productID, img.Name, img.Ext, img.OrigName, i)
		if err != nil {
			return fmt.Errorf("insert image %d: %w", i, err)
		}
	}
	return nil
}

// insertProductOptions inserts options and their values within a transaction
func (q *ProductQueries) insertProductOptions(ctx context.Context, tx *sql.Tx, productID string, options []models.ProductOption) error {
	for _, option := range options {
		_, err := tx.ExecContext(ctx, `
			INSERT INTO product_option (id, product_id, name, position)
			VALUES (?, ?, ?, ?)
		`, option.ID, productID, option.Name, option.Position)
		if err != nil {
			return fmt.Errorf("insert option %s: %w", option.Name, err)
		}

		for _, value := range option.Values {
			_, err = tx.ExecContext(ctx, `
				INSERT INTO product_option_value (id, option_id, value, position)
				VALUES (?, ?, ?, ?)
			`, value.ID, option.ID, value.Value, value.Position)
			if err != nil {
				return fmt.Errorf("insert option value %s: %w", value.Value, err)
			}
		}
	}
	return nil
}

// insertProductVariants inserts variants with relationships and images within a transaction
func (q *ProductQueries) insertProductVariants(ctx context.Context, tx *sql.Tx, product *models.Product) error {
	for _, variant := range product.Variants {
		// Marshal option values to JSON
		optionValuesJSON, err := json.Marshal(variant.OptionValues)
		if err != nil {
			return fmt.Errorf("marshal option values: %w", err)
		}

		_, err = tx.ExecContext(ctx, `
			INSERT INTO product_variant (
				id, product_id, sku, price_surcharge, quantity, option_values, active
			) VALUES (?, ?, ?, ?, ?, ?, ?)
		`, variant.ID, product.ID, variant.SKU, variant.PriceSurcharge, variant.Quantity, string(optionValuesJSON), variant.Active)
		if err != nil {
			return fmt.Errorf("insert variant %s: %w", variant.SKU, err)
		}

		// Insert variant-option relationships
		for optionName, optionValue := range variant.OptionValues {
			// Find the option value ID
			var optionValueID string
			err = tx.QueryRowContext(ctx, `
				SELECT pov.id
				FROM product_option_value pov
				JOIN product_option po ON pov.option_id = po.id
				WHERE po.product_id = ? AND po.name = ? AND pov.value = ?
			`, product.ID, optionName, optionValue).Scan(&optionValueID)
			if err != nil {
				return fmt.Errorf("find option value ID for %s=%s: %w", optionName, optionValue, err)
			}

			_, err = tx.ExecContext(ctx, `
				INSERT INTO product_variant_option (variant_id, option_value_id)
				VALUES (?, ?)
			`, variant.ID, optionValueID)
			if err != nil {
				return fmt.Errorf("insert variant-option relationship: %w", err)
			}
		}

		// Insert variant images
		for i, img := range variant.Images {
			_, err = tx.ExecContext(ctx, `
				INSERT INTO product_variant_image (id, variant_id, name, ext, orig_name, position)
				VALUES (?, ?, ?, ?, ?, ?)
			`, img.ID, variant.ID, img.Name, img.Ext, img.OrigName, i)
			if err != nil {
				return fmt.Errorf("insert variant image %d: %w", i, err)
			}
		}
	}
	return nil
}

// GetProductWithVariants retrieves a product with all its options and variants
func (q *ProductQueries) GetProductWithVariants(ctx context.Context, productID string) (*models.Product, error) {
	product := &models.Product{}

	// 1. Get product base data
	query := `
		SELECT id, name, brief, desc, slug, amount, quantity, sku,
		       has_variants, metadata, attribute, digital, active
		FROM product
		WHERE id = ? AND deleted = FALSE
	`

	var metadata, attributes string
	var digitalType sql.NullString

	err := q.DB.QueryRowContext(ctx, query, productID).Scan(
		&product.ID,
		&product.Name,
		&product.Brief,
		&product.Description,
		&product.Slug,
		&product.Amount,
		&product.Quantity,
		&product.SKU,
		&product.HasVariants,
		&metadata,
		&attributes,
		&digitalType,
		&product.Active,
	)
	if err != nil {
		return nil, fmt.Errorf("query product: %w", err)
	}

	// Parse JSON fields
	if err = json.Unmarshal([]byte(metadata), &product.Metadata); err != nil {
		return nil, fmt.Errorf("unmarshal metadata: %w", err)
	}
	if err = json.Unmarshal([]byte(attributes), &product.Attributes); err != nil {
		return nil, fmt.Errorf("unmarshal attributes: %w", err)
	}
	if digitalType.Valid {
		product.Digital.Type = digitalType.String
	}

	// 2. Get product images
	imageRows, err := q.DB.QueryContext(ctx, `
		SELECT id, name, ext, orig_name
		FROM product_image
		WHERE product_id = ?
		ORDER BY position
	`, productID)
	if err != nil {
		return nil, fmt.Errorf("query product images: %w", err)
	}
	defer imageRows.Close()

	for imageRows.Next() {
		var img models.File
		if err := imageRows.Scan(&img.ID, &img.Name, &img.Ext, &img.OrigName); err != nil {
			return nil, fmt.Errorf("scan product image: %w", err)
		}
		product.Images = append(product.Images, img)
	}

	// Skip options/variants if product doesn't have variants
	if !product.HasVariants {
		return product, nil
	}

	// Continue in next part...
	return q.loadProductOptions(ctx, product)
}

// loadProductOptions loads options, option values, and variants for a product
func (q *ProductQueries) loadProductOptions(ctx context.Context, product *models.Product) (*models.Product, error) {
	// Get options
	optionRows, err := q.DB.QueryContext(ctx, `
		SELECT id, name, position
		FROM product_option
		WHERE product_id = ?
		ORDER BY position
	`, product.ID)
	if err != nil {
		return nil, fmt.Errorf("query options: %w", err)
	}
	defer optionRows.Close()

	for optionRows.Next() {
		option := models.ProductOption{ProductID: product.ID}
		if err := optionRows.Scan(&option.ID, &option.Name, &option.Position); err != nil {
			return nil, fmt.Errorf("scan option: %w", err)
		}
		product.Options = append(product.Options, option)
	}

	// Load option values
	if err = q.loadOptionValues(ctx, &product.Options); err != nil {
		return nil, err
	}

	// Load variants with their data
	variants, err := q.loadProductVariants(ctx, product.ID)
	if err != nil {
		return nil, err
	}
	product.Variants = variants

	return product, nil
}

// loadOptionValues loads values for all product options
func (q *ProductQueries) loadOptionValues(ctx context.Context, options *[]models.ProductOption) error {
	for i := range *options {
		valueRows, err := q.DB.QueryContext(ctx, `
			SELECT id, value, position
			FROM product_option_value
			WHERE option_id = ?
			ORDER BY position
		`, (*options)[i].ID)
		if err != nil {
			return fmt.Errorf("query option values: %w", err)
		}

		for valueRows.Next() {
			value := models.ProductOptionValue{OptionID: (*options)[i].ID}
			if err := valueRows.Scan(&value.ID, &value.Value, &value.Position); err != nil {
				valueRows.Close()
				return fmt.Errorf("scan option value: %w", err)
			}
			(*options)[i].Values = append((*options)[i].Values, value)
		}
		valueRows.Close()
	}
	return nil
}

// loadProductVariants loads all variants with their option values and images
func (q *ProductQueries) loadProductVariants(ctx context.Context, productID string) ([]models.ProductVariant, error) {
	variantRows, err := q.DB.QueryContext(ctx, `
		SELECT id, sku, price_surcharge, quantity, active
		FROM product_variant
		WHERE product_id = ?
	`, productID)
	if err != nil {
		return nil, fmt.Errorf("query variants: %w", err)
	}
	defer variantRows.Close()

	var variants []models.ProductVariant

	for variantRows.Next() {
		variant := models.ProductVariant{
			ProductID:    productID,
			OptionValues: make(map[string]string),
		}

		var sku sql.NullString
		if err := variantRows.Scan(&variant.ID, &sku, &variant.PriceSurcharge, &variant.Quantity, &variant.Active); err != nil {
			return nil, fmt.Errorf("scan variant: %w", err)
		}
		if sku.Valid {
			variant.SKU = sku.String
		}

		// Get variant option values
		optValueRows, err := q.DB.QueryContext(ctx, `
			SELECT po.name, pov.value
			FROM product_variant_option pvo
			JOIN product_option_value pov ON pvo.option_value_id = pov.id
			JOIN product_option po ON pov.option_id = po.id
			WHERE pvo.variant_id = ?
		`, variant.ID)
		if err != nil {
			return nil, fmt.Errorf("query variant option values: %w", err)
		}

		for optValueRows.Next() {
			var optionName, optionValue string
			if err := optValueRows.Scan(&optionName, &optionValue); err != nil {
				optValueRows.Close()
				return nil, fmt.Errorf("scan variant option value: %w", err)
			}
			variant.OptionValues[optionName] = optionValue
		}
		optValueRows.Close()

		// Get variant images
		imgRows, err := q.DB.QueryContext(ctx, `
			SELECT id, name, ext, orig_name
			FROM product_variant_image
			WHERE variant_id = ?
			ORDER BY position
		`, variant.ID)
		if err != nil {
			return nil, fmt.Errorf("query variant images: %w", err)
		}

		for imgRows.Next() {
			var img models.File
			if err := imgRows.Scan(&img.ID, &img.Name, &img.Ext, &img.OrigName); err != nil {
				imgRows.Close()
				return nil, fmt.Errorf("scan variant image: %w", err)
			}
			variant.Images = append(variant.Images, img)
		}
		imgRows.Close()

		variants = append(variants, variant)
	}

	return variants, nil
}

// GenerateUniqueSlug generates a unique URL-friendly slug from product name
func (q *ProductQueries) GenerateUniqueSlug(ctx context.Context, name string, excludeProductID string) (string, error) {
	service := slugify.NewSlugService(q.DB)
	return service.Generate(ctx, name, excludeProductID)
}
