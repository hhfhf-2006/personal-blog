package main

import (
	"fmt"
	"log"

	"personal-blog-backend/internal/config"
	"personal-blog-backend/internal/server"
	"personal-blog-backend/internal/svc"

	"github.com/joho/godotenv"
)

func main() {
	// 自动加载 .env 文件（如果存在）
	// 找不到 .env 也不报错——会用代码里的默认值
	if err := godotenv.Load(); err != nil {
		fmt.Println("⚠ 未找到 .env 文件，使用默认配置")
	}

	cfg := config.Load()

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