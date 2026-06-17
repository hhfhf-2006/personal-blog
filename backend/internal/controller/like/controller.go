package like

import (
	"net/http"
	"strconv"

	"personal-blog-backend/internal/pkg/apperror"
	"personal-blog-backend/internal/pkg/response"
	likeservice "personal-blog-backend/internal/service/like"

	"github.com/gin-gonic/gin"
)

type Controller struct {
	likeService *likeservice.Service
}

func NewController(likeService *likeservice.Service) *Controller {
	return &Controller{likeService: likeService}
}

// TogglePost 切换文章点赞 POST /api/v1/posts/:id/like
func (ctrl *Controller) TogglePost(c *gin.Context) {
	postID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "文章ID格式错误")
		return
	}

	userID := c.GetInt64("userID")
	result, err := ctrl.likeService.TogglePost(userID, postID)
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

// ToggleComment 切换评论点赞 POST /api/v1/posts/:id/comments/:commentId/like
func (ctrl *Controller) ToggleComment(c *gin.Context) {
	commentID, err := strconv.ParseInt(c.Param("commentId"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "评论ID格式错误")
		return
	}

	userID := c.GetInt64("userID")
	result, err := ctrl.likeService.ToggleComment(userID, commentID)
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
