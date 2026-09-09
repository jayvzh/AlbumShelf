package model

// SetupStatus 初始化状态检测结果（SPRINT8 §5.3）。
// 全部字段由只读检测实时得出，不落盘、不缓存。
type SetupStatus struct {
	// ImageRootConfigured 环境变量 IMAGE_ROOT 是否已配置（非空）。
	ImageRootConfigured bool `json:"image_root_configured"`
	// ImageRootExists 图片目录是否存在且为目录。
	ImageRootExists bool `json:"image_root_exists"`
	// ImageRootReadable 图片目录是否可读（真实读目录探测，覆盖 NAS 卷权限错误场景）。
	ImageRootReadable bool `json:"image_root_readable"`
	// AuthEnabled 登录系统是否启用（AUTH_PASSWORD 是否设置；仅布尔，不泄露路径）。
	AuthEnabled bool `json:"auth_enabled"`
	// Initialized 前三项全部为真即视为初始化就绪（auth_enabled 不参与判定——游客模式同样可用）。
	Initialized bool `json:"initialized"`
}
