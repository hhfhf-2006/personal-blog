package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"personal-blog-backend/internal/config"
	"personal-blog-backend/internal/server"
	"personal-blog-backend/internal/svc"

	"github.com/joho/godotenv"
)

func main() {
	// 自动加载 .env 文件，依次尝试多个目录（向上查找）
	loadEnvFile()

	cfg := config.Load()

	// 安全校验：JWT 密钥不能为空（空密钥意味着令牌可被任意伪造）
	if err := cfg.Validate(); err != nil {
		log.Fatalf("配置错误: %v", err)
	}

	serviceContext, err := svc.NewServiceContext(cfg)
	if err != nil {
		log.Fatalf("初始化服务失败: %v", err)
	}

	r := server.NewHTTPServer(serviceContext)

	addr := ":" + cfg.ServerPort
	fmt.Println("后端服务启动成功，地址：http://localhost" + addr)

	if err := r.Run(addr); err != nil {
		log.Fatalf("启动 HTTP 服务失败: %v", err)
	}
}

// loadEnvFile 自动查找并加载 .env 文件
// 依次尝试：环境变量 ENV_FILE → 当前目录 → 上级目录 → 上上级目录
func loadEnvFile() {
	// 如果用户显式指定了 ENV_FILE，优先使用
	if envFile := os.Getenv("ENV_FILE"); envFile != "" {
		if err := godotenv.Load(envFile); err != nil {
			fmt.Printf("⚠ 指定的 .env 文件不存在: %s\n", envFile)
		}
		return
	}

	// 自动向上查找 .env（支持从 backend/cmd/api 目录运行）
	candidates := []string{".env", "../.env", "../../.env"}
	for _, path := range candidates {
		absPath, _ := filepath.Abs(path)
		if _, err := os.Stat(path); err == nil {
			if err := godotenv.Load(path); err == nil {
				fmt.Printf("✓ 已加载环境变量: %s\n", absPath)
				return
			}
		}
	}

	fmt.Println("⚠ 未找到 .env 文件，使用默认配置")
}