// 初始化状态检测结果（GET /setup/status 响应；与后端 model.SetupStatus 字段一一对应，
// 见 SPRINT8_TASK.md §3A' 与 API.md §3.13）
export interface SetupStatus {
  image_root_configured: boolean // 环境变量 IMAGE_ROOT 已配置
  image_root_exists: boolean // 图片目录已挂载且存在
  image_root_readable: boolean // 图片目录可读
  auth_enabled: boolean // 登录系统是否启用（信息展示，不参与 initialized 判定）
  initialized: boolean // 前三项全真即为 true
}
