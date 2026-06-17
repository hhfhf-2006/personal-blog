package post

import (
	"errors"
	"strings"
	"unicode/utf8"

	"personal-blog-backend/internal/dao"
	"personal-blog-backend/internal/dao/model"
	"personal-blog-backend/internal/dto"
	"personal-blog-backend/internal/pkg/apperror"

	"gorm.io/gorm"
)

type Service struct {
	postDAO    *dao.PostDAO
	commentDAO *dao.CommentDAO
	likeDAO    *dao.LikeDAO
	userDAO    *dao.UserDAO
}

func NewService(postDAO *dao.PostDAO, commentDAO *dao.CommentDAO, likeDAO *dao.LikeDAO, userDAO *dao.UserDAO) *Service {
	return &Service{
		postDAO:    postDAO,
		commentDAO: commentDAO,
		likeDAO:    likeDAO,
		userDAO:    userDAO,
	}
}

// Create 创建新文章（需要登录）
func (s *Service) Create(req dto.CreatePostRequest, userID int64) (*dto.PostDetail, error) {
	post := &model.Post{
		UserID:  userID,
		Title:   req.Title,
		Content: req.Content,
	}

	if err := s.postDAO.Create(post); err != nil {
		return nil, apperror.WrapInternal(err)
	}

	return s.toDetail(post, userID)
}

// GetByID 获取文章详情
func (s *Service) GetByID(id int64, userID int64) (*dto.PostDetail, error) {
	post, err := s.postDAO.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.BadRequest("文章不存在")
		}
		return nil, apperror.WrapInternal(err)
	}

	return s.toDetail(post, userID)
}

// List 获取文章列表（分页），使用批量查询避免 N+1 问题
func (s *Service) List(page, pageSize int) (*dto.PostListResponse, error) {
	posts, total, err := s.postDAO.List(page, pageSize)
	if err != nil {
		return nil, apperror.WrapInternal(err)
	}

	if len(posts) == 0 {
		return &dto.PostListResponse{
			Posts:    []dto.PostListItem{},
			Total:    total,
			Page:     page,
			PageSize: pageSize,
		}, nil
	}

	// 收集所有需要的 ID
	userIDs := make([]int64, 0, len(posts))
	postIDs := make([]int64, 0, len(posts))
	seenUsers := make(map[int64]bool)
	for i := range posts {
		postIDs = append(postIDs, posts[i].ID)
		if !seenUsers[posts[i].UserID] {
			userIDs = append(userIDs, posts[i].UserID)
			seenUsers[posts[i].UserID] = true
		}
	}

	// 批量查询：一次性获取所有作者、评论数、点赞数
	userMap, _ := s.userDAO.FindByIDs(userIDs)
	commentCounts, _ := s.commentDAO.CountByPostIDs(postIDs)
	likeCounts, _ := s.likeDAO.CountByTargets(model.LikeTargetPost, postIDs)

	// 组装结果
	items := make([]dto.PostListItem, 0, len(posts))
	for i := range posts {
		p := &posts[i]
		authorName := ""
		if u, ok := userMap[p.UserID]; ok {
			authorName = u.Username
		}

		items = append(items, dto.PostListItem{
			ID:            p.ID,
			Title:         p.Title,
			Summary:       truncateContent(p.Content, 200),
			AuthorID:      p.UserID,
			AuthorName:    authorName,
			CreatedAt:     p.CreatedAt,
			ReadingTime:   estimateReadingTime(p.Content),
			CommentsCount: commentCounts[p.ID],
			LikesCount:    likeCounts[p.ID],
		})
	}

	return &dto.PostListResponse{
		Posts:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// Update 更新文章（仅本人可修改）
func (s *Service) Update(id int64, userID int64, req dto.UpdatePostRequest) (*dto.PostDetail, error) {
	post, err := s.postDAO.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.BadRequest("文章不存在")
		}
		return nil, apperror.WrapInternal(err)
	}

	if post.UserID != userID {
		return nil, apperror.BadRequest("只能修改自己的文章")
	}

	post.Title = req.Title
	post.Content = req.Content

	if err := s.postDAO.Update(post); err != nil {
		return nil, apperror.WrapInternal(err)
	}

	return s.toDetail(post, userID)
}

// Delete 删除文章（仅本人可删除）
func (s *Service) Delete(id int64, userID int64) error {
	post, err := s.postDAO.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperror.BadRequest("文章不存在")
		}
		return apperror.WrapInternal(err)
	}

	if post.UserID != userID {
		return apperror.BadRequest("只能删除自己的文章")
	}

	return s.postDAO.Delete(id)
}

// —— 辅助方法 ——

func (s *Service) toDetail(p *model.Post, viewerID int64) (*dto.PostDetail, error) {
	author, err := s.userDAO.FindByID(p.UserID)
	authorName := ""
	if err == nil {
		authorName = author.Username
	}

	commentsCount, _ := s.commentDAO.CountByPostID(p.ID)
	likesCount, _ := s.likeDAO.CountByTarget(model.LikeTargetPost, p.ID)

	var isLiked bool
	if viewerID > 0 {
		isLiked, _ = s.likeDAO.Exists(viewerID, model.LikeTargetPost, p.ID)
	}

	return &dto.PostDetail{
		ID:            p.ID,
		Title:         p.Title,
		Content:       p.Content,
		AuthorID:      p.UserID,
		AuthorName:    authorName,
		CreatedAt:     p.CreatedAt,
		UpdatedAt:     p.UpdatedAt,
		ReadingTime:   estimateReadingTime(p.Content),
		CommentsCount: commentsCount,
		LikesCount:    likesCount,
		IsLiked:       isLiked,
	}, nil
}

// truncateContent 截取文章前 n 个字符作为摘要（按 rune 计算，支持中文）
func truncateContent(content string, maxLen int) string {
	content = strings.TrimSpace(content)
	if utf8.RuneCountInString(content) <= maxLen {
		return content
	}
	runes := []rune(content)
	return string(runes[:maxLen]) + "…"
}

// estimateReadingTime 估算阅读时间（中英文混合，大约每分钟 400 字）
func estimateReadingTime(content string) int {
	count := utf8.RuneCountInString(content)
	minutes := count / 400
	if minutes < 1 {
		minutes = 1
	}
	return minutes
}
