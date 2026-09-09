// 统一 API 请求层：Base URL、响应解包、错误归一化为 ApiError
const BASE_URL = '/api/v1'

export class ApiError extends Error {
  constructor(
    public code: string,
    message: string,
  ) {
    super(message)
    this.name = 'ApiError'
  }
}

// API.md §1 响应格式：成功 { data }，失败 { error: { code, message } }
interface ApiEnvelope<T> {
  data?: T
  error?: { code: string; message: string }
}

// 401 统一处理（SPRINT7_TASK.md §5.9）：auth enabled 游客访问私有资源时跳登录页并携带
// 回跳地址；认证自身请求（/auth/*）与已在登录页时跳过，防止循环跳转。
// 经 window.location 而非 router 实例，避免 api.ts ← service ← store ← router 循环依赖。
function redirectIfUnauthorized(path: string) {
  if (path.startsWith('/auth/')) return
  const { pathname, search } = window.location
  if (pathname === '/login') return
  window.location.assign(`/login?redirect=${encodeURIComponent(pathname + search)}`)
}

export async function request<T>(path: string, init?: RequestInit): Promise<T> {
  try {
    const response = await fetch(BASE_URL + path, init)
    const body = (await response.json()) as ApiEnvelope<T>
    // 非 2xx 或响应体含 error 均视为业务失败
    if (!response.ok || body.error) {
      const code = body.error?.code ?? 'HTTP_ERROR'
      if (code === 'UNAUTHORIZED') redirectIfUnauthorized(path)
      throw new ApiError(code, body.error?.message ?? `HTTP ${response.status}`)
    }
    return body.data as T
  } catch (e) {
    if (e instanceof ApiError) throw e
    // 网络异常 / 响应非 JSON 统一转为 NETWORK_ERROR
    throw new ApiError('NETWORK_ERROR', e instanceof Error ? e.message : '网络请求失败')
  }
}
