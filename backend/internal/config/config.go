// Package config 负责从环境变量加载服务运行配置。
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config 保存服务运行配置。全部字段来自环境变量，未设置时使用默认值。
type Config struct {
	// ImageRoot 允许浏览的图片根目录（唯一来源）。
	ImageRoot string
	// DataDir 数据目录（缩略图缓存 + SQLite，后续 Sprint 使用）。
	DataDir string
	// Port HTTP 监听端口。
	Port string
	// AuthUsername 管理员用户名（认证未启用时可为空）。
	AuthUsername string
	// AuthPassword 管理员密码；为空表示认证未启用（Sprint 0-6 行为）。
	AuthPassword string
	// SessionMaxAge 会话有效期（秒），默认 7 天。
	SessionMaxAge int
	// ThumbConcurrency 生成调度器并发上限（同时执行的 libvips 任务数），默认 2。
	ThumbConcurrency int
	// ThumbWarmer 是否启用后台缩略图预热器（空闲低优先级补齐全库 thumbs），默认启用。
	ThumbWarmer bool
}

// Load 从环境变量读取配置。
//
//	IMAGE_ROOT        默认 /images
//	DATA_DIR          默认 /data
//	PORT              默认 8080
//	AUTH_USERNAME     默认空
//	AUTH_PASSWORD     默认空（为空时认证未启用）
//	SESSION_MAX_AGE   默认 168h（7 天），非法值回退默认
//	THUMB_CONCURRENCY 默认 2，非正整数回退默认
//	THUMB_WARMER       默认 true；设为 0/false/off 关闭后台预热
func Load() *Config {
	return &Config{
		ImageRoot:        getEnv("IMAGE_ROOT", "/images"),
		DataDir:          getEnv("DATA_DIR", "/data"),
		Port:             getEnv("PORT", "8080"),
		AuthUsername:     getEnv("AUTH_USERNAME", ""),
		AuthPassword:     getEnv("AUTH_PASSWORD", ""),
		SessionMaxAge:    getSessionMaxAge(),
		ThumbConcurrency: getEnvInt("THUMB_CONCURRENCY", 2),
		ThumbWarmer:      getEnvBool("THUMB_WARMER", true),
	}
}

// AuthEnabled 返回认证是否启用：AUTH_PASSWORD 非空即启用。
func (c *Config) AuthEnabled() bool {
	return c.AuthPassword != ""
}

// Addr 返回 HTTP 监听地址，形如 ":8080"。
func (c *Config) Addr() string {
	return fmt.Sprintf(":%s", c.Port)
}

// getEnv 读取环境变量，为空时返回 fallback。
func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// getEnvInt 读取整型环境变量，未设置或非法/非正值时返回 fallback。
func getEnvInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 1 {
		return fallback
	}
	return n
}

// getEnvBool 读取布尔环境变量；仅显式的假值（0/false/off/no，大小写不敏感）为 false，
// 其余（含未设置）均为 true。
func getEnvBool(key string, fallback bool) bool {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	switch strings.ToLower(v) {
	case "0", "false", "off", "no":
		return false
	default:
		return true
	}
}

// getSessionMaxAge 解析 SESSION_MAX_AGE（Go duration 字符串），默认 7 天，非法值回退默认。
func getSessionMaxAge() int {
	const fallback = 168 * time.Hour
	v := os.Getenv("SESSION_MAX_AGE")
	if v == "" {
		return int(fallback.Seconds())
	}
	d, err := time.ParseDuration(v)
	if err != nil || d <= 0 {
		return int(fallback.Seconds())
	}
	return int(d.Seconds())
}
