export interface ApiResponse<T = any> {
  success: boolean
  message?: string
  result?: T
  status?: number
}

export interface RequestOptions extends RequestInit {
  credentials?: RequestCredentials
  method: 'GET' | 'POST' | 'PATCH' | 'DELETE'
  body?: string | FormData
  headers?: HeadersInit
}
