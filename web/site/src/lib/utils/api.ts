import type { ApiResponse, RequestOptions } from '$lib/types/api'
import { extractErrorMessage } from './errorExtractor'

export async function apiGet<T = any>(url: string): Promise<ApiResponse<T>> {
  return handleRequest<T>(url, {
    credentials: 'include',
    method: 'GET'
  })
}

export async function apiPost<T = any>(url: string, body?: any): Promise<ApiResponse<T>> {
  const options = createOptions('POST', body)
  return handleRequest<T>(url, options)
}

async function handleRequest<T = any>(url: string, options: RequestOptions): Promise<ApiResponse<T>> {
  try {
    const response = await fetch(url, options)

    if (response.status === 204) {
      return { success: true } as ApiResponse<T>
    }

    const text = await response.text()
    const data = text ? JSON.parse(text) : {}

    if (!response.ok) {
      const errorMessage = extractErrorMessage(data)
      return {
        success: false,
        message: errorMessage,
        result: data.result || data,
        status: response.status
      } as ApiResponse<T>
    }

    return {
      ...data,
      status: response.status
    } as ApiResponse<T>
  } catch (error) {
    if (error instanceof SyntaxError) {
      return {
        success: false,
        message: 'Invalid response format',
        result: error.message
      } as ApiResponse<T>
    }
    return {
      success: false,
      message: 'Network error',
      result: (error as Error).message
    } as ApiResponse<T>
  }
}

function createOptions(method: 'POST' | 'PATCH', body?: any): RequestOptions {
  const options: RequestOptions = {
    credentials: 'include',
    method
  }

  if (body) {
    if (body instanceof FormData) {
      options.body = body
    } else if (typeof body === 'object' && Object.keys(body).length > 0) {
      options.body = JSON.stringify(body)
      options.headers = {
        'Content-Type': 'application/json'
      }
    } else {
      options.body = body
    }
  }

  return options
}
