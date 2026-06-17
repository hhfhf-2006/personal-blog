package server

import (
	"net/http"
	"os"
	"path/filepath"

	"personal-blog-backend/internal/svc"

	"github.com/gin-gonic/gin"
)

func NewHTTPServer(ctx *svc.ServiceContext) *gin.Engine {
	r := gin.Default()

	// CORS 跨域中间件：允许前端从不同端口访问后端 API
	r.Use(corsMiddleware())

	RegisterRoutes(r, ctx)

	// —— 静态文件服务：把前端 HTML/CSS/JS 直接托管 ——
	frontendDir := findFrontendDir()
	if frontendDir != "" {
		r.Static("/css", filepath.Join(frontendDir, "css"))
		r.Static("/js", filepath.Join(frontendDir, "js"))

		// HTML 页面映射
		r.StaticFile("/", filepath.Join(frontendDir, "index.html"))
		r.StaticFile("/index.html", filepath.Join(frontendDir, "index.html"))
		r.StaticFile("/post.html", filepath.Join(frontendDir, "post.html"))
		r.StaticFile("/login.html", filepath.Join(frontendDir, "login.html"))
		r.StaticFile("/register.html", filepath.Join(frontendDir, "register.html"))
		r.StaticFile("/new-post.html", filepath.Join(frontendDir, "new-post.html"))
	}

	return r
}

// findFrontendDir 找到前端文件所在的目录
// 依次尝试：环境变量 FRONTEND_DIR、../frontend（相对 backend/）、./frontend
func findFrontendDir() string {
	if d := os.Getenv("FRONTEND_DIR"); d != "" {
		return d
	}
	candidates := []string{"../frontend", "./frontend"}
	for _, d := range candidates {
		if info, err := os.Stat(d); err == nil && info.IsDir() {
			return d
		}
	}
	return ""
}

// corsMiddleware 返回一个简单的 CORS 中间件。
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
