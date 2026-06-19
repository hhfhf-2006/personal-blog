package post

import (
	"net/http"
	"strconv"
	"strings"

	"personal-blog-backend/internal/dto"
	"personal-blog-backend/internal/pkg/apperror"
	"personal-blog-backend/internal/pkg/middleware"
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

	userID, err := middleware.GetUserID(c)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "服务器内部错误")
		return
	}
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

// Search 搜索文章 GET /api/v1/posts/search?q=关键词&page=1&page_size=10
// 按空格拆分为多个关键词，标题必须同时包含所有关键词（AND 逻辑）
func (ctrl *Controller) Search(c *gin.Context) {
	q := strings.TrimSpace(c.Query("q"))
	keywords := strings.Fields(q) // 按空白字符分割，自动过滤空串

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 50 {
		pageSize = 10
	}

	var result *dto.PostListResponse
	var err error

	if len(keywords) == 0 {
		// 无关键词 → 返回空列表，不查全库
		result = &dto.PostListResponse{
			Posts:    []dto.PostListItem{},
			Total:    0,
			Page:     page,
			PageSize: pageSize,
		}
	} else {
		result, err = ctrl.postService.Search(keywords, page, pageSize)
		if err != nil {
			handleError(c, err)
			return
		}
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

	// 尝试从上下文获取用户 ID（未登录则为 0），使用安全类型断言
	uid, _ := middleware.GetUserID(c)

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

	userID, err := middleware.GetUserID(c)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "服务器内部错误")
		return
	}
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

	userID, err := middleware.GetUserID(c)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "服务器内部错误")
		return
	}
	if err := ctrl.postService.Delete(id, userID); err != nil {
		handleError(c, err)
		return
	}

	response.Success(c, nil)
}

func handleError(c *gin.Context, err error) {
	if apperror.IsBadRequest(err) {
		response.Error(c, http.StatusBadRequest, err.Error())
	} else if apperror.IsNotFound(err) {
		response.Error(c, http.StatusNotFound, err.Error())
	} else {
		response.Error(c, http.StatusInternalServerError, "服务器内部错误")
	}
}
