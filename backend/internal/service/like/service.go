package like

import (
	"personal-blog-backend/internal/dao"
	"personal-blog-backend/internal/dao/model"
	"personal-blog-backend/internal/dto"
	"personal-blog-backend/internal/pkg/apperror"
)

type Service struct {
	likeDAO *dao.LikeDAO
}

func NewService(likeDAO *dao.LikeDAO) *Service {
	return &Service{likeDAO: likeDAO}
}

// TogglePost 切换文章点赞状态，返回操作后的点赞状态和总数
func (s *Service) TogglePost(userID int64, postID int64) (*dto.LikeResponse, error) {
	return s.toggle(userID, model.LikeTargetPost, postID)
}

// ToggleComment 切换评论点赞状态，返回操作后的点赞状态和总数
func (s *Service) ToggleComment(userID int64, commentID int64) (*dto.LikeResponse, error) {
	return s.toggle(userID, model.LikeTargetComment, commentID)
}

func (s *Service) toggle(userID int64, targetType model.LikeTargetType, targetID int64) (*dto.LikeResponse, error) {
	like := &model.Like{
		UserID:     userID,
		TargetType: targetType,
		TargetID:   targetID,
	}

	liked, err := s.likeDAO.Toggle(like)
	if err != nil {
		return nil, apperror.WrapInternal(err)
	}

	count, err := s.likeDAO.CountByTarget(targetType, targetID)
	if err != nil {
		return nil, apperror.WrapInternal(err)
	}

	return &dto.LikeResponse{
		Liked:      liked,
		LikesCount: count,
	}, nil
}
