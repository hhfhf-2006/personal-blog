package server

import (
	"log"
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
	if frontendDir == "" {
		log.Println("[WARN] 未找到前端文件目录，所有静态文件请求将返回 404")
		log.Println("[WARN] 请设置 FRONTEND_DIR 环境变量指向前端文件所在目录")
	} else {
		r.Static("/css", filepath.Join(frontendDir, "css"))
		r.Static("/js", filepath.Join(frontendDir, "js"))
		r.Static("/games", filepath.Join(frontendDir, "games"))

		// HTML 页面映射
		r.StaticFile("/", filepath.Join(frontendDir, "index.html"))
		r.StaticFile("/index.html", filepath.Join(frontendDir, "index.html"))
		r.StaticFile("/post.html", filepath.Join(frontendDir, "post.html"))
		r.StaticFile("/login.html", filepath.Join(frontendDir, "login.html"))
		r.StaticFile("/register.html", filepath.Join(frontendDir, "register.html"))
		r.StaticFile("/new-post.html", filepath.Join(frontendDir, "new-post.html"))
		r.StaticFile("/admin-users.html", filepath.Join(frontendDir, "admin-users.html"))
		r.StaticFile("/profile.html", filepath.Join(frontendDir, "profile.html"))
		r.StaticFile("/2048.html", filepath.Join(frontendDir, "2048.html"))
		r.StaticFile("/leaderboard.html", filepath.Join(frontendDir, "leaderboard.html"))
		r.StaticFile("/katex-test.html", filepath.Join(frontendDir, "katex-test.html"))
	}

	return r
}

// findFrontendDir 找到前端文件所在的目录
// 依次尝试：环境变量 FRONTEND_DIR → 相对于可执行文件 → 常见相对路径
func findFrontendDir() string {
	if d := os.Getenv("FRONTEND_DIR"); d != "" {
		return d
	}

	// 相对于可执行文件所在的目录（兼容 go run / Docker 等不同运行方式）
	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		for _, rel := range []string{"frontend", "../frontend", "../../frontend"} {
			d := filepath.Join(exeDir, rel)
			if info, err := os.Stat(d); err == nil && info.IsDir() {
				return d
			}
		}
	}

	// 相对于当前工作目录（最后的兜底）
	candidates := []string{"../frontend", "./frontend", "../../frontend"}
	for _, d := range candidates {
		if info, err := os.Stat(d); err == nil && info.IsDir() {
			return d
		}
	}
	return ""
}

// corsMiddleware 返回一个简单的 CORS 中间件。
// 不使用 * 通配符，因为 Authorization 头与 * 不兼容（浏览器会拒绝）。
// 改为反射请求的 Origin 头，兼容本地开发和不同端口的前端。
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" {
			c.Header("Access-Control-Allow-Origin", origin)
		} else {
			c.Header("Access-Control-Allow-Origin", "*")
		}
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
