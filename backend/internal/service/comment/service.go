package comment

import (
	"errors"
	"fmt"
	"log"

	"personal-blog-backend/internal/dao"
	"personal-blog-backend/internal/dao/model"
	"personal-blog-backend/internal/dto"
	"personal-blog-backend/internal/pkg/apperror"
	"personal-blog-backend/internal/pkg/timeutil"

	"gorm.io/gorm"
)

type Service struct {
	commentDAO *dao.CommentDAO
	userDAO    *dao.UserDAO
	likeDAO    *dao.LikeDAO
	postDAO    *dao.PostDAO
}

func NewService(commentDAO *dao.CommentDAO, userDAO *dao.UserDAO, likeDAO *dao.LikeDAO, postDAO *dao.PostDAO) *Service {
	return &Service{
		commentDAO: commentDAO,
		userDAO:    userDAO,
		likeDAO:    likeDAO,
		postDAO:    postDAO,
	}
}

// Create 发表评论
func (s *Service) Create(postID int64, userID int64, req dto.CreateCommentRequest) (*dto.CommentResponse, error) {
	// 校验文章是否存在（避免 FK 违规返回 500）
	if _, err := s.postDAO.FindByID(postID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.BadRequest("文章不存在")
		}
		return nil, apperror.WrapInternal(err)
	}

	// 如果是楼中楼回复，校验父评论是否存在且属于同一篇文章
	if req.ParentID != nil {
		parent, err := s.commentDAO.FindByID(*req.ParentID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, apperror.BadRequest("父评论不存在")
			}
			return nil, apperror.WrapInternal(err)
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

	// 查询用户名填充响应（确保 API 一致性，前端无需从 localStorage 补充）
	username := ""
	if user, err := s.userDAO.FindByID(userID); err == nil {
		username = user.Username
	} else {
		log.Printf("[WARN] 查询评论作者失败 (userID=%d): %v", userID, err)
	}

	return &dto.CommentResponse{
		ID:         comment.ID,
		PostID:     comment.PostID,
		UserID:     comment.UserID,
		Username:   username,
		ParentID:   comment.ParentID,
		Content:    comment.Content,
		CreatedAt:  timeutil.ToBeijing(comment.CreatedAt),
		UpdatedAt:  timeutil.ToBeijing(comment.CreatedAt),
		IsEdited:   false,
		LikesCount: 0,
		IsLiked:    false,
	}, nil
}

// Update 编辑评论（仅本人可编辑）。
// 仅修改正文与更新时间，不改变 post_id / parent_id 等结构关系。
// 返回更新后的评论（含真实点赞数与当前用户点赞状态），供前端就地刷新。
func (s *Service) Update(commentID, postID, userID int64, req dto.UpdateCommentRequest) (*dto.CommentResponse, error) {
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
	if comment.UserID != userID {
		return nil, apperror.BadRequest("只能编辑自己的评论")
	}

	comment.Content = req.Content
	comment.UpdatedAt = timeutil.Now() // 北京时间，确保 IsEdited 检测生效
	if err := s.commentDAO.Update(comment); err != nil {
		return nil, apperror.WrapInternal(err)
	}

	// 查询用户名（保持与 Create/List 一致，前端无需从 localStorage 补充）
	username := ""
	if user, err := s.userDAO.FindByID(userID); err == nil {
		username = user.Username
	} else {
		log.Printf("[WARN] 查询评论作者失败 (userID=%d): %v", userID, err)
	}

	// 点赞数与当前用户点赞状态（编辑不影响点赞，但响应需保持完整）
	likesCount, err := s.likeDAO.CountByTarget(model.LikeTargetComment, comment.ID)
	if err != nil {
		log.Printf("[WARN] 统计评论点赞数失败 (commentID=%d): %v", comment.ID, err)
	}
	isLiked, err := s.likeDAO.Exists(userID, model.LikeTargetComment, comment.ID)
	if err != nil {
		log.Printf("[WARN] 检查评论点赞状态失败 (userID=%d, commentID=%d): %v", userID, comment.ID, err)
	}

	// 统一规范化到北京时间后再比较，避免 DB 读回的 CreatedAt（裸 UTC 占位）
	// 与新生成的 UpdatedAt（东八区）跨基准比较导致 IsEdited 误判。
	createdAt := timeutil.ToBeijing(comment.CreatedAt)
	updatedAt := timeutil.ToBeijing(comment.UpdatedAt)

	return &dto.CommentResponse{
		ID:         comment.ID,
		PostID:     comment.PostID,
		UserID:     comment.UserID,
		Username:   username,
		ParentID:   comment.ParentID,
		Content:    comment.Content,
		CreatedAt:  createdAt,
		UpdatedAt:  updatedAt,
		IsEdited:   updatedAt.After(createdAt),
		LikesCount: likesCount,
		IsLiked:    isLiked,
	}, nil
}

