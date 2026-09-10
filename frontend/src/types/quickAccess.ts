// 快捷访问（目录钉住）API 类型
export interface QuickAccessListResponse {
  paths: string[]
}

export interface QuickAccessToggleResponse {
  path: string
  pinned: boolean
}
