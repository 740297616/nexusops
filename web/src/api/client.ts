import axios, { type AxiosError, type InternalAxiosRequestConfig } from 'axios'
import type { ApiResponse } from '../types/api'
export type { ApiResponse } from '../types/api'
export class ApiError extends Error {
  constructor(
    message: string,
    public readonly status?: number,
    public readonly code?: string,
  ) {
    super(message)
    this.name = 'ApiError'
  }
}
let accessToken: string | null = null
let refreshToken: string | null = null
let refreshPromise: Promise<string | null> | null = null
let onAuthFailure: (() => void) | null = null
export function setAccessToken(token: string | null) {
  accessToken = token
}
export function setRefreshToken(token: string | null) {
  refreshToken = token
}
export function configureAuth(failure: () => void) {
  onAuthFailure = failure
}
export const client = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || '/api/v1',
  timeout: 10000,
})
client.interceptors.request.use((config) => {
  if (accessToken) config.headers.Authorization = `Bearer ${accessToken}`
  return config
})
client.interceptors.response.use(
  (response) => response,
  async (error: AxiosError<ApiResponse<unknown>>) => {
    const original = error.config as
      (InternalAxiosRequestConfig & { _authRetry?: boolean }) | undefined
    if (error.response?.status === 401 && original && !original._authRetry && refreshToken) {
      original._authRetry = true
      if (!refreshPromise) {
        refreshPromise = axios
          .post<ApiResponse<{ access_token: string; refresh_token: string }>>(
            `${import.meta.env.VITE_API_BASE_URL || '/api/v1'}/auth/refresh`,
            { refresh_token: refreshToken },
          )
          .then((r) => {
            accessToken = r.data.data.access_token
            refreshToken = r.data.data.refresh_token
            return accessToken
          })
          .catch(() => {
            accessToken = null
            refreshToken = null
            onAuthFailure?.()
            return null
          })
          .finally(() => {
            refreshPromise = null
          })
      }
      const token = await refreshPromise
      if (token) {
        original.headers.Authorization = `Bearer ${token}`
        return client(original)
      }
    }
    const detail = error.response?.data?.error
    return Promise.reject(
      new ApiError(
        detail?.message || (error.code === 'ECONNABORTED' ? '请求超时' : '请求失败'),
        error.response?.status,
        detail?.code,
      ),
    )
  },
)
export async function unwrap<T>(request: Promise<{ data: ApiResponse<T> }>): Promise<T> {
  return (await request).data.data
}
