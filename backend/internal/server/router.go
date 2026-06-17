package server

import (
	"personal-blog-backend/internal/pkg/middleware"
	"personal-blog-backend/internal/svc"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.Engine, ctx *svc.ServiceContext) {
	api := r.Group("/api/v1")

	// —— 公开路由（不需要登录）——

	auth := api.Group("/auth")
	{
		auth.POST("/register", ctx.UserController.Register)
		auth.POST("/login", ctx.UserController.Login)
	}

	// 文章（公开浏览，可选认证以获取个性化数据如 is_liked）
	posts := api.Group("/posts")
	posts.Use(middleware.OptionalAuth(ctx.Config.JWTSecret))
	{
		posts.GET("", ctx.PostController.List)
		posts.GET("/:id", ctx.PostController.Detail)
		posts.GET("/:id/comments", ctx.CommentController.List)
	}

	// —— 需要登录的路由 ——

	authRequired := api.Group("")
	authRequired.Use(middleware.AuthRequired(ctx.Config.JWTSecret))
	{
		// 文章（创建、修改、删除）
		authRequired.POST("/posts", ctx.PostController.Create)
		authRequired.PUT("/posts/:id", ctx.PostController.Update)
		authRequired.DELETE("/posts/:id", ctx.PostController.Delete)

		// 评论
		authRequired.POST("/posts/:id/comments", ctx.CommentController.Create)

		// 点赞
		authRequired.POST("/posts/:id/like", ctx.LikeController.TogglePost)
		authRequired.POST("/posts/:id/comments/:commentId/like", ctx.LikeController.ToggleComment)
	}
}
