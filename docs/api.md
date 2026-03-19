# LiteCart API Reference

LiteCart provides a RESTful JSON API split into two groups:

- **Public API** — accessible without authentication (product catalog, pages, payments).
- **Private API** — requires a JWT token obtained via `/api/sign/in` (admin panel operations).

All responses follow a common envelope:

```json
{
  "success": true,
  "message": "Human-readable summary",
  "result": {}
}
```

Errors return `"success": false` with the appropriate HTTP status code.

---

## Authentication

### Sign In

```
POST /api/sign/in
```

Authenticates an admin user and returns a JWT token in both the response body and a `token` HTTP-only cookie.

**Request Body:**

| Field    | Type   | Required | Description               |
|----------|--------|----------|---------------------------|
| email    | string | yes      | Admin email address       |
| password | string | yes      | Admin password (6-72 chars) |

**Response (200):**

```json
{
  "success": true,
  "message": "Token",
  "result": "<jwt_token>"
}
```

**Set-Cookie:** `token=<jwt_token>; HttpOnly; SameSite=Strict`

**Errors:** `400` invalid credentials or validation error, `500` internal error.

---

### Sign Out

```
POST /api/sign/out
```

**Authentication:** Required (JWT)

Invalidates the current session and clears the `token` cookie.

**Response:** `204 No Content`

---

## Installation

### Install

```
POST /api/install
```

Performs the initial application setup. Can only be called once.

**Request Body:**

| Field    | Type   | Required | Description         |
|----------|--------|----------|---------------------|
| email    | string | yes      | Admin email         |
| password | string | yes      | Admin password      |
| domain   | string | no       | Site domain          |

**Response (200):**

```json
{
  "success": true,
  "message": "Cart installed"
}
```

---

## Public API

### Health Check

```
GET /ping
```

Returns a pong response for liveness probes.

**Response (200):**

```json
{
  "success": true,
  "message": "Pong"
}
```

---

### Public Settings

```
GET /api/settings
```

Returns public site settings including main info, social links, and published pages.

**Response (200):**

```json
{
  "success": true,
  "message": "Settings",
  "result": {
    "main": {
      "site_name": "My Store",
      "domain": "example.com",
      "currency": "USD"
    },
    "socials": {
      "facebook": "",
      "instagram": "",
      "twitter": ""
    },
    "pages": [...]
  }
}
```

---

### Products (Public)

#### List Products

```
GET /api/products?page=1&limit=20
```

Returns a paginated list of active products visible to customers.

**Query Parameters:**

| Param | Type | Default | Description               |
|-------|------|---------|---------------------------|
| page  | int  | 1       | Page number (min 1)       |
| limit | int  | 20      | Items per page (max 100)  |

**Response (200):**

```json
{
  "success": true,
  "message": "Products",
  "result": {
    "total": 10,
    "currency": "USD",
    "products": [...]
  }
}
```

#### Get Product

```
GET /api/products/:product_id
```

Returns a single active product by ID.

**Path Parameters:**

| Param      | Type   | Description |
|------------|--------|-------------|
| product_id | string | Product ID  |

**Response (200):**

```json
{
  "success": true,
  "message": "Product info",
  "result": { ... }
}
```

---

### Pages (Public)

#### Get Page

```
GET /api/pages/:page_slug
```

Returns a page by its URL slug.

**Path Parameters:**

| Param     | Type   | Description |
|-----------|--------|-------------|
| page_slug | string | Page slug   |

**Response (200):**

```json
{
  "success": true,
  "message": "Page content",
  "result": { ... }
}
```

**Errors:** `404` page not found.

---

### Cart & Payments (Public)

#### Payment List

```
GET /api/cart/payment
```

Returns a map of available payment providers and their active status.

**Response (200):**

```json
{
  "success": true,
  "message": "Payment list",
  "result": {
    "stripe": true,
    "paypal": false,
    "spectrocoin": false,
    "coinbase": true,
    "dummy": true
  }
}
```

#### Get Cart

```
GET /api/cart/:cart_id
```

Returns cart details by cart ID.

**Path Parameters:**

| Param   | Type   | Description |
|---------|--------|-------------|
| cart_id | string | Cart ID     |

**Response (200):**

```json
{
  "success": true,
  "message": "Cart",
  "result": {
    "id": "abc123",
    "email": "user@example.com",
    "amount_total": 1000,
    "currency": "USD",
    "payment_status": "new",
    "payment_system": "stripe",
    "items": [...]
  }
}
```

**Errors:** `400` missing cart_id, `404` cart not found.

#### Initiate Payment

