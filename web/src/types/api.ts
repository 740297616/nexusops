export interface ApiErrorBody {
  code: string
  message: string
}
export interface ApiResponse<T> {
  data: T
  error: ApiErrorBody | null
}
export interface User {
  id: number
  username: string
  display_name: string
  status: string
  role: string
  created_at: string
  updated_at: string
  last_login_at?: string | null
}
export interface Profile extends User {
  roles: string[]
  permissions: string[]
}
export interface Tokens {
  access_token: string
  refresh_token: string
  access_expires_at: string
  refresh_expires_at: string
  token_type: string
}
export interface Role {
  id: number
  code: string
  name: string
  description: string
  status: string
  is_system: boolean
  created_at: string
  updated_at: string
}
export interface Permission {
  id: number
  code: string
  name: string
  description: string
  resource_type: string
  action_type: string
  created_at: string
  updated_at: string
}
export interface Session {
  id: number
  created_at: string
  access_expires_at: string
  expires_at: string
  current: boolean
}
export interface Page<T> {
  items: T[]
  total: number
  page: number
  page_size: number
}
export interface AuthAudit {
  id: number
  user_id?: number | null
  event_type: string
  outcome: string
  source_ip: string
  user_agent: string
  session_id?: number | null
  metadata: Record<string, unknown>
  created_at: string
}
export interface RoleAudit {
  id: number
  actor_user_id?: number | null
  target_user_id?: number | null
  action: string
  previous_role_ids: number[]
  new_role_ids: number[]
  created_at: string
}
export interface ConfigItem {
  key: string
  value: unknown
  editable: boolean
  updated_at: string
}
export interface ServerAsset {
  id: string
  name: string
  hostname: string
  description: string
  os: string
  os_version: string
  architecture: string
  kernel_version: string
  cpu_model: string
  cpu_cores: number
  memory_total: number
  disk_total: number
  primary_ip: string
  agent_version: string
  agent_status: string
  last_heartbeat_at?: string | null
  last_report_at?: string | null
  status: string
  created_at: string
  updated_at: string
}
export interface AgentInfo { server_id: string; status: string; issued_at: string; rotated_at?: string | null; revoked_at?: string | null }
export interface ServerGroup { id: number; code: string; name: string; description: string; created_at: string; updated_at: string }
export interface ServerTag { id: number; code: string; name: string; created_at: string }
