import { request } from './api'
import type { AuthStatus, LoginRequest, LoginResult } from '../types/auth'

// 探测登录状态（GET /auth/status；未登录仅返回 enabled/authenticated）
export function fetchAuthStatus(): Promise<AuthStatus> {
  return request<AuthStatus>('/auth/status')
}

// 登录（POST /auth/login；成功后种 session_token Cookie，由浏览器自动携带）
export function login(req: LoginRequest): Promise<LoginResult> {
  return request<LoginResult>('/auth/login', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(req),
  })
}

// 退出登录（POST /auth/logout；服务端删除会话并清除 Cookie）
export function logout(): Promise<{ authenticated: boolean }> {
  return request<{ authenticated: boolean }>('/auth/logout', { method: 'POST' })
}
