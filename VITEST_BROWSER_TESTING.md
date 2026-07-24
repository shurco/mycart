# Vitest Browser Mode Testing Implementation

## ✅ Completed (RED Phase)

### Dependencies Installed
- **Site**: `@vitest/browser`, `@vitest/browser-webdriverio`, `webdriverio`
- **Admin**: `@vitest/browser`, `@vitest/browser-webdriverio`, `webdriverio`

### Configuration Created
- **Site**: `web/site/vitest.browser.config.ts`
  - WebdriverIO provider with Chrome
  - Headless mode enabled
  - 30s test timeout
  - Tests files: `src/**/*.browser.test.ts`

- **Admin**: `web/admin/vitest.browser.config.ts`
  - WebdriverIO provider with Chrome
  - Headless mode enabled
  - 30s test timeout
  - Tests files: `src/**/*.browser.test.ts`

### Test Scripts Added
Both `package.json` files updated with:
```json
{
  "test:browser": "vitest --config vitest.browser.config.ts",
  "test:browser:ui": "vitest --config vitest.browser.config.ts --ui"
}
```

### Browser Tests Written (RED Phase - TDD)

#### Site Tests (3 files, 12 tests)

**1. Page Views** (`web/site/src/routes/page-views.browser.test.ts`)
- ✍️ Display home page with products
- ✍️ Navigate to product detail page
- ✍️ Display cart page
- ✍️ Navigate to cart from header

**2. Add to Cart** (`web/site/src/routes/add-to-cart.browser.test.ts`)
- ✍️ Add product to cart from product list page
- ✍️ Toggle product in cart on repeated clicks
- ✍️ Add product to cart from product detail page
- ✍️ Reflect cart state when navigating back to list

**3. Cart Delete** (`web/site/src/routes/cart-delete.browser.test.ts`)
- ✍️ Remove product from cart page
- ✍️ Show empty cart message when all items removed
- ✍️ Update cart total after removing item

#### Admin Tests (2 files, 9 tests)

**1. Page Views** (`web/admin/src/routes/page-views.browser.test.ts`)
- ✍️ Display admin dashboard/home page
- ✍️ Display products page
- ✍️ Display carts page
- ✍️ Display pages management page
- ✍️ Navigate between admin pages

**2. Product Add** (`web/admin/src/routes/product-add.browser.test.ts`)
- ✍️ Open product add drawer when clicking add button
- ✍️ Validate required fields when adding product
- ✍️ Successfully add a new product
- ✍️ Close drawer without saving on cancel

## ❌ Platform Limitation

### OpenBSD Not Supported

**Error**: `The current platform is not supported.`

WebdriverIO (like Playwright) does not support OpenBSD. Browser automation requires:
- Chrome/Chromium browser drivers
- Native browser automation protocols
- Platform-specific binaries

### Solutions

#### 1. Run on Supported Platform
Move to Linux, macOS, or Windows to run tests:
```bash
# Linux/macOS/Windows
cd web/site
npm run test:browser

cd web/admin
npm run test:browser
```

#### 2. Use Container
Run tests in a Linux container with Chrome installed:
```bash
# Using podman/docker
podman run --rm \
  -v $(pwd):/app \
  -w /app/web/site \
  --network host \
  node:22 \
  bash -c "npm install && npm run test:browser"
```

#### 3. Remote Testing
Use a CI/CD pipeline on GitHub Actions, GitLab CI, or similar:
```yaml
# .github/workflows/browser-tests.yml
name: Browser Tests
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-node@v3
      - name: Install dependencies
        run: |
          cd web/site && npm install
          cd ../admin && npm install
      - name: Run browser tests
        run: |
          cd web/site && npm run test:browser
          cd ../admin && npm run test:browser
```

## TDD Status

Following the Red-Green-Refactor cycle:

1. ✅ **RED**: Write failing tests - **COMPLETED**
2. ⏸️ **Verify RED**: Run tests to see them fail - **BLOCKED** (OpenBSD limitation)
3. ⏸️ **GREEN**: Write minimal code to pass - **PENDING**
4. ⏸️ **Verify GREEN**: Run tests to see them pass - **PENDING**
5. ⏸️ **REFACTOR**: Clean up code - **PENDING**

## Test Scenarios Covered

### Site (Storefront)
- ✅ Each page view (home, product detail, cart)
- ✅ Add to cart from product list page
- ✅ Add to cart from product detail page
- ✅ Product delete in cart page

### Admin (Admin Portal)
- ✅ Each page view (dashboard, products, carts, pages)
- ✅ Product add in product list page

## Running Tests (When on Supported Platform)

### Build and Serve
```bash
# Terminal 1: Build and start server
pushd web/admin/ && npx vite build && popd
pushd web/site/ && npx vite build && popd
go run ./cmd serve
```

### Run Browser Tests
```bash
# Terminal 2: Run site tests
cd web/site
npm run test:browser

# Run admin tests
cd web/admin
npm run test:browser
```

### Interactive UI Mode
```bash
# Site
cd web/site
npm run test:browser:ui

# Admin
cd web/admin
npm run test:browser:ui
```

## Test Structure

All browser tests use Vitest browser mode API:
```typescript
import { expect, test } from 'vitest'
import { page } from '@vitest/browser/context'

test('should display home page', async () => {
  await page.goto('http://localhost:8080/')
  const heading = page.getByRole('heading', { name: /products/i })
  await expect.element(heading).toBeVisible()
})
```

## Next Steps

When running on a supported platform:
1. **Verify RED**: Run tests to confirm they fail appropriately
2. **GREEN Phase**: Add test IDs or minimal code changes to make tests pass
3. **Verify GREEN**: Confirm all tests pass
4. **REFACTOR**: Clean up any duplicate code or improve test structure
5. **Commit**: Commit the working browser tests

## Notes

- Tests use `data-testid` attributes for reliable element selection
- Tests handle authentication (admin tests skip if redirected to signin)
- Tests clean up state between runs
- All tests follow TDD best practices (one behavior per test, clear names, real code not mocks)
