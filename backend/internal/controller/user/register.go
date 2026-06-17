package user

import (
	"net/http"

	"personal-blog-backend/internal/dto"
	"personal-blog-backend/internal/pkg/apperror"
	"personal-blog-backend/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

func (ctrl *Controller) Register(c *gin.Context) {
	var req dto.RegisterRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误")
		return
	}

	result, err := ctrl.userService.Register(req)
	if err != nil {
		// 根据错误类型决定返回的状态码
		if apperror.IsBadRequest(err) {
			// 用户的问题 → 400
			response.Error(c, http.StatusBadRequest, err.Error())
		} else {
			// 服务器的问题 → 500
			response.Error(c, http.StatusInternalServerError, "服务器内部错误")
		}
		return
	}

	response.Success(c, result)
}