// 登录态（GET /auth/status 响应；username/image_root/data_dir 仅已登录时返回，
// 游客不泄露服务器路径，见 SPRINT7_TASK.md §5.8）
export interface AuthStatus {
  enabled: boolean
  authenticated: boolean
  username?: string
  image_root?: string
  data_dir?: string
}

// 登录请求（POST /auth/login）
export interface LoginRequest {
  username: string
  password: string
}

// 登录响应（POST /auth/login）
export interface LoginResult {
  authenticated: boolean
  username: string
}
