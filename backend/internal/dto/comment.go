package dto

import "time"

type CreateCommentRequest struct {
	Content  string `json:"content" binding:"required,min=1,max=2000"`
	ParentID *int64 `json:"parent_id"` // 回复哪条评论，nil 表示顶级评论
}

// UpdateCommentRequest 编辑评论请求（仅允许修改正文，不可改变父子关系）
type UpdateCommentRequest struct {
	Content string `json:"content" binding:"required,min=1,max=2000"`
}

type CommentResponse struct {
	ID         int64             `json:"id"`
	PostID     int64             `json:"post_id"`
	UserID     int64             `json:"user_id"`
	Username   string            `json:"username"`
	ParentID   *int64            `json:"parent_id"`
	Content    string            `json:"content"`
	CreatedAt  time.Time         `json:"created_at"`
	UpdatedAt  time.Time         `json:"updated_at"`
	IsEdited   bool              `json:"is_edited"`   // 是否在创建后被编辑过
	LikesCount int64             `json:"likes_count"`
	IsLiked    bool              `json:"is_liked"`
	Replies    []CommentResponse `json:"replies,omitempty"` // 子回复
}
