# Complexity Refactoring Design

**Date:** 2026-07-22  
**Goal:** Reduce cyclomatic complexity of 7 methods to under 15 to pass PR checks  
**Approach:** Extract helper methods within existing files/structs

## Overview

Seven methods across the codebase have cyclomatic complexity between 16-21, exceeding the PR check threshold of 15. This refactoring will decompose each complex method into 2-4 focused helper methods using the extract method pattern.

**Current complexity:**
- Go: `UpdateProduct` (21), `AddProductWithVariants` (19), `loadProductOptions` (17)
- Svelte: `openEdit` (17), `handleSubmit` (19)
- Tests: `TestIntegration_ProductWithVariants_FullLifecycle` (16), `TestIntegration_CSV_ImportExport_Roundtrip` (16)

**Target:** All methods under 15

## Architecture

### Refactoring Strategy

Each complex method will be decomposed into 2-4 focused helper methods that handle one specific responsibility. Helpers stay in the same file/struct, use the same transaction context when applicable, and maintain the existing error handling pattern.

**Complexity reduction mechanism:**
- Each `if` statement and loop adds to cyclomatic complexity
- Extract logical blocks (variant handling, option processing, data transformation) into helpers
- Each helper reduces parent method's complexity by its own branch/loop count

**Naming conventions:**
- Go: lowercase private methods like `updateVariantData()`, `insertOptionValues()`
- Svelte: camelCase functions like `validateProductForm()`, `convertFormToSubmitData()`
- Tests: camelCase helper functions like `createTestProductWithVariants()`

### Code Organization

**Go helpers:**
- Placed immediately after the parent method in the same file
- Belong to the same struct (e.g., `ProductQueries`)
- Accept `ctx context.Context` and `tx *sql.Tx` (or `*sql.DB`)
- Return `error` as last return value

**Svelte helpers:**
- Placed above the component `<script>` block as module-level functions
- Accept necessary parameters explicitly
- Return transformed data or error objects

**Test helpers:**
- Placed immediately after the parent test function
- Accept `*testing.T` as first parameter
- Call `t.Fatalf()` or `t.Errorf()` directly for assertions

## Go Backend Refactoring

### 1. UpdateProduct (complexity 21 → ~10)

**Current structure:**
- Marshal 3 JSON fields (metadata, attributes, seo)
- Begin transaction
- Prepare and execute UPDATE statement with NULL SKU handling
- If product has variants: delete old variants, insert new options/values/variants
- Else: clean up variant data
- Commit transaction

**Refactored structure:**

Main method `UpdateProduct`:
- Call `marshalProductJSON()` to get JSON strings
- Begin transaction with defer rollback
- Call `updateProductMainFields()` to update product table
- Call `syncProductVariants()` to handle variant operations
- Commit transaction

Helper methods:

**`marshalProductJSON(product *models.Product) (metadata, attributes, seo []byte, err error)`**
- Marshals the 3 JSON fields
- Returns all 3 or first error encountered
- Reduces complexity by 3 (3 error checks)

**`updateProductMainFields(ctx context.Context, tx *sql.Tx, product *models.Product, metadata, attributes, seo []byte) error`**
- Prepares and executes the UPDATE statement
- Handles NULL SKU conversion
- Returns error if update fails
- Reduces complexity by 1 (SKU NULL handling)

**`syncProductVariants(ctx context.Context, tx *sql.Tx, product *models.Product) error`**
- If has_variants: delete old options/variants, insert new ones
- Else: clean up all variant data
- Contains all the variant CRUD logic
- Reduces complexity by 6+ (main if/else + nested loops)

**Complexity reduction:** 21 → 10 (removes 11 decision points)

### 2. AddProductWithVariants (complexity 19 → ~10)

**Current structure:**
- Begin transaction
- Insert product with marshaled JSON
- Loop: insert product images
- Nested loops: insert options and their values
- Nested loops: insert variants, find option value IDs, insert variant-option relationships, insert variant images
- Commit transaction

**Refactored structure:**

Main method `AddProductWithVariants`:
- Begin transaction with defer rollback
- Marshal and insert product
- Call `insertProductImages()`
- Call `insertProductOptions()`
- Call `insertProductVariants()`
- Commit transaction

Helper methods:

**`insertProductImages(ctx context.Context, tx *sql.Tx, productID string, images []models.File) error`**
- Loops through images and inserts each
- Returns error on failure
- Reduces complexity by 1 (loop)

**`insertProductOptions(ctx context.Context, tx *sql.Tx, productID string, options []models.ProductOption) error`**
- Loops through options
- For each option, loops through values
- Inserts options and their values
- Reduces complexity by 2 (nested loops)

**`insertProductVariants(ctx context.Context, tx *sql.Tx, product *models.Product) error`**
- Loops through variants
- For each variant: marshals option values, inserts variant
- Loops through option values to create relationships
- Loops through variant images
- Reduces complexity by 4+ (multiple nested loops)