```
POST /cart/payment
```

Creates a payment session and returns a redirect URL.

**Request Body:**

| Field    | Type               | Required | Description                        |
|----------|--------------------|----------|------------------------------------|
| email    | string             | yes      | Customer email                     |
| provider | string             | yes      | Payment system: `stripe`, `paypal`, `spectrocoin`, `coinbase`, `dummy` |
| products | array of CartProduct | yes    | Products in cart                   |

CartProduct:

| Field    | Type   | Description |
|----------|--------|-------------|
| id       | string | Product ID  |
| quantity | int    | Quantity    |

**Response (200):**

```json
{
  "success": true,
  "message": "Payment url",
  "result": {
    "url": "https://checkout.stripe.com/..."
  }
}
```

**Errors:** `400` dummy provider used for paid cart, `500` internal error.

> **Note:** The `dummy` provider is only valid when the cart total is 0 (free products only).

#### Payment Callback

```
POST /cart/payment/callback
```

Webhook endpoint called by payment providers (e.g., SpectroCoin) to report payment status changes.

**Query Parameters:**

| Param          | Type   | Description     |
|----------------|--------|-----------------|
| cart_id        | string | Cart ID         |
| payment_system | string | Payment system  |

**Response:** `200` with `*ok*` text body.

#### Payment Success

```
GET /cart/payment/success
```

Handles successful payment redirects from payment providers. Verifies payment status and updates the cart.

**Query Parameters:**

| Param          | Type   | Description                |
|----------------|--------|----------------------------|
| cart_id        | string | Cart ID                    |
| payment_system | string | Payment system             |
| session        | string | Stripe session ID (Stripe) |
| token          | string | PayPal token (PayPal)      |
| charge_id      | string | Coinbase charge (Coinbase) |

**Response:** Passes control to the SPA handler after processing.

#### Payment Cancel

```
GET /cart/payment/cancel
```

Handles canceled payment redirects. Updates cart status to `canceled`.

**Query Parameters:**

| Param          | Type   | Description    |
|----------------|--------|----------------|
| cart_id        | string | Cart ID        |
| payment_system | string | Payment system |

**Response:** `302` redirect to SPA cancel page.

---

## Private API (Admin)

All private endpoints require a valid JWT token sent as a `token` cookie or `Authorization: Bearer <token>` header.

### Version

```
GET /api/_/version
```

Returns the current application version and update information. Results are cached for 24 hours.

**Response (200):**

```json
{
  "success": true,
  "message": "Version",
  "result": {
    "current_version": "v0.10.0",
    "new_version": "v0.11.0",
    "release_url": "https://github.com/shurco/litecart/releases/..."
  }
}
```

---

### Settings (Admin)

#### Get Setting

```
GET /api/_/settings/:setting_key
```

Returns a setting group or individual setting.

**Path Parameters:**

| Param       | Type   | Description                                                  |
|-------------|--------|--------------------------------------------------------------|
| setting_key | string | One of: `main`, `social`, `auth`, `jwt`, `webhook`, `payment`, `stripe`, `paypal`, `spectrocoin`, `coinbase`, `dummy`, `mail`, or a custom key |

**Response (200):**

```json
{
  "success": true,
  "message": "Setting",
  "result": { ... }
}
```

**Errors:** `404` setting not found, `404` when requesting `password`.

#### Update Setting

```
PATCH /api/_/settings/:setting_key
```

Updates a setting group or individual setting.

**Path Parameters:** Same as Get Setting.

**Request Body:** Depends on the `setting_key`. Examples:

For `main`:
```json
{ "site_name": "My Store", "domain": "example.com", "email": "admin@example.com" }
```

For `stripe`:
```json
{ "secret_key": "sk_...", "active": true }
```

For `paypal`:
```json
{ "client_id": "...", "secret_key": "...", "active": true }
```

For `spectrocoin`:
```json
{ "merchant_id": "...", "project_id": "...", "private_key": "...", "active": true }
```

For `coinbase`:
```json
{ "api_key": "...", "active": true }
```

For `password`:
```json
{ "old": "current_password", "new": "new_password" }
```

**Response (200):**

```json
{ "success": true, "message": "Setting group updated" }
```

---

#### Test Letter

```
GET /api/_/test/letter/:letter_name
```

Sends a test email using the configured SMTP settings.

**Path Parameters:**

| Param       | Type   | Description               |
|-------------|--------|---------------------------|
| letter_name | string | Letter template name       |

**Response (200):**

```json
{
  "success": true,
  "message": "Test letter",
  "result": "Message sent to your mailbox"
}
```

