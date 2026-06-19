package like

import (
	"errors"

	"personal-blog-backend/internal/dao"
	"personal-blog-backend/internal/dao/model"
	"personal-blog-backend/internal/dto"
	"personal-blog-backend/internal/pkg/apperror"

	"gorm.io/gorm"
)

type Service struct {
	likeDAO    *dao.LikeDAO
	postDAO    *dao.PostDAO
	commentDAO *dao.CommentDAO
}

func NewService(likeDAO *dao.LikeDAO, postDAO *dao.PostDAO, commentDAO *dao.CommentDAO) *Service {
	return &Service{
		likeDAO:    likeDAO,
		postDAO:    postDAO,
		commentDAO: commentDAO,
	}
}

// TogglePost 切换文章点赞状态，返回操作后的点赞状态和总数
func (s *Service) TogglePost(userID int64, postID int64) (*dto.LikeResponse, error) {
	return s.toggle(userID, model.LikeTargetPost, postID)
}

// ToggleComment 切换评论点赞状态，校验评论属于指定文章后执行点赞切换
func (s *Service) ToggleComment(userID int64, postID int64, commentID int64) (*dto.LikeResponse, error) {
	// 校验评论存在且属于指定文章
	comment, err := s.commentDAO.FindByID(commentID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.BadRequest("评论不存在")
		}
		return nil, apperror.WrapInternal(err)
	}
	if comment.PostID != postID {
		return nil, apperror.BadRequest("评论不属于该文章")
	}
	return s.toggle(userID, model.LikeTargetComment, commentID)
}

func (s *Service) toggle(userID int64, targetType model.LikeTargetType, targetID int64) (*dto.LikeResponse, error) {
	// 校验目标是否存在（避免对不存在的文章/评论点赞）
	if err := s.validateTarget(targetType, targetID); err != nil {
		return nil, err
	}

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

// validateTarget 校验点赞目标是否存在
func (s *Service) validateTarget(targetType model.LikeTargetType, targetID int64) error {
	switch targetType {
	case model.LikeTargetPost:
		_, err := s.postDAO.FindByID(targetID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return apperror.BadRequest("文章不存在")
			}
			return apperror.WrapInternal(err)
		}
	case model.LikeTargetComment:
		_, err := s.commentDAO.FindByID(targetID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return apperror.BadRequest("评论不存在")
			}
			return apperror.WrapInternal(err)
		}
	}
	return nil
}
