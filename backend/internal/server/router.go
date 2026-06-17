package server

import (
	"personal-blog-backend/internal/svc"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.Engine, ctx *svc.ServiceContext) {
	api := r.Group("/api/v1")

	auth := api.Group("/auth")
	{
		auth.POST("/register", ctx.UserController.Register)
	}
}