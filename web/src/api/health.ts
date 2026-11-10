import { client, type ApiResponse } from './client'

export interface Health {
  status: string
  version: string
}

export async function getHealth(): Promise<Health> {
  const response = await client.get<ApiResponse<Health>>('/health')
  return response.data.data
}
