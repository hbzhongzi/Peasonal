// Package config 负责加载与校验应用配置。
//
// 约定：配置缺失即启动失败，绝不做运行时兜底（例如"没配 JWT_SECRET 就随机生成一个"），
// 否则线上会出现每次重启签名密钥都变、所有 token 集体失效的诡异问题。
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

// Config 是应用全部配置的唯一来源。
type Config struct {
	// --- 应用 ---
	AppEnv          string        `env:"APP_ENV" env-default:"development"`
	HTTPAddr        string        `env:"HTTP_ADDR" env-default:":8080"`
	ReadTimeout     time.Duration `env:"HTTP_READ_TIMEOUT" env-default:"15s"`
	WriteTimeout    time.Duration `env:"HTTP_WRITE_TIMEOUT" env-default:"15s"`
	IdleTimeout     time.Duration `env:"HTTP_IDLE_TIMEOUT" env-default:"60s"`
	ShutdownTimeout time.Duration `env:"SHUTDOWN_TIMEOUT" env-default:"10s"`

	// --- 数据库 ---
	DatabaseURL   string        `env:"DATABASE_URL" env-required:"true"`
	DBMaxOpen     int           `env:"DB_MAX_OPEN" env-default:"25"`
	DBMaxIdle     int           `env:"DB_MAX_IDLE" env-default:"5"`
	DBMaxLifetime time.Duration `env:"DB_MAX_LIFETIME" env-default:"30m"`

	// --- 认证 ---
	JWTSecret string `env:"JWT_SECRET" env-required:"true"`

	// --- 媒体（本地磁盘）---
	MediaDir         string `env:"MEDIA_DIR" env-default:"./data/uploads"`
	MediaURLPrefix   string `env:"MEDIA_URL_PREFIX" env-default:"/uploads"`
	MediaMaxUploadMB int64  `env:"MEDIA_MAX_UPLOAD_MB" env-default:"5"`
	MediaQuotaMB     int64  `env:"MEDIA_QUOTA_MB" env-default:"2048"`

	// --- 站点 ---
	BaseURL      string   `env:"BASE_URL" env-default:"http://localhost:8080"`
	CORSOrigins  []string `env:"CORS_ORIGINS" env-default:"http://localhost:5173"`
	RateLimitRPS int      `env:"RATE_LIMIT_RPS" env-default:"10"`
}

// Load 读取 .env（若存在）再叠加环境变量，环境变量优先级更高。
//
// envFile 解析规则（让从任意目录启动都能找到仓库根的 .env）：
//  1. 显式传入的路径（绝对或相对当前目录）若存在则用它；
//  2. 否则从当前工作目录逐级向上查找名为 .env 的文件，取第一个命中的；
//  3. 都找不到再退回只读环境变量。
func Load(envFile string) (Config, error) {
	var cfg Config

	resolved := resolveEnvFile(envFile)
	if resolved != "" {
		// cleanenv.ReadConfig 会先读文件、再读环境变量（后者覆盖前者）
		if err := cleanenv.ReadConfig(resolved, &cfg); err != nil {
			return cfg, fmt.Errorf("读取配置文件 %s: %w", resolved, err)
		}
		// 把相对路径类配置锚定到 .env 所在目录（仓库根），避免从子目录启动时
		// 把 uploads 等落到错误位置。
		cfg = anchorPaths(cfg, filepath.Dir(resolved))
		return cfg, validate(&cfg)
	}

	if err := cleanenv.ReadEnv(&cfg); err != nil {
		return cfg, fmt.Errorf("读取环境变量: %w", err)
	}
	return cfg, validate(&cfg)
}

// resolveEnvFile 返回实际可用的 .env 路径，找不到返回空串。
func resolveEnvFile(envFile string) string {
	if envFile != "" {
		if abs, err := filepath.Abs(envFile); err == nil {
			if _, err := os.Stat(abs); err == nil {
				return abs
			}
		}
	}
	// 从当前目录向上查找 .env，兼容在 cmd/server 等子目录直接 go run 的场景
	if wd, err := os.Getwd(); err == nil {
		dir := wd
		for {
			candidate := filepath.Join(dir, ".env")
			if _, err := os.Stat(candidate); err == nil {
				return candidate
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}
	return ""
}

// anchorPaths 把相对路径类配置锚定到 root 目录（仓库根，即 .env 所在目录），
// 使从任意子目录启动时数据落点保持一致。
func anchorPaths(cfg Config, root string) Config {
	if root != "" && !filepath.IsAbs(cfg.MediaDir) {
		cfg.MediaDir = filepath.Join(root, cfg.MediaDir)
	}
	return cfg
}

func validate(cfg *Config) error {
	cfg.DatabaseURL = normalizeDSN(cfg.DatabaseURL)

	if cfg.CORSOrigins == nil {
		cfg.CORSOrigins = []string{}
	}
	return nil
}

// normalizeDSN 补齐 MySQL 连接串的三个必要参数。
//
// 少任何一项都会埋雷：
//   - parseTime=true：不加会导致 time.Time 字段扫描直接报错
//   - loc=UTC：配合"全链路存 UTC"的约定（见设计文档 §1.2.2）
//   - charset=utf8mb4：否则 emoji 存不进去
func normalizeDSN(dsn string) string {
	want := map[string]string{
		"parseTime": "true",
		"loc":       "UTC",
		"charset":   "utf8mb4",
	}

	sep := "?"
	if strings.Contains(dsn, "?") {
		sep = "&"
	}

	for k, v := range want {
		if !strings.Contains(dsn, k+"=") {
			dsn += sep + k + "=" + v
			sep = "&"
		}
	}
	return dsn
}

// IsProduction 判断是否生产环境，用于决定 Cookie Secure、日志格式等。
func (c Config) IsProduction() bool { return c.AppEnv == "production" }
