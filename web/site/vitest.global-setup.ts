import { spawn, type ChildProcess } from 'child_process'
import { promisify } from 'util'
import { exec } from 'child_process'

const execAsync = promisify(exec)
const sleep = (ms: number) => new Promise(resolve => setTimeout(resolve, ms))

let serverProcess: ChildProcess | null = null

/**
 * Global setup - runs once before all tests
 * Builds the apps and starts the Go backend server
 */
export async function setup() {
  console.log('🏗️  Building apps...')

  try {
    // Build admin
    console.log('Building admin...')
    await execAsync('npx vite build', { cwd: '../admin' })

    // Build site
    console.log('Building site...')
    await execAsync('npx vite build', { cwd: '.' })

    console.log('✅ Apps built successfully')

    // Start Go server
    console.log('🚀 Starting Go backend server...')
    serverProcess = spawn('go', ['run', './cmd', 'serve'], {
      cwd: '../..',
      stdio: 'pipe',
      detached: false
    })

    // Wait for server to be ready
    let serverReady = false
    let attempts = 0
    const maxAttempts = 30 // 30 seconds timeout

    while (!serverReady && attempts < maxAttempts) {
      try {
        const response = await fetch('http://localhost:8080/')
        if (response.ok) {
          serverReady = true
          console.log('✅ Backend server is ready!')
        }
      } catch (error) {
        // Server not ready yet
        await sleep(1000)
        attempts++
      }
    }

    if (!serverReady) {
      throw new Error('Backend server failed to start within 30 seconds')
    }

  } catch (error) {
    console.error('❌ Setup failed:', error)
    // Kill server process if it exists
    if (serverProcess) {
      serverProcess.kill()
    }
    throw error
  }
}

/**
 * Global teardown - runs once after all tests
 * Stops the Go backend server
 */
export async function teardown() {
  console.log('🛑 Stopping backend server...')

  if (serverProcess) {
    serverProcess.kill('SIGTERM')

    // Wait a moment for graceful shutdown
    await sleep(2000)

    // Force kill if still running
    if (!serverProcess.killed) {
      serverProcess.kill('SIGKILL')
    }

    console.log('✅ Backend server stopped')
  }
}
