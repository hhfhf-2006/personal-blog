package user

import (
	"errors"

	"personal-blog-backend/internal/dao"
	"personal-blog-backend/internal/dao/model"
	"personal-blog-backend/internal/dto"
	"personal-blog-backend/internal/pkg/apperror"
	"personal-blog-backend/internal/pkg/auth"
	"personal-blog-backend/internal/pkg/password"

	"gorm.io/gorm"
)

type Service struct {
	userDAO   *dao.UserDAO
	jwtSecret string
}

func NewService(userDAO *dao.UserDAO, jwtSecret string) *Service {
	return &Service{
		userDAO:   userDAO,
		jwtSecret: jwtSecret,
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

// Login 验证邮箱和密码，成功则返回 JWT 令牌和用户信息
func (s *Service) Login(req dto.LoginRequest) (*dto.LoginResponse, error) {
	user, err := s.userDAO.FindByEmail(req.Email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.BadRequest("邮箱或密码错误")
		}
		return nil, apperror.WrapInternal(err)
	}

	if !password.Check(req.Password, user.PasswordHash) {
		return nil, apperror.BadRequest("邮箱或密码错误")
	}

	token, err := auth.GenerateToken(s.jwtSecret, user.ID, user.Username, user.IsAdmin)
	if err != nil {
		return nil, apperror.WrapInternal(err)
	}

	return &dto.LoginResponse{
		Token: token,
		User: dto.UserResponse{
			ID:        user.ID,
			Username:  user.Username,
			Email:     user.Email,
			IsAdmin:   user.IsAdmin,
			CreatedAt: user.CreatedAt,
		},
	}, nil
}