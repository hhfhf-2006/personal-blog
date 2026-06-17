package middleware

import (
	"net/http"
	"strings"

	"personal-blog-backend/internal/pkg/auth"
	"personal-blog-backend/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

// AuthRequired 验证请求头中的 JWT 令牌。没有令牌或令牌无效 → 401。
func AuthRequired(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" {
			response.Error(c, http.StatusUnauthorized, "请先登录")
			c.Abort()
			return
		}

		// "Bearer <token>" → 取后半部分
		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			response.Error(c, http.StatusUnauthorized, "令牌格式错误")
			c.Abort()
			return
		}

		claims, err := auth.ParseToken(jwtSecret, parts[1])
		if err != nil {
			response.Error(c, http.StatusUnauthorized, "令牌无效或已过期")
			c.Abort()
			return
		}

		// 把用户信息存入上下文，后续 handler 可以读取
		c.Set("userID", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("isAdmin", claims.IsAdmin)

		c.Next()
	}
}

// OptionalAuth 尝试解析 JWT 令牌，但不强制要求。
// 如果携带了有效令牌，就把用户信息存入上下文；没有令牌或令牌无效也继续执行。
func OptionalAuth(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" {
			c.Next()
			return
		}

		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.Next()
			return
		}

		claims, err := auth.ParseToken(jwtSecret, parts[1])
		if err != nil {
			c.Next()
			return
		}

		c.Set("userID", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("isAdmin", claims.IsAdmin)
		c.Next()
	}
}

// AdminRequired 要求当前用户是管理员（在 AuthRequired 之后使用）
func AdminRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		isAdmin, _ := c.Get("isAdmin")
		if isAdmin == nil || !isAdmin.(bool) {
			response.Error(c, http.StatusForbidden, "需要管理员权限")
			c.Abort()
			return
		}
		c.Next()
	}
}