// Delete 删除评论（仅本人可删除），同时清理所有子孙评论的点赞
// 所有写入操作在同一个数据库事务中完成，保证数据一致性
func (s *Service) Delete(commentID int64, postID int64, userID int64) error {
	comment, err := s.commentDAO.FindByID(commentID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperror.BadRequest("评论不存在")
		}
		return apperror.WrapInternal(err)
	}

	if comment.PostID != postID {
		return apperror.BadRequest("评论不属于该文章")
	}

	if comment.UserID != userID {
		return apperror.BadRequest("只能删除自己的评论")
	}

	// 在事务中执行：收集子孙评论 → 清理点赞 → 删除评论（FK 级联删回复）
	return s.commentDAO.DB().Transaction(func(tx *gorm.DB) error {
		txCommentDAO := dao.NewCommentDAO(tx)
		txLikeDAO := dao.NewLikeDAO(tx)

		// 收集该评论及其所有子孙评论的 ID，用于清理点赞
		allComments, err := txCommentDAO.ListByPostID(comment.PostID)
		if err != nil {
			return fmt.Errorf("查询文章评论列表失败 (postID=%d): %w", comment.PostID, err)
		}
		descendantIDs := collectDescendants(allComments, commentID)
		allIDs := append([]int64{commentID}, descendantIDs...)

		// 手动清理点赞（likes 表无 FK，不会级联删除）
		if err := txLikeDAO.DeleteByTargets(model.LikeTargetComment, allIDs); err != nil {
			return fmt.Errorf("清理评论点赞失败 (commentIDs=%v): %w", allIDs, err)
		}

		// 删除评论（数据库 FK 级联删除所有子回复）
		return txCommentDAO.Delete(commentID)
	})
}

// collectDescendants 收集某个评论的所有子孙 ID（O(n)，先建索引再递归）
func collectDescendants(all []model.Comment, rootID int64) []int64 {
	// 构建 parentID → children 索引（O(n)）
	children := make(map[int64][]int64, len(all))
	for _, c := range all {
		if c.ParentID != nil {
			children[*c.ParentID] = append(children[*c.ParentID], c.ID)
		}
	}

	// 基于索引递归收集（每个节点只访问一次）
	var ids []int64
	var collect func(parentID int64)
	collect = func(parentID int64) {
		for _, childID := range children[parentID] {
			ids = append(ids, childID)
			collect(childID)
		}
	}
	collect(rootID)
	return ids
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
	userMap, err := s.userDAO.FindByIDs(userIDs)
	if err != nil {
		log.Printf("[WARN] 批量查询评论作者失败 (userIDs=%v): %v", userIDs, err)
	}
	likeCounts, err := s.likeDAO.CountByTargets(model.LikeTargetComment, commentIDs)
	if err != nil {
		log.Printf("[WARN] 批量统计评论点赞数失败 (commentIDs=%v): %v", commentIDs, err)
	}

	var likedMap map[int64]bool
	if viewerID > 0 {
		likedMap, err = s.likeDAO.LikedByUser(viewerID, model.LikeTargetComment, commentIDs)
		if err != nil {
			log.Printf("[WARN] 批量查询用户点赞状态失败 (userID=%d): %v", viewerID, err)
		}
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
			CreatedAt:  timeutil.ToBeijing(c.CreatedAt),
			UpdatedAt:  timeutil.ToBeijing(c.UpdatedAt),
			IsEdited:   c.UpdatedAt.After(c.CreatedAt),
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
