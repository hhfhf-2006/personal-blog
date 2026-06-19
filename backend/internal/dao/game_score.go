package dao

import (
	"log"
	"time"

	"personal-blog-backend/internal/dao/model"

	"gorm.io/gorm"
)

type GameScoreDAO struct {
	db *gorm.DB
}

func NewGameScoreDAO(db *gorm.DB) *GameScoreDAO {
	return &GameScoreDAO{db: db}
}

func (d *GameScoreDAO) DB() *gorm.DB { return d.db }

// —— 单记录 UPSERT 模型：每个用户每个游戏只保留一条最高分记录 ——

// FindByUserAndGame 查询用户在指定游戏的最佳记录（可能返回 nil）
func (d *GameScoreDAO) FindByUserAndGame(userID int64, gameName string) (*model.GameScore, error) {
	var record model.GameScore
	err := d.db.Where("user_id = ? AND game_name = ?", userID, gameName).First(&record).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &record, nil
}

// Create 插入一条游戏分数记录
func (d *GameScoreDAO) Create(score *model.GameScore) error {
	return d.db.Create(score).Error
}

// UpdateScore 将已有记录的最高分更新为 newScore，同时更新达成时间（achieved_at）。
// created_at 保持记录初始创建时间不变。
//
// WHERE 子句增加 AND score < ? 作为并发安全锁：
// 如果另一个并发请求已经将 score 更新为更大的值，本次 UPDATE 将影响 0 行，
// 从而保证数据库中的 score 永远不会被更小的值覆盖。
func (d *GameScoreDAO) UpdateScore(id int64, newScore int64, achievedAt time.Time) error {
	result := d.db.Model(&model.GameScore{}).
		Where("id = ? AND score < ?", id, newScore).
		Updates(map[string]interface{}{
			"score":       newScore,
			"achieved_at": achievedAt,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		log.Printf("[GAME] UpdateScore 未更新任何行: id=%d, newScore=%d（可能已被并发请求更新为更高分）", id, newScore)
	}
	return nil
}

// —— 排行榜查询 ——

// LeaderboardRow 排行榜查询结果行（JOIN users 获取用户名）
type LeaderboardRow struct {
	UserID     int64     `gorm:"column:user_id"`
	Username   string    `gorm:"column:username"`
	Score      int64     `gorm:"column:score"`
	AchievedAt time.Time `gorm:"column:achieved_at"`
}

// ListByGame 查询指定游戏的排行榜（每个用户一条记录，按分数降序）。
// 过滤 score=0 的记录（用户注册时自动创建的默认记录，表示尚未玩过）。
func (d *GameScoreDAO) ListByGame(gameName string, limit int) ([]LeaderboardRow, error) {
	var rows []LeaderboardRow
	err := d.db.Model(&model.GameScore{}).
		Select("game_scores.user_id, users.username, game_scores.score, game_scores.achieved_at").
		Joins("JOIN users ON users.id = game_scores.user_id").
		Where("game_scores.game_name = ? AND game_scores.score > 0", gameName).
		Order("game_scores.score DESC, game_scores.achieved_at ASC").
		Limit(limit).
		Find(&rows).Error
	return rows, err
}

// CountPlayers 统计玩过指定游戏的独立用户数（score > 0）。
func (d *GameScoreDAO) CountPlayers(gameName string) (int64, error) {
	var count int64
	err := d.db.Model(&model.GameScore{}).
		Where("game_name = ? AND score > 0", gameName).
		Count(&count).Error
	return count, err
}

// —— 用户个人最佳 ——

// UserBestRecord 获取用户在指定游戏中的最佳成绩及其达成时间。
// score=0 表示暂无成绩。
func (d *GameScoreDAO) UserBestRecord(userID int64, gameName string) (int64, time.Time, error) {
	var result struct {
		Score      int64
		AchievedAt time.Time
	}
	err := d.db.Model(&model.GameScore{}).
		Select("score, achieved_at").
		Where("user_id = ? AND game_name = ?", userID, gameName).
		Order("score DESC, achieved_at ASC").
		Limit(1).
		Scan(&result).Error
	if err != nil {
		return 0, time.Time{}, err
	}
	return result.Score, result.AchievedAt, nil
}

// UserRank 获取用户在指定游戏中的排名（1-based）。
// 排名规则：分数更高排更前，分数相同则达成时间更早排更前。
// rank=0 表示尚无成绩。
func (d *GameScoreDAO) UserRank(userID int64, gameName string, bestScore int64, bestAchievedAt time.Time) (int64, error) {
	if bestScore <= 0 {
		return 0, nil
	}

	// 统计有多少个其他玩家的记录优于当前用户
	var rank int64
	err := d.db.Model(&model.GameScore{}).
		Where("game_name = ? AND (score > ? OR (score = ? AND achieved_at < ?))",
			gameName, bestScore, bestScore, bestAchievedAt).
		Count(&rank).Error
	if err != nil {
		return 0, err
	}
	return rank + 1, nil
}
