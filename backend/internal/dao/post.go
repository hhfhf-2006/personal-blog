package dao

import (
	"personal-blog-backend/internal/dao/model"

	"gorm.io/gorm"
)

type PostDAO struct {
	db *gorm.DB
}

func NewPostDAO(db *gorm.DB) *PostDAO {
	return &PostDAO{db: db}
}

func (d *PostDAO) Create(post *model.Post) error {
	return d.db.Create(post).Error
}

func (d *PostDAO) FindByID(id int64) (*model.Post, error) {
	var post model.Post
	err := d.db.Where("id = ?", id).First(&post).Error
	if err != nil {
		return nil, err
	}
	return &post, nil
}

func (d *PostDAO) Update(post *model.Post) error {
	return d.db.Save(post).Error
}

func (d *PostDAO) Delete(id int64) error {
	return d.db.Delete(&model.Post{}, id).Error
}

// List 分页查询文章列表，按创建时间倒序
func (d *PostDAO) List(page, pageSize int) ([]model.Post, int64, error) {
	var posts []model.Post
	var total int64

	if err := d.db.Model(&model.Post{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	err := d.db.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&posts).Error
	return posts, total, err
}
