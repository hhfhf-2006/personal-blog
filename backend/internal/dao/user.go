package dao

import (
	"personal-blog-backend/internal/dao/model"

	"gorm.io/gorm"
)

type UserDAO struct {
	db *gorm.DB
}

func NewUserDAO(db *gorm.DB) *UserDAO {
	return &UserDAO{db: db}
}

// DB 暴露底层 *gorm.DB，用于跨 DAO 的事务编排
func (d *UserDAO) DB() *gorm.DB { return d.db }

func (d *UserDAO) Create(user *model.User) error {
	return d.db.Create(user).Error
}

// Update 用主键更新用户。注意：此方法使用 GORM Updates，
// 只更新非零值字段，不会把 password_hash、is_admin 等误覆盖为零值。
// 如需将某个字段重置为零值（如清空 avatar_url），请用 UpdateField。
// 如需强制写入所有字段（包括零值），请用 UpdateAll。
func (d *UserDAO) Update(user *model.User) error {
	return d.db.Model(user).Updates(user).Error
}

// UpdateAll 强制写入所有字段（包括零值），用于需要将 is_admin 等 bool 设为 false 的场景。
// 注意：会清空所有未显式设置的字段，使用前确保 user struct 包含正确的所有列值。
func (d *UserDAO) UpdateAll(user *model.User) error {
	return d.db.Model(user).Select("*").Updates(user).Error
}

// UpdateField 更新单个字段（包括零值，如清空 avatar_url 或设置 is_admin=false）
func (d *UserDAO) UpdateField(userID int64, column string, value interface{}) error {
	return d.db.Model(&model.User{}).Where("id = ?", userID).Update(column, value).Error
}

// Save 全字段覆盖（仅在你确定 struct 包含所有列的正确值时使用）
func (d *UserDAO) Save(user *model.User) error {
	return d.db.Save(user).Error
}

func (d *UserDAO) FindByEmail(email string) (*model.User, error) {
	var user model.User

	err := d.db.Where("email = ?", email).First(&user).Error
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (d *UserDAO) FindByUsername(username string) (*model.User, error) {
	var user model.User

	err := d.db.Where("username = ?", username).First(&user).Error
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (d *UserDAO) FindByID(id int64) (*model.User, error) {
	var user model.User

	err := d.db.Where("id = ?", id).First(&user).Error
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (d *UserDAO) FindByGithubID(githubID int64) (*model.User, error) {
	var user model.User

	err := d.db.Where("github_id = ?", githubID).First(&user).Error
	if err != nil {
		return nil, err
	}

	return &user, nil
}

// FindAll 查询所有用户，按创建时间倒序
func (d *UserDAO) FindAll() ([]model.User, error) {
	var users []model.User
	err := d.db.Order("created_at DESC").Find(&users).Error
	return users, err
}

// Delete 删除用户（外键 CASCADE 会自动清理关联的 posts/comments/likes/game_scores）
func (d *UserDAO) Delete(id int64) error {
	return d.db.Delete(&model.User{}, id).Error
}

// FindByIDs 批量查询用户，返回 map[id]*User
func (d *UserDAO) FindByIDs(ids []int64) (map[int64]*model.User, error) {
	if len(ids) == 0 {
		return map[int64]*model.User{}, nil
	}
	var users []model.User
	if err := d.db.Where("id IN ?", ids).Find(&users).Error; err != nil {
		return nil, err
	}
	result := make(map[int64]*model.User, len(users))
	for i := range users {
		result[users[i].ID] = &users[i]
	}
	return result, nil
}

// MergePassword 原子合并密码和用户名到已有 GitHub 用户（无密码用户）。
// WHERE password_hash = '' 确保只有尚未设置密码的 GitHub 用户才能被合并，
// 防止并发注册请求互相覆盖（第一个请求成功后第二个请求的 RowsAffected 为 0）。
func (d *UserDAO) MergePassword(userID int64, passwordHash, username string, isAdmin bool) (int64, error) {
	updates := map[string]interface{}{
		"password_hash": passwordHash,
		"username":      username,
		"is_admin":      isAdmin,
	}
	result := d.db.Model(&model.User{}).
		Where("id = ? AND password_hash = ''", userID).
		Updates(updates)
	return result.RowsAffected, result.Error
}