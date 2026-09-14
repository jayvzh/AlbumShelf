package thumbnail

import (
	"os"
	"runtime"
	"strconv"
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
//
// 线程数必须显式指定：govips 的 Startup(nil) 会走内部 else 分支，将 libvips 并发
// 锁死为 defaultConcurrencyLevel = 1（govips.go），表现为日志恒为
// "threadpool completed with 1 workers"，大图解码无法并行。这里默认取 CPU 核数，
// 可用 VIPS_CONCURRENCY 覆盖（如 NAS 上限制资源占用）。
func ensureVipsInitialized() error {
	vipsOnce.Do(func() {
		workers := runtime.NumCPU()
		if n, err := strconv.Atoi(os.Getenv("VIPS_CONCURRENCY")); err == nil && n >= 1 {
			workers = n
		}
		vipsInitErr = vips.Startup(&vips.Config{ConcurrencyLevel: workers})
	})
	return vipsInitErr
}