**Complexity reduction:** 19 → 10 (removes 9 decision points)

### 3. loadProductOptions (complexity 17 → ~12)

**Current structure:**
- Query and loop through options
- For each option, query and loop through option values
- Query and loop through variants
- For each variant: query and loop through option values, query and loop through images

**Refactored structure:**

Main method `loadProductOptions`:
- Query and loop through options, build optionMap
- Call `loadOptionValues()` to populate option.Values
- Call `loadProductVariants()` to load all variant data

Helper methods:

**`loadOptionValues(ctx context.Context, db *sql.DB, options []models.ProductOption) error`**
- Loops through options
- For each option, queries and loads its values
- Reduces complexity by 2 (loop + nested query loop)

**`loadProductVariants(ctx context.Context, db *sql.DB, productID string) ([]models.ProductVariant, error)`**
- Queries and loops through variants
- For each variant: loads option values (nested query/loop), loads images (nested query/loop)
- Returns populated variants
- Reduces complexity by 4+ (multiple nested loops)

**Complexity reduction:** 17 → 12 (removes 5 decision points)

## Svelte Frontend Refactoring

### 4. openEdit (complexity 17 → ~8)

**Current structure:**
- Set drawer state
- Load product from API
- If result: convert price, populate all form fields, set images, open drawer

**Refactored structure:**

Main function `openEdit`:
- Set initial drawer state
- Load product from API
- If result: call `convertProductToFormData()`, call `populateEditDrawer()`, open drawer

Helper functions:

**`convertProductToFormData(product: Product): FormData`**
- Handles price conversion from cents to regular number
- Extracts and formats all product fields
- Returns populated FormData object
- Reduces complexity by converting nested property access and calculations

**`populateEditDrawer(formData: FormData, fullProduct: Product): void`**
- Sets fullProductData
- Sets formData
- Sets amountDisplay
- Sets productImages
- Encapsulates all drawer state updates

**Complexity reduction:** 17 → 8 (removes 9 decision points from nested property handling)

### 5. handleSubmit (complexity 19 → ~10)

**Current structure:**
- Validate fields (name, slug, amount, digital type)
- Check if errors, return early
- Determine add vs edit mode
- Convert amount to cents
- Serialize Svelte proxies to plain objects
- Submit via API
- If success: update list or reload
- Else if update failed: reload from API

**Refactored structure:**

Main function `handleSubmit`:
- Call `validateProductForm()`, return if errors
- Determine mode and URL
- Call `prepareSubmitData()` to get submit payload
- Submit via API
- Call `handleSubmitResponse()` with result

Helper functions:

**`validateProductForm(formData: FormData, drawerMode: string): FieldErrors`**
- Validates name, slug, amount, digital type
- Returns error object (empty if valid)
- Reduces complexity by 4 (multiple conditional validations)

**`prepareSubmitData(formData: FormData, amountValue: number): Partial<Product>`**
- Converts amount to cents
- Serializes Svelte proxies (options, variants)
- Returns API-ready payload
- Reduces complexity by removing data transformation logic

**`handleSubmitResponse(result: Product | null, isUpdate: boolean, drawerProduct: DrawerProduct | null): Promise<void>`**
- If success: updates list or reloads products
- If update failed: reloads product from API
- Handles all post-submit UI updates
- Reduces complexity by 3 (conditional branches)

**Complexity reduction:** 19 → 10 (removes 9 decision points)

## Test Refactoring

### 6. TestIntegration_ProductWithVariants_FullLifecycle (complexity 16 → ~12)

**Current structure:**
- Register routes
- Create product with variants via API
- Parse response and extract product ID
- Retrieve product via API
- Assert product name, has_variants, option count, variant count
- Assert option structure and values
- Loop through variants to find specific one and assert its fields

**Refactored structure:**

Main test function:
- Register routes
- Call `createTestProductWithVariants()` to get product ID
- Retrieve product via API
- Call `verifyProductVariantData()` to run all assertions

Helper functions:

**`createTestProductWithVariants(t *testing.T, app *fiber.App, cookie string) string`**
- Executes POST request with test payload
- Parses response
- Asserts status and extracts product ID
- Returns product ID or fails test
- Reduces complexity by 2

**`verifyProductVariantData(t *testing.T, product models.Product)`**
- Asserts all product fields
- Checks options structure
- Checks variants structure
- Finds and verifies specific variant
- Reduces complexity by 4 (multiple assertions and loop)

**Complexity reduction:** 16 → 12 (removes 4 decision points)

### 7. TestIntegration_CSV_ImportExport_Roundtrip (complexity 16 → ~12)

**Current structure:**
- Register routes
- Create CSV content
- Create multipart form for preview
- Execute preview request and verify response
- Create second multipart form for import
- Execute import request and verify response
- Execute export request
- Verify export CSV content

