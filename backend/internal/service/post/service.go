package post

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"unicode/utf8"

	"personal-blog-backend/internal/dao"
	"personal-blog-backend/internal/dao/model"
	"personal-blog-backend/internal/dto"
	"personal-blog-backend/internal/pkg/apperror"
	"personal-blog-backend/internal/pkg/timeutil"

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
	return s.buildPostListResponse(posts, total, page, pageSize)
}

// Search 按关键词搜索文章标题（AND 逻辑），分页返回
func (s *Service) Search(keywords []string, page, pageSize int) (*dto.PostListResponse, error) {
	posts, total, err := s.postDAO.Search(keywords, page, pageSize)
	if err != nil {
		return nil, apperror.WrapInternal(err)
	}
	return s.buildPostListResponse(posts, total, page, pageSize)
}

// buildPostListResponse 将帖子列表 + 批量查询的作者/评论/点赞信息组装为分页响应
func (s *Service) buildPostListResponse(posts []model.Post, total int64, page, pageSize int) (*dto.PostListResponse, error) {
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
	userMap, err := s.userDAO.FindByIDs(userIDs)
	if err != nil {
		log.Printf("[WARN] 批量查询作者信息失败 (userIDs=%v): %v", userIDs, err)
	}
	commentCounts, err := s.commentDAO.CountByPostIDs(postIDs)
	if err != nil {
		log.Printf("[WARN] 批量统计评论数失败 (postIDs=%v): %v", postIDs, err)
	}
	likeCounts, err := s.likeDAO.CountByTargets(model.LikeTargetPost, postIDs)
	if err != nil {
		log.Printf("[WARN] 批量统计点赞数失败 (postIDs=%v): %v", postIDs, err)
	}

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
			Summary:       truncateContent(p.Content, 500),
			AuthorID:      p.UserID,
			AuthorName:    authorName,
			CreatedAt:     timeutil.ToBeijing(p.CreatedAt),
			UpdatedAt:     timeutil.ToBeijing(p.UpdatedAt),
			IsEdited:      p.UpdatedAt.After(p.CreatedAt),
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

// Update 更新文章（路由层已通过 AdminRequired 确保仅管理员可达，
// 管理员有权修改任意文章，因此服务层不再检查所有权）
func (s *Service) Update(id int64, userID int64, req dto.UpdatePostRequest) (*dto.PostDetail, error) {
	post, err := s.postDAO.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.BadRequest("文章不存在")
		}
		return nil, apperror.WrapInternal(err)
	}

	post.Title = req.Title
	post.Content = req.Content
	post.UpdatedAt = timeutil.Now() // 确保 IsEdited 检测生效（北京时间）

	if err := s.postDAO.Update(post); err != nil {
		return nil, apperror.WrapInternal(err)
	}

	return s.toDetail(post, userID)
}

// Delete 删除文章（路由层已通过 AdminRequired 确保仅管理员可达，
// 管理员有权删除任意文章，因此服务层不再检查所有权）
// 所有写入操作在同一个数据库事务中完成，保证数据一致性
func (s *Service) Delete(id int64, userID int64) error {
	// 先验证文章存在（不存在返回友好错误而非 DB 约束错误）
	if _, err := s.postDAO.FindByID(id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperror.BadRequest("文章不存在")
		}
		return apperror.WrapInternal(err)
	}

	// 在事务中执行：收集评论 ID → 清理点赞 → 删除文章（FK 级联删评论）
	return s.postDAO.DB().Transaction(func(tx *gorm.DB) error {
		txPostDAO := dao.NewPostDAO(tx)
		txCommentDAO := dao.NewCommentDAO(tx)
		txLikeDAO := dao.NewLikeDAO(tx)

		// 收集要删除的文章下的所有评论 ID，用于清理点赞
		comments, err := txCommentDAO.ListByPostID(id)
		if err != nil {
			return fmt.Errorf("查询文章评论列表失败 (postID=%d): %w", id, err)
		}
		commentIDs := make([]int64, 0, len(comments))
		for _, c := range comments {
			commentIDs = append(commentIDs, c.ID)
		}

		// 清理文章和评论的所有点赞（likes 表无 FK，需手动清理）
		if err := txLikeDAO.DeleteByTargets(model.LikeTargetPost, []int64{id}); err != nil {
			return fmt.Errorf("清理文章点赞失败 (postID=%d): %w", id, err)
		}
		if len(commentIDs) > 0 {
			if err := txLikeDAO.DeleteByTargets(model.LikeTargetComment, commentIDs); err != nil {
				return fmt.Errorf("清理评论点赞失败 (commentIDs=%v): %w", commentIDs, err)
			}
		}

		// 删除文章（数据库 FK 级联自动删除所有评论及其子回复）
		return txPostDAO.Delete(id)
	})
}

// —— 辅助方法 ——

func (s *Service) toDetail(p *model.Post, viewerID int64) (*dto.PostDetail, error) {
	author, err := s.userDAO.FindByID(p.UserID)
	authorName := ""
	if err == nil {
		authorName = author.Username
	} else {
		log.Printf("[WARN] 查询文章作者失败 (postID=%d, userID=%d): %v", p.ID, p.UserID, err)
	}

	commentsCount, err := s.commentDAO.CountByPostID(p.ID)
	if err != nil {
		log.Printf("[WARN] 统计评论数失败 (postID=%d): %v", p.ID, err)
	}
	likesCount, err := s.likeDAO.CountByTarget(model.LikeTargetPost, p.ID)
	if err != nil {
		log.Printf("[WARN] 统计点赞数失败 (postID=%d): %v", p.ID, err)
	}

	var isLiked bool
	if viewerID > 0 {
		isLiked, err = s.likeDAO.Exists(viewerID, model.LikeTargetPost, p.ID)
		if err != nil {
			log.Printf("[WARN] 检查点赞状态失败 (userID=%d, postID=%d): %v", viewerID, p.ID, err)
		}
	}

	// 统一规范化到北京时间后再比较，避免 Update 流程中新生成的 UpdatedAt（东八区）
	// 与 DB 读回的 CreatedAt（裸 UTC 占位）跨基准比较导致 IsEdited 误判。
	createdAt := timeutil.ToBeijing(p.CreatedAt)
	updatedAt := timeutil.ToBeijing(p.UpdatedAt)

	return &dto.PostDetail{
		ID:            p.ID,
		Title:         p.Title,
		Content:       p.Content,
		AuthorID:      p.UserID,
		AuthorName:    authorName,
		CreatedAt:     createdAt,
		UpdatedAt:     updatedAt,
		IsEdited:      updatedAt.After(createdAt),
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
