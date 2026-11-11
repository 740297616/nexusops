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
export const getConfig = () => unwrap<ConfigItem[]>(client.get('/system/config'))
export const setConfig = (body: Record<string, unknown>) =>
  unwrap<{ status: string }>(client.put('/system/config', body))
