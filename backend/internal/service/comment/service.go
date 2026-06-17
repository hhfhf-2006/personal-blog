package comment

import (
	"personal-blog-backend/internal/dao"
	"personal-blog-backend/internal/dao/model"
	"personal-blog-backend/internal/dto"
	"personal-blog-backend/internal/pkg/apperror"
)

type Service struct {
	commentDAO *dao.CommentDAO
	userDAO    *dao.UserDAO
	likeDAO    *dao.LikeDAO
}

func NewService(commentDAO *dao.CommentDAO, userDAO *dao.UserDAO, likeDAO *dao.LikeDAO) *Service {
	return &Service{
		commentDAO: commentDAO,
		userDAO:    userDAO,
		likeDAO:    likeDAO,
	}
}

// Create 发表评论
func (s *Service) Create(postID int64, userID int64, req dto.CreateCommentRequest) (*dto.CommentResponse, error) {
	// 如果是楼中楼回复，校验父评论是否存在且属于同一篇文章
	if req.ParentID != nil {
		parent, err := s.commentDAO.FindByID(*req.ParentID)
		if err != nil {
			return nil, apperror.BadRequest("父评论不存在")
		}
		if parent.PostID != postID {
			return nil, apperror.BadRequest("不能跨文章回复评论")
		}
	}

	comment := &model.Comment{
		PostID:   postID,
		UserID:   userID,
		ParentID: req.ParentID,
		Content:  req.Content,
	}

	if err := s.commentDAO.Create(comment); err != nil {
		return nil, apperror.WrapInternal(err)
	}

	// 新创建的评论，点赞数为 0，未被点赞
	return &dto.CommentResponse{
		ID:         comment.ID,
		PostID:     comment.PostID,
		UserID:     comment.UserID,
		Username:   "", // 由前端从 localStorage 获取
		ParentID:   comment.ParentID,
		Content:    comment.Content,
		CreatedAt:  comment.CreatedAt,
		LikesCount: 0,
		IsLiked:    false,
	}, nil
}

// ListByPostID 获取文章的所有评论（树形结构：顶级评论 + 子回复）
// viewerID 为 0 表示未登录，不填充 is_liked 字段
func (s *Service) ListByPostID(postID int64, viewerID int64) ([]dto.CommentResponse, error) {
	comments, err := s.commentDAO.ListByPostID(postID)
	if err != nil {
		return nil, apperror.WrapInternal(err)
	}

	if len(comments) == 0 {
		return []dto.CommentResponse{}, nil
	}

	// 收集所有用户 ID 和评论 ID
	userIDs := make([]int64, 0, len(comments))
	commentIDs := make([]int64, 0, len(comments))
	seenUsers := make(map[int64]bool)
	for _, c := range comments {
		commentIDs = append(commentIDs, c.ID)
		if !seenUsers[c.UserID] {
			userIDs = append(userIDs, c.UserID)
			seenUsers[c.UserID] = true
		}
	}

	// 批量查询：用户信息、点赞数、当前用户点赞状态
	userMap, _ := s.userDAO.FindByIDs(userIDs)
	likeCounts, _ := s.likeDAO.CountByTargets(model.LikeTargetComment, commentIDs)

	var likedMap map[int64]bool
	if viewerID > 0 {
		likedMap, _ = s.likeDAO.LikedByUser(viewerID, model.LikeTargetComment, commentIDs)
	}

	// 转为响应对象
	all := make([]dto.CommentResponse, 0, len(comments))
	for i := range comments {
		c := &comments[i]
		username := ""
		if u, ok := userMap[c.UserID]; ok {
			username = u.Username
		}
		isLiked := false
		if likedMap != nil {
			isLiked = likedMap[c.ID]
		}

		all = append(all, dto.CommentResponse{
			ID:         c.ID,
			PostID:     c.PostID,
			UserID:     c.UserID,
			Username:   username,
			ParentID:   c.ParentID,
			Content:    c.Content,
			CreatedAt:  c.CreatedAt,
			LikesCount: likeCounts[c.ID],
			IsLiked:    isLiked,
		})
	}

	// 构建树形结构：顶级评论下面挂子回复
	replyMap := make(map[int64][]dto.CommentResponse)
	var roots []dto.CommentResponse

	for _, c := range all {
		if c.ParentID == nil {
			roots = append(roots, c)
		} else {
			replyMap[*c.ParentID] = append(replyMap[*c.ParentID], c)
		}
	}

	// 递归填充 replies
	var fillReplies func(comments []dto.CommentResponse) []dto.CommentResponse
	fillReplies = func(comments []dto.CommentResponse) []dto.CommentResponse {
		result := make([]dto.CommentResponse, 0, len(comments))
		for _, c := range comments {
			if replies, ok := replyMap[c.ID]; ok {
				c.Replies = fillReplies(replies)
			}
			result = append(result, c)
		}
		return result
	}

	return fillReplies(roots), nil
}
