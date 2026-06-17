package user

import (
	"errors"

	"personal-blog-backend/internal/dao"
	"personal-blog-backend/internal/dao/model"
	"personal-blog-backend/internal/dto"
	"personal-blog-backend/internal/pkg/apperror"
	"personal-blog-backend/internal/pkg/password"

	"gorm.io/gorm"
)

type Service struct {
	userDAO *dao.UserDAO
}

func NewService(userDAO *dao.UserDAO) *Service {
	return &Service{
		userDAO: userDAO,
	}
}

func (s *Service) Register(req dto.RegisterRequest) (*dto.RegisterResponse, error) {
	// —— 检查邮箱是否已被注册 ——
	_, err := s.userDAO.FindByEmail(req.Email)
	if err == nil {
		// 找到了记录 → 邮箱已占用，属于业务错误（400）
		return nil, apperror.BadRequest("邮箱已经被注册")
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		// 不是"没找到"的错误 → 数据库本身出问题了（500）
		return nil, apperror.WrapInternal(err)
	}

	// —— 检查用户名是否已被注册 ——
	_, err = s.userDAO.FindByUsername(req.Username)
	if err == nil {
		return nil, apperror.BadRequest("用户名已经被注册")
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperror.WrapInternal(err)
	}

	// —— 密码加密 ——
	hashedPassword, err := password.Hash(req.Password)
	if err != nil {
		return nil, apperror.WrapInternal(err)
	}

	// —— 写入数据库 ——
	newUser := &model.User{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: hashedPassword,
		IsAdmin:      false,
	}

	if err := s.userDAO.Create(newUser); err != nil {
		return nil, apperror.WrapInternal(err)
	}

	return &dto.RegisterResponse{
		User: dto.UserResponse{
			ID:        newUser.ID,
			Username:  newUser.Username,
			Email:     newUser.Email,
			IsAdmin:   newUser.IsAdmin,
			CreatedAt: newUser.CreatedAt,
		},
	}, nil
}