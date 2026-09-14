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
// "threadpool completed with 1 workers"，大图解码无法并行。取值见 EffectiveVipsConcurrency。
func ensureVipsInitialized() error {
	vipsOnce.Do(func() {
		vipsInitErr = vips.Startup(&vips.Config{ConcurrencyLevel: EffectiveVipsConcurrency()})
	})
	return vipsInitErr
}

// EffectiveVipsConcurrency 返回实际生效的 libvips 线程数：
//  1. VIPS_CONCURRENCY 显式设置（≥1 的整数）→ 直接采用（用户覆盖，启动自检
//     在乘积超过可用核数时打 WARN 记录在案）；
//  2. 未设置 → 按"调度器并发 × vips 线程 ≈ 可用核数"自动配平：
//     GOMAXPROCS / schedulerConcurrency（至少 1）。四核 NAS 默认
//     THUMB_CONCURRENCY=2 → vips=2，乘积 4=核数；八核 → 2×4=8，自动适配。
//
// 用 GOMAXPROCS(0) 而非 NumCPU：容器配额限核时（Go 1.25 起自动感知 cgroup）
// 不会按宿主机核数过配。惰性初始化发生在首个生成请求，此时 app 组装早已完成，
// schedulerConcurrency 已就绪。
func EffectiveVipsConcurrency() int {
	if n, err := strconv.Atoi(os.Getenv("VIPS_CONCURRENCY")); err == nil && n >= 1 {
		return n
	}
	c := schedulerConcurrency
	if c < 1 {
		c = 1
	}
	if w := runtime.GOMAXPROCS(0) / c; w >= 1 {
		return w
	}
	return 1
}
