package comment

import (
	"net/http"
	"strconv"

	"personal-blog-backend/internal/dto"
	"personal-blog-backend/internal/pkg/apperror"
	"personal-blog-backend/internal/pkg/response"
	commentservice "personal-blog-backend/internal/service/comment"

	"github.com/gin-gonic/gin"
)

type Controller struct {
	commentService *commentservice.Service
}

func NewController(commentService *commentservice.Service) *Controller {
	return &Controller{commentService: commentService}
}

// Create 发表评论 POST /api/v1/posts/:id/comments
func (ctrl *Controller) Create(c *gin.Context) {
	postID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "文章ID格式错误")
		return
	}

	var req dto.CreateCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误")
		return
	}

	userID := c.GetInt64("userID")
	result, err := ctrl.commentService.Create(postID, userID, req)
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

// List 获取文章评论 GET /api/v1/posts/:id/comments
func (ctrl *Controller) List(c *gin.Context) {
	postID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "文章ID格式错误")
		return
	}

	// 尝试从上下文获取用户 ID（未登录则为 0），用于填充 is_liked 字段
	userID, _ := c.Get("userID")
	var viewerID int64
	if userID != nil {
		viewerID = userID.(int64)
	}

	result, err := ctrl.commentService.ListByPostID(postID, viewerID)
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
