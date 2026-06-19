package server

import (
	"time"

	"personal-blog-backend/internal/pkg/middleware"
	"personal-blog-backend/internal/svc"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.Engine, ctx *svc.ServiceContext) {
	// 健康检查端点（用于 Docker HEALTHCHECK / 负载均衡探测）
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	api := r.Group("/api/v1")

	// —— 限流器 ——
	loginLimiter := middleware.NewRateLimiter(10, time.Minute)    // 登录：10次/分钟
	registerLimiter := middleware.NewRateLimiter(3, time.Minute)  // 注册：3次/分钟

	// —— 公开路由（不需要登录）——

	auth := api.Group("/auth")
	{
		auth.POST("/register", registerLimiter.Middleware(), ctx.UserController.Register)
		auth.POST("/login", loginLimiter.Middleware(), ctx.UserController.Login)

		// GitHub OAuth
		auth.GET("/github", ctx.UserController.GitHubLogin)
		auth.GET("/github/callback", ctx.UserController.GitHubCallback)
	}

	// 文章（公开浏览，可选认证以获取个性化数据如 is_liked）
	posts := api.Group("/posts")
	posts.Use(middleware.OptionalAuth(ctx.Config.JWTSecret))
	{
		posts.GET("", ctx.PostController.List)
		posts.GET("/search", ctx.PostController.Search) // 必须在 :id 之前注册
		posts.GET("/:id", ctx.PostController.Detail)
		posts.GET("/:id/comments", ctx.CommentController.List)
	}

	// 游戏排行榜（公开，可选认证以获取个人最佳成绩）
	api.GET("/games/leaderboard", middleware.OptionalAuth(ctx.Config.JWTSecret), ctx.GameController.GetLeaderboard)

	// —— 需要登录的路由 ——

	authRequired := api.Group("")
	authRequired.Use(middleware.AuthRequired(ctx.Config.JWTSecret))
	{
		// 个人中心（编辑本人昵称 / 邮箱 / 密码）
		authRequired.GET("/users/me", ctx.UserController.GetMe)
		authRequired.PUT("/users/me", ctx.UserController.UpdateProfile)

		// 评论
		authRequired.POST("/posts/:id/comments", ctx.CommentController.Create)
		authRequired.PUT("/posts/:id/comments/:commentId", ctx.CommentController.Update)
		authRequired.DELETE("/posts/:id/comments/:commentId", ctx.CommentController.Delete)

		// 点赞
		authRequired.POST("/posts/:id/like", ctx.LikeController.TogglePost)
		authRequired.POST("/posts/:id/comments/:commentId/like", ctx.LikeController.ToggleComment)

		// 游戏分数提交
		authRequired.POST("/games/scores", ctx.GameController.SubmitScore)
		authRequired.GET("/games/scores/mine", ctx.GameController.GetMyScore)
	}

	// —— 需要管理员权限的路由 ——

	adminRequired := api.Group("")
	adminRequired.Use(middleware.AuthRequired(ctx.Config.JWTSecret))
	adminRequired.Use(middleware.AdminRequired())
	{
		// 文章（创建、修改、删除）
		adminRequired.POST("/posts", ctx.PostController.Create)
		adminRequired.PUT("/posts/:id", ctx.PostController.Update)
		adminRequired.DELETE("/posts/:id", ctx.PostController.Delete)

		// 用户管理
		adminRequired.GET("/admin/users", ctx.UserController.ListAll)
		adminRequired.POST("/admin/users", ctx.UserController.Create)
		adminRequired.PUT("/admin/users/:id", ctx.UserController.Update)
		adminRequired.DELETE("/admin/users/:id", ctx.UserController.Delete)
	}
}
