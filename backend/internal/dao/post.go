package dao

import (
	"personal-blog-backend/internal/dao/model"
	"strings"

	"gorm.io/gorm"
)

type PostDAO struct {
	db *gorm.DB
}

func NewPostDAO(db *gorm.DB) *PostDAO {
	return &PostDAO{db: db}
}

// DB 暴露底层 *gorm.DB，用于跨 DAO 的事务编排
func (d *PostDAO) DB() *gorm.DB { return d.db }

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

// Update 更新文章的标题、内容和更新时间。
// 使用 Select 指定要更新的列，避免 Save 的全字段覆盖和 upsert 行为。
func (d *PostDAO) Update(post *model.Post) error {
	return d.db.Model(post).Select("title", "content", "updated_at").Updates(post).Error
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

// escapeLikeWildcards 转义 LIKE/ILIKE 模式中的通配符（%、_）和转义符（\），
// 防止用户输入的特殊字符被解释为 LIKE 通配符。
func escapeLikeWildcards(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`) // 先转义反斜杠
	s = strings.ReplaceAll(s, `%`, `\%`)
	s = strings.ReplaceAll(s, `_`, `\_`)
	return s
}

// Search 按关键词搜索文章标题（AND 逻辑），每个关键词必须精确出现在标题中
// 关键词之间无顺序要求；按创建时间倒序分页返回。
// 关键词中的 LIKE 通配符（%、_）会被转义，防止用户用 % 匹配全部文章。
func (d *PostDAO) Search(keywords []string, page, pageSize int) ([]model.Post, int64, error) {
	query := d.db.Model(&model.Post{})
	for _, kw := range keywords {
		escaped := escapeLikeWildcards(kw)
		query = query.Where("title ILIKE ? ESCAPE '\\'", "%"+escaped+"%")
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	var posts []model.Post
	err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&posts).Error
	return posts, total, err
}
