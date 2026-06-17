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

func (d *UserDAO) Create(user *model.User) error {
	return d.db.Create(user).Error
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