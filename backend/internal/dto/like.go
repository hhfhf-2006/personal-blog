package dto

type LikeResponse struct {
	Liked      bool  `json:"liked"`       // 操作后是否已点赞
	LikesCount int64 `json:"likes_count"` // 操作后的点赞总数
}
