package model

import "time"

// LikeTargetType 是 PostgreSQL 的枚举类型 like_target_type
// 在 Go 里用 string 表示，GORM 会自动转换
type LikeTargetType string

const (
	LikeTargetPost    LikeTargetType = "post"
	LikeTargetComment LikeTargetType = "comment"
)

// Like 对应数据库的 likes 表（点赞记录）
type Like struct {
	ID         int64         `gorm:"column:id;primaryKey"`
	UserID     int64         `gorm:"column:user_id"`
	TargetType LikeTargetType `gorm:"column:target_type"`
	TargetID   int64         `gorm:"column:target_id"`
	CreatedAt  time.Time     `gorm:"column:created_at"`
}

func (Like) TableName() string {
	return "likes"
}
