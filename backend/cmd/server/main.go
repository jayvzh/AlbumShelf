// main 是服务入口，只做三件事：config.Load() → app.New() → app.Run()。
//
// 支持子命令：
//
//	server            启动 HTTP 服务（默认行为）
//	server healthcheck  探测本机 /api/v1/health，用于 Docker HEALTHCHECK，
//	                    避免在运行时镜像中安装 curl/wget（减小体积）。
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"imageshelf/backend/internal/app"
	"imageshelf/backend/internal/config"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		os.Exit(runHealthcheck())
	}

	cfg := config.Load()

	a := app.New(cfg)

	if err := a.Run(); err != nil {
		log.Fatalf("server exited: %v", err)
	}
}

// runHealthcheck 向本机 /api/v1/health 发送 GET 请求，成功返回 0，否则返回 1。
// 端口取自 PORT 环境变量（默认 8080），与服务监听端口保持一致。
func runHealthcheck() int {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	url := fmt.Sprintf("http://localhost:%s/api/v1/health", port)

	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return 1
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 1
	}
	return 0
}
