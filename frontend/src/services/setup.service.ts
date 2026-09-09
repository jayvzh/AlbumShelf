import { request } from './api'
import type { SetupStatus } from '../types/setup'

// 初始化状态检测（GET /setup/status；公开端点，只读检测不落盘）
export function fetchSetupStatus(): Promise<SetupStatus> {
  return request<SetupStatus>('/setup/status')
}
