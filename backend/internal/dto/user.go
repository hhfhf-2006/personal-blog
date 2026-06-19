package dto

import "time"

type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=2,max=50"`
	Email    string `json:"email" binding:"required,email,max=100"`
	Password string `json:"password" binding:"required,min=8,max=100"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8,max=100"`
}

type UpdateUserRequest struct {
	Username string `json:"username" binding:"required,min=2,max=50"`
	Email    string `json:"email" binding:"required,email,max=100"`
	IsAdmin  *bool  `json:"is_admin"` // 指针类型，区分"未传"与"传了 false"
}

// UpdateProfileRequest 普通用户编辑自己的基础信息（昵称、邮箱、可选改密码）
type UpdateProfileRequest struct {
	Username    string `json:"username" binding:"required,min=2,max=50"`
	Email       string `json:"email" binding:"required,email,max=100"`
	OldPassword string `json:"old_password"` // 修改密码时校验旧密码（已设密码的账号必填）
	NewPassword string `json:"new_password"` // 留空表示不修改密码
}

type CreateUserRequest struct {
	Username string `json:"username" binding:"required,min=2,max=50"`
	Email    string `json:"email" binding:"required,email,max=100"`
	Password string `json:"password" binding:"required,min=8,max=100"`
	IsAdmin  bool   `json:"is_admin"` // 默认 false，即普通用户
}

type UserResponse struct {
	ID        int64     `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	IsAdmin   bool      `json:"is_admin"`
	AvatarURL *string   `json:"avatar_url,omitempty"`
	Bio       *string   `json:"bio,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type RegisterResponse struct {
	User UserResponse `json:"user"`
}

type LoginResponse struct {
	Token string       `json:"token"`
	User  UserResponse `json:"user"`
}

type UserListResponse struct {
	Users []UserResponse `json:"users"`
}
