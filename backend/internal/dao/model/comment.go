package model

import "time"

// Comment 对应数据库的 comments 表（评论）
// parent_id 可为空，表示"顶级评论"还是"楼中楼回复"
type Comment struct {
	ID        int64     `gorm:"column:id;primaryKey"`
	PostID    int64     `gorm:"column:post_id"`
	UserID    int64     `gorm:"column:user_id"`
	ParentID  *int64    `gorm:"column:parent_id"` // 用指针，NULL 表示顶级评论
	Content   string    `gorm:"column:content"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (Comment) TableName() string {
	return "comments"
}
