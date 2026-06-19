package dto

import "time"

// SubmitScoreRequest 提交游戏分数请求
type SubmitScoreRequest struct {
	GameName string `json:"game_name" binding:"required"`
	Score    int64  `json:"score" binding:"required,min=1"`
}

// LeaderboardEntry 排行榜单条记录
type LeaderboardEntry struct {
	Rank       int64     `json:"rank"`
	Username   string    `json:"username"`
	Score      int64     `json:"score"`
	AchievedAt time.Time `json:"achieved_at"`
}

// LeaderboardResponse 排行榜响应
type LeaderboardResponse struct {
	GameName     string             `json:"game_name"`
	Entries      []LeaderboardEntry `json:"entries"`
	TotalPlayers int64              `json:"total_players"`
	UserBest     *UserBest          `json:"user_best,omitempty"` // 仅登录用户返回
}

// UserBest 当前用户在该游戏中的最佳成绩
type UserBest struct {
	Score int64 `json:"score"`
	Rank  int64 `json:"rank"` // 0 表示尚无成绩
}

// MyScoreResponse 当前用户在指定游戏中的最高分（用于 GET /games/scores/mine 和 POST /games/scores 响应）
type MyScoreResponse struct {
	GameName string `json:"game_name"`
	Score    int64  `json:"score"`
	Rank     int64  `json:"rank"` // 0 表示尚无成绩
}
