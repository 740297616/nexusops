import { client, unwrap } from './client'
import type {
  AuthAudit,
  ConfigItem,
  Page,
  Permission,
  Role,
  RoleAudit,
  Session,
  User,
  AgentInfo,
  ServerAsset,
  ServerGroup,
  ServerTag,
} from '../types/api'
export const listUsers = (params: Record<string, unknown>) =>
  unwrap<Page<User>>(client.get('/users', { params }))
export const getUser = (id: number) => unwrap<User>(client.get(`/users/${id}`))
export const createUser = (body: Record<string, unknown>) =>
  unwrap<User>(client.post('/users', body))
export const updateUser = (id: number, body: Record<string, unknown>) =>
  unwrap<{ status: string }>(client.put(`/users/${id}`, body))
export const setUserStatus = (id: number, status: string) =>
  unwrap<{ status: string }>(client.patch(`/users/${id}/status`, { status }))
export const resetPassword = (id: number, password: string) =>
  unwrap<{ status: string }>(client.put(`/users/${id}/password`, { password }))
export const userRoles = (id: number) => unwrap<Role[]>(client.get(`/users/${id}/roles`))
export const setUserRoles = (id: number, role_ids: number[]) =>
  unwrap<{ status: string }>(client.put(`/users/${id}/roles`, { role_ids }))
export const listRoles = () => unwrap<Role[]>(client.get('/roles'))
export const getRole = (id: number) => unwrap<Role>(client.get(`/roles/${id}`))
export const createRole = (body: Record<string, unknown>) =>
  unwrap<Role>(client.post('/roles', body))
export const updateRole = (id: number, body: Record<string, unknown>) =>
  unwrap<{ status: string }>(client.put(`/roles/${id}`, body))
export const setRoleStatus = (id: number, status: string) =>
  unwrap<{ status: string }>(client.patch(`/roles/${id}/status`, { status }))
export const deleteRole = (id: number) => unwrap<{ status: string }>(client.delete(`/roles/${id}`))
export const listPermissions = () => unwrap<Permission[]>(client.get('/permissions'))
export const rolePermissions = (id: number) =>
  unwrap<Permission[]>(client.get(`/roles/${id}/permissions`))
export const setRolePermissions = (id: number, permission_ids: number[]) =>
  unwrap<{ status: string }>(client.put(`/roles/${id}/permissions`, { permission_ids }))
export const listAuthAudit = (params: Record<string, unknown>) =>
  unwrap<Page<AuthAudit>>(client.get('/audit/auth', { params }))
export const listRoleAudit = (params: Record<string, unknown>) =>
  unwrap<Page<RoleAudit>>(client.get('/audit/roles', { params }))
export const listSessions = (params: Record<string, unknown>) =>
  unwrap<Page<Session>>(client.get('/admin/sessions', { params }))
export const listUserSessions = (id: number, params: Record<string, unknown>) =>
  unwrap<Page<Session>>(client.get(`/admin/users/${id}/sessions`, { params }))
export const revokeUserSessions = (id: number) =>
  unwrap<{ status: string }>(client.delete(`/admin/users/${id}/sessions`))
export const revokeUserSession = (userId: number, sessionId: number) =>
  unwrap<{ status: string }>(client.delete(`/admin/users/${userId}/sessions/${sessionId}`))
export const getConfig = () => unwrap<ConfigItem[]>(client.get('/system/config'))
export const setConfig = (body: Record<string, unknown>) =>
  unwrap<{ status: string }>(client.put('/system/config', body))
export const listServers = (params: Record<string, unknown>) => unwrap<Page<ServerAsset>>(client.get('/servers', { params }))
export const listServerGroups = (params: Record<string, unknown> = {}) => unwrap<{ items: ServerGroup[]; total: number; page: number; page_size: number }>(client.get('/groups', { params }))
export const createServerGroup = (body: Record<string, unknown>) => unwrap<ServerGroup>(client.post('/groups', body))
export const updateServerGroup = (id: number, body: Record<string, unknown>) => unwrap<{ status: string }>(client.put(`/groups/${id}`, body))
export const deleteServerGroup = (id: number) => unwrap<{ status: string }>(client.delete(`/groups/${id}`))
export const listServerTags = () => unwrap<ServerTag[]>(client.get('/tags'))
export const createServerTag = (body: Record<string, unknown>) => unwrap<ServerTag>(client.post('/tags', body))
export const updateServerTag = (id: number, body: Record<string, unknown>) => unwrap<{ status: string }>(client.put(`/tags/${id}`, body))
export const deleteServerTag = (id: number) => unwrap<{ status: string }>(client.delete(`/tags/${id}`))
export const setServerGroups = (id: string, group_ids: number[]) => unwrap<{ status: string }>(client.put(`/servers/${id}/groups`, { group_ids }))
export const setServerTags = (id: string, tag_ids: number[]) => unwrap<{ status: string }>(client.put(`/servers/${id}/tags`, { tag_ids }))
export const serverGroups = (id: string) => unwrap<ServerGroup[]>(client.get(`/servers/${id}/groups`))
export const serverTags = (id: string) => unwrap<ServerTag[]>(client.get(`/servers/${id}/tags`))
export const getServer = (id: string) => unwrap<ServerAsset>(client.get(`/servers/${id}`))
export const createServer = (body: Record<string, unknown>) => unwrap<ServerAsset>(client.post('/servers', body))
export const updateServer = (id: string, body: Record<string, unknown>) => unwrap<{ status: string }>(client.put(`/servers/${id}`, body))
export const setServerStatus = (id: string, status: string) => unwrap<{ status: string }>(client.patch(`/servers/${id}/status`, { status }))
export const deleteServer = (id: string) => unwrap<{ status: string }>(client.delete(`/servers/${id}`))
export const getAgentInfo = (id: string) => unwrap<AgentInfo>(client.get(`/servers/${id}/agent`))
export const createAgentRegistrationToken = (id: string) => unwrap<{ token: string; expires_at: string }>(client.post(`/servers/${id}/agent/register-token`))
export const rotateAgentCredential = (id: string) => unwrap<{ credential: string }>(client.post(`/servers/${id}/agent/rotate-token`))
export const revokeAgentCredential = (id: string) => unwrap<{ status: string }>(client.post(`/servers/${id}/agent/revoke`))
