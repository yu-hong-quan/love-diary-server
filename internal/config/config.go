// Package config 读取服务运行所需的环境变量。
package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// Config 应用配置，对应 .env 中的键。
type Config struct {
	Port       string // HTTP 监听端口，默认 3000
	UploadDir  string // 图片落盘目录，默认 ./uploads
	DBHost     string // PostgreSQL 主机
	DBPort     string // PostgreSQL 端口
	DBUser     string // 数据库用户名
	DBPassword string // 数据库密码
	DBName     string // 数据库名
	JWTSecret  string // JWT 签名密钥
}

// Load 加载配置：优先读取项目根目录 .env，再合并系统环境变量（Docker 注入）。
func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		Port:       getEnv("PORT", "3000"),
		UploadDir:  getEnv("UPLOAD_DIR", "./uploads"),
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "love-diary-sql"),
		DBPassword: getEnv("DB_PASSWORD", ""),
		DBName:     getEnv("DB_NAME", "love_diary"),
		JWTSecret:  getEnv("JWT_SECRET", "love-diary-secret-key"),
	}
	return cfg, cfg.Validate()
}

// Validate 启动前检查必填项，避免连库错误信息难以理解。
func (c *Config) Validate() error {
	missing := []string{}
	if strings.TrimSpace(c.DBHost) == "" {
		missing = append(missing, "DB_HOST")
	}
	if strings.TrimSpace(c.DBName) == "" {
		missing = append(missing, "DB_NAME")
	}
	if strings.TrimSpace(c.DBUser) == "" {
		missing = append(missing, "DB_USER")
	}
	if strings.TrimSpace(c.DBPassword) == "" {
		missing = append(missing, "DB_PASSWORD")
	}
	if len(missing) > 0 {
		return fmt.Errorf("缺少配置: %s。请复制 .env.example 为 .env 并填写（数据库在远程则 DB_HOST 填服务器 IP，不要填 localhost）", strings.Join(missing, ", "))
	}
	return nil
}

func getEnv(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}
