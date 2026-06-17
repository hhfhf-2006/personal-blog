package post

import (
	"net/http"
	"strconv"

	"personal-blog-backend/internal/dto"
	"personal-blog-backend/internal/pkg/apperror"
	"personal-blog-backend/internal/pkg/response"
	postservice "personal-blog-backend/internal/service/post"

	"github.com/gin-gonic/gin"
)

type Controller struct {
	postService *postservice.Service
}

func NewController(postService *postservice.Service) *Controller {
	return &Controller{postService: postService}
}

// Create 创建文章 POST /api/v1/posts
func (ctrl *Controller) Create(c *gin.Context) {
	var req dto.CreatePostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误")
		return
	}

	userID := c.GetInt64("userID")
	result, err := ctrl.postService.Create(req, userID)
	if err != nil {
		handleError(c, err)
		return
	}

	response.Success(c, result)
}

// List 文章列表 GET /api/v1/posts?page=1&page_size=10
func (ctrl *Controller) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 50 {
		pageSize = 10
	}

	result, err := ctrl.postService.List(page, pageSize)
	if err != nil {
		handleError(c, err)
		return
	}

	response.Success(c, result)
}

// Detail 文章详情 GET /api/v1/posts/:id
func (ctrl *Controller) Detail(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "文章ID格式错误")
		return
	}

	// 尝试从上下文获取用户 ID（未登录则为 0）
	userID, _ := c.Get("userID")
	var uid int64
	if userID != nil {
		uid = userID.(int64)
	}

	result, err := ctrl.postService.GetByID(id, uid)
	if err != nil {
		handleError(c, err)
		return
	}

	response.Success(c, result)
}

// Update 更新文章 PUT /api/v1/posts/:id
func (ctrl *Controller) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "文章ID格式错误")
		return
	}

	var req dto.UpdatePostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误")
		return
	}

	userID := c.GetInt64("userID")
	result, err := ctrl.postService.Update(id, userID, req)
	if err != nil {
		handleError(c, err)
		return
	}

	response.Success(c, result)
}

// Delete 删除文章 DELETE /api/v1/posts/:id
func (ctrl *Controller) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "文章ID格式错误")
		return
	}

	userID := c.GetInt64("userID")
	if err := ctrl.postService.Delete(id, userID); err != nil {
		handleError(c, err)
		return
	}

	response.Success(c, nil)
}

func handleError(c *gin.Context, err error) {
	if apperror.IsBadRequest(err) {
		response.Error(c, http.StatusBadRequest, err.Error())
	} else {
		response.Error(c, http.StatusInternalServerError, "服务器内部错误")
	}
}
