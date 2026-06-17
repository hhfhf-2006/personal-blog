package server

import (
	"net/http"

	"personal-blog-backend/internal/svc"

	"github.com/gin-gonic/gin"
)

func NewHTTPServer(ctx *svc.ServiceContext) *gin.Engine {
	r := gin.Default()

	// CORS 跨域中间件：允许前端从不同端口访问后端 API
	r.Use(corsMiddleware())

	RegisterRoutes(r, ctx)

	return r
}

// corsMiddleware 返回一个简单的 CORS 中间件。
// 它允许所有来源的请求（开发阶段够用），并处理 OPTIONS 预检请求。
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// 浏览器在跨域请求前会先发一个 OPTIONS "预检请求"
		// 直接返回 204，告诉浏览器"可以继续"
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}