package dto

import "time"

type CreatePostRequest struct {
	Title   string `json:"title" binding:"required,min=1,max=200"`
	Content string `json:"content" binding:"required,min=1"`
}

type UpdatePostRequest struct {
	Title   string `json:"title" binding:"required,min=1,max=200"`
	Content string `json:"content" binding:"required,min=1"`
}

type PostListItem struct {
	ID            int64     `json:"id"`
	Title         string    `json:"title"`
	Summary       string    `json:"summary"`
	AuthorID      int64     `json:"author_id"`
	AuthorName    string    `json:"author_name"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	IsEdited      bool      `json:"is_edited"`     // 是否在创建后被编辑过
	ReadingTime   int       `json:"reading_time"`   // 估算阅读分钟数
	CommentsCount int64     `json:"comments_count"` // 评论数
	LikesCount    int64     `json:"likes_count"`    // 点赞数
}

type PostDetail struct {
	ID            int64     `json:"id"`
	Title         string    `json:"title"`
	Content       string    `json:"content"`
	AuthorID      int64     `json:"author_id"`
	AuthorName    string    `json:"author_name"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	IsEdited      bool      `json:"is_edited"`     // 是否在创建后被编辑过
	ReadingTime   int       `json:"reading_time"`
	CommentsCount int64     `json:"comments_count"`
	LikesCount    int64     `json:"likes_count"`
	IsLiked       bool      `json:"is_liked"` // 当前用户是否已点赞
}

type PostListResponse struct {
	Posts    []PostListItem `json:"posts"`
	Total    int64          `json:"total"`
	Page     int            `json:"page"`
	PageSize int            `json:"page_size"`
}
