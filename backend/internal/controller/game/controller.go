package game

import (
	"log"
	"net/http"
	"strconv"

	"personal-blog-backend/internal/dto"
	"personal-blog-backend/internal/pkg/apperror"
	"personal-blog-backend/internal/pkg/middleware"
	"personal-blog-backend/internal/pkg/response"
	gameservice "personal-blog-backend/internal/service/game"

	"github.com/gin-gonic/gin"
)

type Controller struct {
	gameService *gameservice.Service
}

func NewController(gameService *gameservice.Service) *Controller {
	return &Controller{gameService: gameService}
}

// SubmitScore 提交游戏分数 POST /api/v1/games/scores
func (ctrl *Controller) SubmitScore(c *gin.Context) {
	var req dto.SubmitScoreRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("[GAME] SubmitScore 参数绑定失败: %v", err)
		response.Error(c, http.StatusBadRequest, "参数错误：game_name 和 score 为必填项")
		return
	}

	userID, err := middleware.GetUserID(c)
	if err != nil {
		log.Printf("[GAME] SubmitScore 获取用户ID失败: %v", err)
		response.Error(c, http.StatusInternalServerError, "服务器内部错误")
		return
	}

	log.Printf("[GAME] SubmitScore 收到请求: userID=%d, game=%s, score=%d", userID, req.GameName, req.Score)

	if err := ctrl.gameService.SubmitScore(userID, req.GameName, req.Score); err != nil {
		log.Printf("[GAME] SubmitScore 业务失败: userID=%d, err=%v", userID, err)
		if apperror.IsBadRequest(err) {
			response.Error(c, http.StatusBadRequest, err.Error())
		} else {
			response.Error(c, http.StatusInternalServerError, "服务器内部错误")
		}
		return
	}

	log.Printf("[GAME] SubmitScore 成功: userID=%d, game=%s, score=%d", userID, req.GameName, req.Score)

	// 提交成功后返回用户当前在数据库中的确认最高分，
	// 前端用此值更新 Best Score 显示，确保与排行榜一致。
	myScore, _ := ctrl.gameService.GetMyScore(userID, req.GameName)
	response.Success(c, myScore)
}

// GetMyScore 获取当前用户在指定游戏中的最高分 GET /api/v1/games/scores/mine?game=xxx
func (ctrl *Controller) GetMyScore(c *gin.Context) {
	gameName := c.DefaultQuery("game", "2048")

	userID, err := middleware.GetUserID(c)
	if err != nil {
		log.Printf("[GAME] GetMyScore 获取用户ID失败: %v", err)
		response.Error(c, http.StatusInternalServerError, "服务器内部错误")
		return
	}

	result, err := ctrl.gameService.GetMyScore(userID, gameName)
	if err != nil {
		log.Printf("[GAME] GetMyScore 查询失败: userID=%d, game=%s, err=%v", userID, gameName, err)
		response.Error(c, http.StatusInternalServerError, "服务器内部错误")
		return
	}

	response.Success(c, result)
}

// GetLeaderboard 获取排行榜 GET /api/v1/games/leaderboard?game=xxx&limit=20
func (ctrl *Controller) GetLeaderboard(c *gin.Context) {
	gameName := c.DefaultQuery("game", "2048")
	limit := 20
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	// 尝试获取已登录用户 ID（可选认证场景）
	var userID int64
	if id, err := middleware.GetUserID(c); err == nil {
		userID = id
	}

	result, err := ctrl.gameService.GetLeaderboard(gameName, limit, userID)
	if err != nil {
		log.Printf("[GAME] GetLeaderboard 查询失败: game=%s, err=%v", gameName, err)
		response.Error(c, http.StatusInternalServerError, "服务器内部错误")
		return
	}

	log.Printf("[GAME] GetLeaderboard: game=%s, entries=%d, totalPlayers=%d", gameName, len(result.Entries), result.TotalPlayers)
	response.Success(c, result)
}
