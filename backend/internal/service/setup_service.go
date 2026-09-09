package service

import (
	"context"
	"errors"
	"io"
	"os"

	"imageshelf/backend/internal/model"
)

// SetupService 初始化检测业务（SPRINT8 §5.3）：只读探测 IMAGE_ROOT 可用性，
// 不写任何状态、不缓存结果——每次调用实时检测。
type SetupService struct {
	imageRoot    string
	authPassword string
}

// NewSetupService 构造 SetupService。
func NewSetupService(imageRoot, authPassword string) *SetupService {
	return &SetupService{imageRoot: imageRoot, authPassword: authPassword}
}

// Status 执行只读初始化检测。initialized 仅由 IMAGE_ROOT 三项检测决定，
// auth_enabled 不参与判定（游客模式同样可用）。
func (s *SetupService) Status(_ context.Context) model.SetupStatus {
	configured := s.imageRoot != ""
	exists := false
	readable := false
	if configured {
		if info, err := os.Stat(s.imageRoot); err == nil && info.IsDir() {
			exists = true
			readable = s.isReadable(s.imageRoot)
		}
	}
	return model.SetupStatus{
		ImageRootConfigured: configured,
		ImageRootExists:     exists,
		ImageRootReadable:   readable,
		AuthEnabled:         s.authPassword != "",
		Initialized:         configured && exists && readable,
	}
}

// isReadable 真实读权限探测：os.Open 目录 + Readdirnames(1)，
// 覆盖 NFS/卷挂载权限错误场景（Stat 成功但 Open/读失败）。空目录的 io.EOF 视为可读。
func (s *SetupService) isReadable(dir string) bool {
	f, err := os.Open(dir)
	if err != nil {
		return false
	}
	defer f.Close()
	if _, err = f.Readdirnames(1); err != nil && !errors.Is(err, io.EOF) {
		return false
	}
	return true
}
