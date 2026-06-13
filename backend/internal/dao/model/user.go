package model

import "time"

type User struct {
	ID           int64     `gorm:"column:id;primaryKey"`
	Username     string    `gorm:"column:username"`
	Email        string    `gorm:"column:email"`
	PasswordHash string    `gorm:"column:password_hash"`
	IsAdmin      bool      `gorm:"column:is_admin"`
	AvatarURL    *string   `gorm:"column:avatar_url"`
	Bio          *string   `gorm:"column:bio"`
	CreatedAt    time.Time `gorm:"column:created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at"`
}

func (User) TableName() string {
	return "users"
}