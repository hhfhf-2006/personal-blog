package user

import (
	"net/http"
	"strconv"

	"personal-blog-backend/internal/dto"
	"personal-blog-backend/internal/pkg/apperror"
	"personal-blog-backend/internal/pkg/middleware"
	"personal-blog-backend/internal/pkg/oauth"
	"personal-blog-backend/internal/pkg/response"
	userservice "personal-blog-backend/internal/service/user"

	"github.com/gin-gonic/gin"
)

type Controller struct {
	userService  *userservice.Service
	githubOAuth  *oauth.Client
	secureCookie bool // 是否在 OAuth cookie 上设置 Secure 标志（生产 HTTPS 为 true）
}

func NewController(userService *userservice.Service, githubOAuth *oauth.Client, secureCookie bool) *Controller {
	return &Controller{
		userService:  userService,
		githubOAuth:  githubOAuth,
		secureCookie: secureCookie,
	}
}

// ListAll 返回所有用户列表 GET /api/v1/admin/users（仅管理员）
func (ctrl *Controller) ListAll(c *gin.Context) {
	result, err := ctrl.userService.ListAll()
	if err != nil {
		if apperror.IsBadRequest(err) {
			response.Error(c, http.StatusBadRequest, err.Error())
		} else {
			response.Error(c, http.StatusInternalServerError, "服务器内部错误")
		}
		return
	}
	response.Success(c, result)
}

// Create 管理员创建新用户 POST /api/v1/admin/users
func (ctrl *Controller) Create(c *gin.Context) {
	var req dto.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误："+err.Error())
		return
	}

	result, err := ctrl.userService.CreateUser(req)
	if err != nil {
		if apperror.IsBadRequest(err) {
			response.Error(c, http.StatusBadRequest, err.Error())
		} else {
			response.Error(c, http.StatusInternalServerError, "服务器内部错误")
		}
		return
	}
	response.Success(c, result)
}

// Update 管理员编辑用户信息 PUT /api/v1/admin/users/:id
func (ctrl *Controller) Update(c *gin.Context) {
	targetID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的用户 ID")
		return
	}

	operatorID, err := middleware.GetUserID(c)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "服务器内部错误")
		return
	}

	var req dto.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误："+err.Error())
		return
	}

	result, err := ctrl.userService.UpdateUser(operatorID, targetID, req)
	if err != nil {
		if apperror.IsBadRequest(err) {
			response.Error(c, http.StatusBadRequest, err.Error())
		} else if apperror.IsNotFound(err) {
			response.Error(c, http.StatusNotFound, err.Error())
		} else {
			response.Error(c, http.StatusInternalServerError, "服务器内部错误")
		}
		return
	}
	response.Success(c, result)
}

// GetMe 获取当前登录用户自己的资料 GET /api/v1/users/me
func (ctrl *Controller) GetMe(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "服务器内部错误")
		return
	}

	result, err := ctrl.userService.GetByID(userID)
	if err != nil {
		if apperror.IsNotFound(err) {
			response.Error(c, http.StatusNotFound, err.Error())
		} else {
			response.Error(c, http.StatusInternalServerError, "服务器内部错误")
		}
		return
	}
	response.Success(c, result)
}

// UpdateProfile 当前登录用户编辑自己的基础信息 PUT /api/v1/users/me
func (ctrl *Controller) UpdateProfile(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "服务器内部错误")
		return
	}

	var req dto.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误："+err.Error())
		return
	}

	result, err := ctrl.userService.UpdateProfile(userID, req)
	if err != nil {
		if apperror.IsBadRequest(err) {
			response.Error(c, http.StatusBadRequest, err.Error())
		} else if apperror.IsNotFound(err) {
			response.Error(c, http.StatusNotFound, err.Error())
		} else {
			response.Error(c, http.StatusInternalServerError, "服务器内部错误")
		}
		return
	}
	response.Success(c, result)
}

// Delete 管理员删除用户 DELETE /api/v1/admin/users/:id
func (ctrl *Controller) Delete(c *gin.Context) {
	targetID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的用户 ID")
		return
	}

	operatorID, err := middleware.GetUserID(c)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "服务器内部错误")
		return
	}

	if err := ctrl.userService.DeleteUser(operatorID, targetID); err != nil {
		if apperror.IsBadRequest(err) {
			response.Error(c, http.StatusBadRequest, err.Error())
		} else if apperror.IsNotFound(err) {
			response.Error(c, http.StatusNotFound, err.Error())
		} else {
			response.Error(c, http.StatusInternalServerError, "服务器内部错误")
		}
		return
	}
	response.Success(c, nil)
}