import { redirect } from '@sveltejs/kit'
import { browser } from '$app/environment'
import { base } from '$app/paths'
import type { LayoutLoad } from './$types'

export const load: LayoutLoad = async ({ url, fetch }) => {
  const pathname = url.pathname

  // Allow signin and install pages without authentication
  // pathname includes base path (e.g., /_/signin or /_/install)
  // Check if pathname ends with these routes
  if (pathname.endsWith('/signin') || pathname.endsWith('/install')) {
    return {}
  }

  // Only check authentication on client side where cookies are available
  // On server side (SSR), backend middleware InstallCheck will handle redirects
  if (!browser) {
    return {}
  }

  try {
    const statusResponse = await fetch('/api/install/status', {
      method: 'GET',
      credentials: 'include'
    })

    if (statusResponse.ok) {
      const statusData = await statusResponse.json()
      if (!statusData?.result?.installed) {
        throw redirect(302, `${base}/install`)
      }
    }
  } catch (error) {
    // If it's already a redirect, rethrow it
    if (error && typeof error === 'object' && 'status' in error && error.status === 302) {
      throw error
    }
  }

  // Check if application is installed and user is authenticated
  // Backend middleware InstallCheck redirects to /_/install if not installed
  // We check here for client-side navigation to ensure proper redirects
  let isAuthenticated = false

  try {
    // Try version endpoint to check authentication
    // If app is not installed, backend middleware should redirect this request
    const versionResponse = await fetch('/api/_/version', {
      method: 'GET',
      credentials: 'include'
    })

    if (versionResponse.ok) {
      const versionData = await versionResponse.json()
      if (versionData?.success) {
        isAuthenticated = true
      }
    } else if (versionResponse.status === 400 || versionResponse.status === 401 || versionResponse.status === 403) {
      // 400 = missing/malformed token (app installed but not authenticated)
      // 401/403 = unauthorized (app installed but not authenticated)
      isAuthenticated = false
    }
  } catch (error) {
    // If it's already a redirect, rethrow it
    if (error && typeof error === 'object' && 'status' in error && error.status === 302) {
      throw error
    }
    // Network error or other issue - try products endpoint as fallback
  }

  // If not authenticated yet, try products endpoint
  if (!isAuthenticated) {
    try {
      const productsResponse = await fetch('/api/_/products', {
        method: 'GET',
        credentials: 'include'
      })

      if (productsResponse.ok) {
        isAuthenticated = true
      } else if (
        productsResponse.status === 400 ||
        productsResponse.status === 401 ||
        productsResponse.status === 403
      ) {
        // 400 = missing/malformed token (app installed but not authenticated)
        // 401/403 = unauthorized (app installed but not authenticated)
        throw redirect(302, `${base}/signin`)
      }
    } catch (error) {
      // If it's already a redirect, rethrow it
      if (error && typeof error === 'object' && 'status' in error && error.status === 302) {
        throw error
      }
      // If products endpoint also fails, assume not authenticated
      throw redirect(302, `${base}/signin`)
    }
  }

  if (!isAuthenticated) {
    throw redirect(302, `${base}/signin`)
  }

  return {}
}