---

### Pages (Admin)

#### List Pages

```
GET /api/_/pages?page=1&limit=20
```

Returns a paginated list of all pages (including inactive).

**Query Parameters:**

| Param | Type | Default | Description               |
|-------|------|---------|---------------------------|
| page  | int  | 1       | Page number               |
| limit | int  | 20      | Items per page (max 100)  |

**Response (200):**

```json
{
  "success": true,
  "message": "Pages",
  "result": {
    "pages": [...],
    "total": 5,
    "page": 1,
    "limit": 20
  }
}
```

#### Get Page

```
GET /api/_/pages/:page_id
```

Returns a single page by ID.

**Response (200):**

```json
{
  "success": true,
  "message": "Page",
  "result": { "id": "...", "name": "...", "slug": "...", ... }
}
```

**Errors:** `404` page not found.

#### Create Page

```
POST /api/_/pages
```

Creates a new page.

**Request Body:**

| Field    | Type   | Required | Description                     |
|----------|--------|----------|---------------------------------|
| name     | string | yes      | Page name (3-50 chars)          |
| slug     | string | yes      | URL slug (3-20 chars)           |
| position | string | no       | Position: `header` or `footer`  |
| content  | string | no       | Page HTML content               |

**Response (200):**

```json
{
  "success": true,
  "message": "Page added",
  "result": { "id": "...", ... }
}
```

#### Update Page

```
PATCH /api/_/pages/:page_id
```

Updates page metadata (name, slug, position, SEO).

**Request Body:** Same fields as Create Page (all optional).

**Response (200):**

```json
{ "success": true, "message": "Page updated" }
```

#### Update Page Content

```
PATCH /api/_/pages/:page_id/content
```

Updates only the HTML content of a page.

**Request Body:**

| Field   | Type   | Description     |
|---------|--------|-----------------|
| content | string | HTML content    |

**Response (200):**

```json
{ "success": true, "message": "Page content updated" }
```

#### Toggle Page Active

```
PATCH /api/_/pages/:page_id/active
```

Toggles the active/inactive status of a page.

**Response (200):**

```json
{
  "success": true,
  "message": "Page active updated",
  "result": { ... }
}
```

#### Delete Page

```
DELETE /api/_/pages/:page_id
```

Deletes a page.

**Response (200):**

```json
{ "success": true, "message": "Page deleted" }
```

---

### Products (Admin)

#### List Products

```
GET /api/_/products?page=1&limit=20
```

Returns a paginated list of all products (including inactive).

**Query Parameters:**

| Param | Type | Default | Description               |
|-------|------|---------|---------------------------|
| page  | int  | 1       | Page number               |
| limit | int  | 20      | Items per page (max 100)  |

**Response (200):**

```json
{
  "success": true,
  "message": "Products",
  "result": { "total": 10, "currency": "USD", "products": [...] }
}
```

#### Create Product

```
POST /api/_/products
```

Creates a new product.

**Request Body:**

| Field       | Type    | Required | Description                                |
|-------------|---------|----------|--------------------------------------------|
| name        | string  | yes      | Product name (3-50 chars)                  |
| slug        | string  | yes      | URL slug (3-20 chars)                      |
| amount      | int     | yes      | Price in cents (0 for free products)       |
| digital     | object  | yes      | `{ "type": "file" | "data" | "api" }`     |
| description | string  | no       | Product description                        |
| brief       | string  | no       | Short description                          |
| metadata    | array   | no       | Key-value metadata                         |
| attributes  | array   | no       | String attributes                          |

**Response (200):**

```json
{
  "success": true,
  "message": "Product added",
  "result": { "id": "...", ... }
}
```

**Errors:** `400` validation error.

#### Get Product

```
GET /api/_/products/:product_id
```

Returns a single product by ID.

**Response (200):**

```json
{
  "success": true,
  "message": "Product info",
  "result": { ... }
}
```

#### Update Product

```
PATCH /api/_/products/:product_id
```

Updates product metadata.

**Response (200):**

```json
{
  "success": true,
  "message": "Product updated",
  "result": { ... }
}
```

#### Delete Product

```
DELETE /api/_/products/:product_id
```

Deletes a product.

**Response (200):**

```json
{ "success": true, "message": "Product deleted" }
```

#### Toggle Product Active

```
PATCH /api/_/products/:product_id/active
```

Toggles the active/inactive status of a product.

**Response (200):**

```json
{ "success": true, "message": "Product active updated" }
```

#### List Product Images

```
GET /api/_/products/:product_id/image
```

Returns all images for a product.

