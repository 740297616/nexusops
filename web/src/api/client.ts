import axios from 'axios'

export interface ApiResponse<T> {
  data: T
  error: { code: string; message: string } | null
}

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

export const client = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || '/api/v1',
  timeout: 10000,
})

client.interceptors.response.use(
  (response) => response,
  (error: unknown) => {
    if (axios.isAxiosError<ApiResponse<unknown>>(error)) {
      const detail = error.response?.data?.error
      return Promise.reject(
        new ApiError(
          detail?.message || (error.code === 'ECONNABORTED' ? '请求超时' : '无法连接服务'),
          error.response?.status,
          detail?.code,
        ),
      )
    }
    return Promise.reject(new ApiError('请求发生未知错误'))
  },
)
