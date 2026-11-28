# Swagger API Documentation Setup

This document explains how to use and maintain the Swagger API documentation for myCart.

## Overview

The myCart API is documented using [swaggo/swag](https://github.com/swaggo/swag), which generates OpenAPI 2.0 (Swagger) documentation from Go annotations in the source code.

## Local Development

### Prerequisites

Install the Swag CLI tool:

```bash
go install github.com/swaggo/swag/cmd/swag@latest
```

### Generate Documentation

Generate Swagger documentation files:

```bash
swag init -g cmd/main.go --output docs/swagger --parseDependency --parseInternal
```

This creates:
- `docs/swagger/docs.go` - Generated Go code
- `docs/swagger/swagger.json` - OpenAPI spec in JSON format
- `docs/swagger/swagger.yaml` - OpenAPI spec in YAML format

**Note:** Generated Swagger files (docs.go, swagger.json, swagger.yaml) are git-ignored. Documentation is auto-generated in CI/CD.

### View Swagger UI Locally

Start the server in development mode:

```bash
go run ./cmd serve --dev
```

Then visit: **http://localhost:8080/_/swagger/** or **http://localhost:8080/_/swagger/index.html**

The Swagger UI is available at `/_/swagger/` in all modes. Access to OpenAPI endpoints (`/swagger.json`, `/openapi.json`) is controlled via SEO settings.

**Note:** This project uses Fiber v3, which is not yet supported by fiber-swagger. The Swagger UI is served using Fiber's static middleware instead.

## Deployed Documentation

API documentation is automatically published to GitHub Pages whenever code is pushed to the `main` branch, as part of the comprehensive documentation deployment.

**Live documentation:**
- **Local:** `http://localhost:8080/_/swagger/`
- **GitHub Pages:** `https://<username>.github.io/<repo>/swagger/`

### How It Works

1. On push to `main`, the GitHub Actions workflow `.github/workflows/deploy-docs.yml` runs
2. The workflow installs Go and the Swag CLI
3. It generates fresh Swagger documentation in `docs/swagger/`
4. It downloads Swagger UI static assets
5. It combines Swagger docs with E2E reports and Solidebase docs
6. It deploys everything to the `gh-pages` branch

The deployed site includes:
- **Swagger API docs** at `/swagger/`
- **E2E test reports** at `/e2e/admin/` and `/e2e/site/`
- **Main documentation** (Solidebase) at the root

### Enabling GitHub Pages

If this is the first deployment:

1. Go to **Settings** → **Pages** in your GitHub repository
2. Under **Build and deployment**, set:
   - **Source**: Deploy from a branch
   - **Branch**: `gh-pages`, folder: `/ (root)`
3. Click **Save**

The documentation will be live at `https://<username>.github.io/<repo>/` after the next push to `main`.

## Writing Swagger Annotations

### General API Info

Main annotations are in `cmd/main.go`:

```go
// @title           myCart API
// @version         1.0
// @description     Open source shopping-cart backend API
// @host            localhost:8080
// @BasePath        /
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
```

### Handler Annotations

Example from `internal/handlers/private/product.go`:

```go
// AddProduct creates a new product.
//
// @Summary      Create product
// @Description  Create a new product with name, slug, price, and digital type
// @Tags         Products
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        request body models.Product true "Product data"
// @Success      200 {object} webutil.HTTPResponse{result=models.Product} "Created product"
// @Failure      400 {object} webutil.HTTPResponse "Validation error"
// @Failure      500 {object} webutil.HTTPResponse "Internal server error"
// @Router       /api/_/products [post]
func AddProduct(c fiber.Ctx) error {
    // implementation
}
```

### Required Annotations

- `@Summary` - Short description (< 120 chars)
- `@Description` - Detailed description
- `@Tags` - Group endpoints (Products, Cart, Auth, etc.)
- `@Accept` / `@Produce` - Content types (json, multipart/form-data, etc.)
- `@Param` - Parameters (path, query, body)
- `@Success` / `@Failure` - Response codes with models
- `@Router` - Route path and HTTP method
- `@Security BearerAuth` - For protected endpoints

### Common Patterns

**Path parameter:**
```go
// @Param product_id path string true "Product ID"
```

**Query parameter:**
```go
// @Param page query int false "Page number" default(1)
```

**Request body:**
```go
// @Param request body models.Product true "Product data"
```

**Response with nested model:**
```go
// @Success 200 {object} webutil.HTTPResponse{result=models.Product} "Product details"
```

## Troubleshooting

### Build Errors After Generating Docs

If you get errors about undefined fields in `docs/docs.go`, you may have a version mismatch:

```bash
# Check versions
go list -m github.com/swaggo/swag
swag --version

# Update to matching version
go get github.com/swaggo/swag@latest
go mod tidy
```

### Missing Model Definitions

If Swagger cannot find a model type, ensure the package is imported:

```go
import (
    _ "github.com/shurco/mycart/internal/models"
)
```

Use blank import (`_`) if the package is only referenced in annotations.

### Workflow Fails

Check the GitHub Actions logs:
1. Go to **Actions** tab in GitHub
2. Click the failed workflow run
3. Expand the failing step

Common issues:
- Go version mismatch (update in `.github/workflows/deploy-swagger.yml`)
- Missing permissions (check repository settings)
- Annotation syntax errors (validate with `swag init` locally)

## API Documentation Coverage

Current documentation includes:

- **Auth**: Sign in, sign out
- **Cart**: List, get, create, update, payment
- **Products**: CRUD operations, images, digital files, variants
- **Pages**: CRUD operations for static pages
- **Settings**: Get and update settings
- **Public API**: Products, pages, cart operations for storefront

**Total endpoints documented:** 40+

## References

- [swaggo/swag documentation](https://github.com/swaggo/swag)
- [Swagger 2.0 Specification](https://swagger.io/specification/v2/)
- [Fiber Swagger middleware](https://github.com/swaggo/fiber-swagger)