**Response (200):**

```json
{
  "success": true,
  "message": "Product images",
  "result": [{ "id": "...", "name": "...", "ext": "png" }]
}
```

#### Upload Product Image

```
POST /api/_/products/:product_id/image
```

Uploads an image for a product. Automatically creates `sm` (147x147) and `md` (400x400) resized variants.

**Content-Type:** `multipart/form-data`

**Form Fields:**

| Field    | Type | Description                     |
|----------|------|---------------------------------|
| document | file | Image file (PNG or JPEG only)   |

**Response (200):**

```json
{
  "success": true,
  "message": "Image added",
  "result": { "id": "...", "name": "...", "ext": "png", "orig_name": "photo.png" }
}
```

#### Delete Product Image

```
DELETE /api/_/products/:product_id/image/:image_id
```

Deletes a product image.

**Response (200):**

```json
{ "success": true, "message": "Image deleted" }
```

#### Get Product Digital

```
GET /api/_/products/:product_id/digital
```

Returns digital content (files or license keys) for a product.

**Response (200):**

```json
{
  "success": true,
  "message": "Product digital",
  "result": { "type": "file", "files": [...] }
}
```

**Errors:** `404` product not found.

#### Add Product Digital

```
POST /api/_/products/:product_id/digital
```

Adds digital content to a product. If a file is uploaded via `document` form field, it is stored as a digital file. Otherwise, a new data (license key) entry is created.

**Content-Type:** `multipart/form-data` (with file) or `application/json` (without file)

**Response (200):**

```json
{
  "success": true,
  "message": "Digital added",
  "result": { ... }
}
```

#### Update Product Digital

```
PATCH /api/_/products/:product_id/digital/:digital_id
```

Updates the content of a digital data entry (license key).

**Request Body:**

| Field   | Type   | Description         |
|---------|--------|---------------------|
| content | string | License key content |

**Response (200):**

```json
{ "success": true, "message": "Digital updated" }
```

#### Delete Product Digital

```
DELETE /api/_/products/:product_id/digital/:digital_id
```

Deletes a digital content entry.

**Response (200):**

```json
{ "success": true, "message": "Digital deleted" }
```

---

### Carts (Admin)

#### List Carts

```
GET /api/_/carts?page=1&limit=20
```

Returns a paginated list of all carts.

**Query Parameters:**

| Param | Type | Default | Description               |
|-------|------|---------|---------------------------|
| page  | int  | 1       | Page number               |
| limit | int  | 20      | Items per page (max 100)  |

**Response (200):**

```json
{
  "success": true,
  "message": "Carts",
  "result": {
    "carts": [...],
    "total": 50,
    "page": 1,
    "limit": 20
  }
}
```

#### Get Cart

```
GET /api/_/carts/:cart_id
```

Returns detailed cart information including product items.

**Response (200):**

```json
{
  "success": true,
  "message": "Cart",
  "result": {
    "id": "...",
    "email": "user@example.com",
    "amount_total": 1000,
    "currency": "USD",
    "payment_status": "paid",
    "payment_system": "stripe",
    "payment_id": "...",
    "created": 1700000000,
    "updated": 1700000100,
    "items": [...]
  }
}
```

**Errors:** `400` missing cart_id, `404` cart not found.

#### Send Cart Email

```
POST /api/_/carts/:cart_id/mail
```

Re-sends the purchase confirmation email for a cart.

**Response (200):**

```json
{
  "success": true,
  "message": "Mail sended"
}
```

---

## Payment Systems

| Provider    | Key          | Description                      |
|-------------|--------------|----------------------------------|
| Stripe      | `stripe`     | Credit/debit card payments       |
| PayPal      | `paypal`     | PayPal account payments          |
| SpectroCoin | `spectrocoin`| Cryptocurrency via SpectroCoin   |
| Coinbase    | `coinbase`   | Cryptocurrency via Coinbase Commerce |
| Dummy       | `dummy`      | Built-in provider for free products (always active) |

## Payment Statuses

| Status    | Description            |
|-----------|------------------------|
| `new`     | Payment initiated      |
| `pending` | Awaiting confirmation  |
| `paid`    | Payment confirmed      |
| `expired` | Payment expired        |
| `canceled`| Payment canceled       |

## Webhook Events

When a webhook URL is configured, LiteCart sends POST notifications for:

| Event               | Description              |
|---------------------|--------------------------|
| `payment_initiation`| Payment session created  |
| `payment_callback`  | Provider callback received |
| `payment_success`   | Payment confirmed        |
| `payment_cancel`    | Payment canceled         |
