package thumbnail

import (
	"sync"

	vips "github.com/davidbyttow/govips/v2/vips"
)

// libvips 初始化是进程级一次性操作（govips v2 要求 Startup 全进程只调用一次），
// 用 sync.Once 保证并发安全与幂等。
var (
	vipsOnce    sync.Once
	vipsInitErr error
)

// ensureVipsInitialized 首次调用时初始化 libvips，后续调用直接返回缓存结果。
func ensureVipsInitialized() error {
	vipsOnce.Do(func() {
		vipsInitErr = vips.Startup(nil)
	})
	return vipsInitErr
}