**Refactored structure:**

Main test function:
- Register routes
- Define CSV content
- Call `executeCSVPreview()` and verify
- Call `executeCSVImport()` and verify
- Call `executeCSVExport()` and verify

Helper functions:

**`createMultipartCSVRequest(t *testing.T, csvContent string, filename string) (*bytes.Buffer, string)`**
- Creates multipart writer
- Writes CSV file part
- Returns body buffer and content type
- Reduces complexity by 2 (used twice, eliminates duplication)

**`executeCSVExport(t *testing.T, app *fiber.App, cookie string) string`**
- Executes export request
- Asserts status
- Reads and returns CSV content
- Reduces complexity by 2

**Complexity reduction:** 16 → 12 (removes 4 decision points through deduplication)

## Error Handling

All helper methods follow existing error handling patterns:

### Go Helpers

- Return `error` as last return value
- Wrap errors with descriptive context: `fmt.Errorf("insert options: %w", err)`
- Parent method checks returned errors and either returns immediately or rolls back transaction
- Transaction rollback stays in parent's `defer` block
- No change to error propagation behavior

Example:
```go
if err := insertProductOptions(ctx, tx, product.ID, product.Options); err != nil {
    return fmt.Errorf("insert options: %w", err)
}
```

### Svelte Helpers

- Validation functions return error objects: `Record<string, string>` (field → error message)
- Data transformation functions return transformed data directly
- API call wrappers handle errors internally using existing `loadData`/`saveData` utilities
- Parent function checks for errors/null before proceeding

Example:
```typescript
const errors = validateProductForm(formData, drawerMode)
if (Object.keys(errors).length > 0) {
    formErrors = errors
    return
}
```

### Test Helpers

- Call `t.Fatalf()` for critical failures (setup, parsing)
- Call `t.Errorf()` for assertion failures
- No error return values needed - test helpers use `*testing.T` directly
- Parent test continues or stops based on Fatal vs Error calls

Example:
```go
func createTestProductWithVariants(t *testing.T, ...) string {
    resp := testutil.DoRequest(...)
    if err := json.Decode(...); err != nil {
        t.Fatalf("Failed to parse response: %v", err)
    }
    return productID
}
```

## Testing Strategy

### Verification Approach

This is a pure refactoring with no behavior changes - only structural improvements. All existing tests must pass without modification (except the 2 integration tests we're refactoring internally).

### Test Execution Plan

**Before refactoring:**
Run existing tests to establish baseline:
```bash
# Integration tests
go test ./internal/integration/product_variants_test.go -v

# Unit tests for queries package
go test ./internal/queries -v

# Frontend tests (if any)
cd web/admin && npm test
```

**During refactoring:**
- Refactor one method at a time in this order:
  1. Go: `marshalProductJSON`, `updateProductMainFields`, `syncProductVariants`, then `UpdateProduct`
  2. Go: `insertProductImages`, `insertProductOptions`, `insertProductVariants`, then `AddProductWithVariants`
  3. Go: `loadOptionValues`, `loadProductVariants`, then `loadProductOptions`
  4. Svelte: `convertProductToFormData`, `populateEditDrawer`, then `openEdit`
  5. Svelte: `validateProductForm`, `prepareSubmitData`, `handleSubmitResponse`, then `handleSubmit`
  6. Tests: `createTestProductWithVariants`, `verifyProductVariantData`, then update test
  7. Tests: `createMultipartCSVRequest`, `executeCSVExport`, then update test

- Run tests after each parent method refactoring
- If tests fail, fix immediately before proceeding

**After refactoring:**
- Run full test suite
- Verify complexity with PR check tool:
  ```bash
  # Use whatever tool checks complexity in your CI/CD
  gocyclo -over 14 internal/queries/products.go
  ```
- All complexity values should be under 15

### Risk Mitigation

- **Small steps:** Extract one helper at a time, test immediately
- **Transaction safety:** Helpers receive transaction context, rollback logic stays in parent
- **Type safety:** Go compiler and TypeScript ensure correct signatures
- **Git safety:** Commit after each successful method refactoring
- **Rollback ready:** If a refactoring causes issues, revert that commit and adjust approach

### Success Criteria

- All methods have complexity under 15
- All existing tests pass
- No behavior changes (same inputs produce same outputs)
- Code is more readable with single-responsibility helpers
- PR check passes

## Summary

This refactoring decomposes 7 complex methods using the extract method pattern:

- **3 Go backend methods:** Extract transaction-aware helpers for product and variant CRUD
- **2 Svelte frontend methods:** Extract validation and data transformation functions
- **2 integration tests:** Extract setup and assertion helpers

Total complexity reduction: ~50 decision points moved from parent methods into focused helpers, bringing all methods under the PR check threshold of 15.

The refactoring maintains all existing behavior, error handling patterns, and transaction semantics while improving code readability and maintainability.
