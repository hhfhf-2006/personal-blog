package dao

import (
	"personal-blog-backend/internal/dao/model"

	"gorm.io/gorm"
)

type CommentDAO struct {
	db *gorm.DB
}

func NewCommentDAO(db *gorm.DB) *CommentDAO {
	return &CommentDAO{db: db}
}

func (d *CommentDAO) Create(comment *model.Comment) error {
	return d.db.Create(comment).Error
}

// FindByID 根据 ID 查找单条评论
func (d *CommentDAO) FindByID(id int64) (*model.Comment, error) {
	var comment model.Comment
	err := d.db.Where("id = ?", id).First(&comment).Error
	if err != nil {
		return nil, err
	}
	return &comment, nil
}

// ListByPostID 查询某篇文章的所有评论，按时间正序（早→晚）
func (d *CommentDAO) ListByPostID(postID int64) ([]model.Comment, error) {
	var comments []model.Comment
	err := d.db.Where("post_id = ?", postID).Order("created_at ASC").Find(&comments).Error
	return comments, err
}

// CountByPostID 统计某篇文章的评论数
func (d *CommentDAO) CountByPostID(postID int64) (int64, error) {
	var count int64
	err := d.db.Model(&model.Comment{}).Where("post_id = ?", postID).Count(&count).Error
	return count, err
}

// CountByPostIDs 批量统计多篇文章的评论数，返回 map[postID]count
func (d *CommentDAO) CountByPostIDs(postIDs []int64) (map[int64]int64, error) {
	if len(postIDs) == 0 {
		return map[int64]int64{}, nil
	}

	type row struct {
		PostID int64
		Count  int64
	}

	var rows []row
	err := d.db.Model(&model.Comment{}).
		Select("post_id, COUNT(*) AS count").
		Where("post_id IN ?", postIDs).
		Group("post_id").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}

	result := make(map[int64]int64, len(rows))
	for _, r := range rows {
		result[r.PostID] = r.Count
	}
	return result, nil
}
