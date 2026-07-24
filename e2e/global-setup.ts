import { FullConfig } from 'patchright'
import * as fs from 'fs'
import * as path from 'path'

/**
 * Playwright Global Setup
 *
 * Runs once before all tests to:
 * 1. Check if clean setup needed (only if server not reusing)
 * 2. Install the store via API (if not already installed)
 * 3. Seed test data (10+ products)
 */
async function globalSetup(config: FullConfig) {
  const baseURL = config.projects[0].use.baseURL || 'http://localhost:8080'

  // Environment is cleaned by test-server-start.sh before server starts
  // This ensures server opens a fresh database connection

  console.log('⏳ Waiting for server to be ready...')
  await waitForServer(baseURL, 60000)

  console.log('📦 Checking installation...')
  await installStore(baseURL)

  console.log('🌱 Seeding test data...')
  await seedTestData(baseURL)

  console.log('✅ Global setup complete!')
}

/**
 * Remove lc_base/, lc_digitals/, lc_uploads/ directories
 */
function cleanTestEnvironment() {
  const dirs = ['lc_base', 'lc_digitals', 'lc_uploads']

  for (const dir of dirs) {
    const dirPath = path.join(process.cwd(), dir)
    if (fs.existsSync(dirPath)) {
      fs.rmSync(dirPath, { recursive: true, force: true })
      console.log(`  ✓ Removed ${dir}/`)
    }
  }
}

/**
 * Wait for server to be ready
 */
async function waitForServer(baseURL: string, timeout: number) {
  const startTime = Date.now()

  while (Date.now() - startTime < timeout) {
    try {
      const response = await fetch(`${baseURL}/api/install/status`)
      if (response.ok) {
        console.log('  ✓ Server is ready')
        return
      }
    } catch (error) {
      // Server not ready yet, continue waiting
    }

    await new Promise(resolve => setTimeout(resolve, 1000))
  }

  throw new Error('Server did not become ready in time')
}

/**
 * Install store via API (matches browser behavior from HAR log)
 */
async function installStore(baseURL: string) {
  // Check if already installed
  const statusResponse = await fetch(`${baseURL}/api/install/status`)
  const status = await statusResponse.json()

  if (status.result?.installed) {
    console.log('  ℹ Store already installed, skipping')
    return
  }

  const response = await fetch(`${baseURL}/api/install`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      email: 'admin@example.com',
      password: 'test1234',
      domain: 'localhost:8080',
    }),
  })

  if (!response.ok) {
    const text = await response.text()
    throw new Error(`Installation failed: ${response.status} ${text}`)
  }

  const result = await response.json()
  if (!result.success) {
    throw new Error(`Installation failed: ${result.message}`)
  }

  console.log('  ✓ Store installed via API')
}

/**
 * Login via API and get auth token
 */
async function loginAndGetToken(baseURL: string): Promise<string> {
  const response = await fetch(`${baseURL}/api/sign/in`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      email: 'admin@example.com',
      password: 'test1234',
    }),
  })

  if (!response.ok) {
    const text = await response.text()
    throw new Error(`Login failed: ${response.status} ${text}`)
  }

  // Extract Set-Cookie header for token
  const setCookie = response.headers.get('set-cookie')
  if (!setCookie) {
    throw new Error('No token cookie returned from login')
  }

  // Extract token value from Set-Cookie header
  const tokenMatch = setCookie.match(/token=([^;]+)/)
  if (!tokenMatch) {
    throw new Error('Could not parse token cookie')
  }

  console.log('  ✓ Logged in successfully')
  return tokenMatch[1]
}

/**
 * Seed test data (ensure 10+ active products exist)
 */
async function seedTestData(baseURL: string) {
  const token = await loginAndGetToken(baseURL)
  const productCount = 15

  // Get existing products
  const listResponse = await fetch(`${baseURL}/api/_/products?limit=100`, {
    headers: { 'Cookie': `token=${token}` },
  })

  let existingProducts: any[] = []
  if (listResponse.ok) {
    const listResult = await listResponse.json()
    existingProducts = listResult.result?.products || []
  }

  console.log(`  → Found ${existingProducts.length} existing products`)

  const allProductIds: string[] = []

  // Create missing products
  for (let i = 1; i <= productCount; i++) {
    const slug = `test-product-${i}`
    const existing = existingProducts.find((p: any) => p.slug === slug)

    if (existing) {
      allProductIds.push(existing.id)
      continue
    }

    // Create new product
    const product = {
      name: `Test Product ${i}`,
      slug,
      brief: `This is test product ${i} for E2E testing`,
      description: `Detailed description for test product ${i}. This product is used for automated testing.`,
      amount: 999 + i * 100,
      quantity: 100,
      sku: `TEST-${String(i).padStart(3, '0')}`,
      digital: { type: 'file' },
    }

    const response = await fetch(`${baseURL}/api/_/products`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Cookie': `token=${token}`,
      },
      body: JSON.stringify(product),
    })

    if (response.ok) {
      const result = await response.json()
      if (result.result?.id) {
        allProductIds.push(result.result.id)
      }
    }
  }

  console.log(`  ✓ Ensured ${allProductIds.length}/${productCount} products exist`)

  // Enable all products
  let enabledCount = 0
  for (const productId of allProductIds) {
    const response = await fetch(`${baseURL}/api/_/products/${productId}/active`, {
      method: 'PATCH',
      headers: {
        'Content-Type': 'application/json',
        'Cookie': `token=${token}`,
      },
      body: JSON.stringify({ active: true }),
    })

    if (response.ok) {
      enabledCount++
    }
  }

  console.log(`  ✓ Enabled ${enabledCount}/${allProductIds.length} products`)
}

export default globalSetup
