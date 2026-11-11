import { client, unwrap } from './client'
import type { Profile, Tokens } from '../types/api'
export const login = (username: string, password: string) =>
  unwrap<Tokens>(client.post('/auth/login', { username, password }))
export const me = () => unwrap<Profile>(client.get('/auth/me'))
export const logout = () => unwrap<{ status: string }>(client.post('/auth/logout'))
