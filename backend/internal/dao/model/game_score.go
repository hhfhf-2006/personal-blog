package model

import "time"

type GameScore struct {
	ID         int64     `gorm:"column:id;primaryKey;autoIncrement"`
	UserID     int64     `gorm:"column:user_id;uniqueIndex:idx_user_game"`
	GameName   string    `gorm:"column:game_name;uniqueIndex:idx_user_game;index:idx_game_score,priority:1"`
	Score      int64     `gorm:"column:score;index:idx_game_score,priority:2,sort:desc"`
	CreatedAt  time.Time `gorm:"column:created_at"`                                                      // 记录创建时间（注册时为 NOW()，不变）
	AchievedAt time.Time `gorm:"column:achieved_at;index:idx_game_score,priority:3,sort:asc"`           // 最佳成绩达成时间
}

func (GameScore) TableName() string {
	return "game_scores"
}
