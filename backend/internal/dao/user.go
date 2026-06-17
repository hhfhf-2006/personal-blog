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