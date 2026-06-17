package model

import "time"

// Post 对应数据库的 posts 表（博客文章）
type Post struct {
	ID        int64     `gorm:"column:id;primaryKey"`
	UserID    int64     `gorm:"column:user_id"`
	Title     string    `gorm:"column:title"`
	Content   string    `gorm:"column:content"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (Post) TableName() string {
	return "posts"
}
