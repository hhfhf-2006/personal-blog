package user

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"unicode"

	"personal-blog-backend/internal/dao"
	"personal-blog-backend/internal/dao/model"
	"personal-blog-backend/internal/dto"
	"personal-blog-backend/internal/pkg/apperror"
	"personal-blog-backend/internal/pkg/auth"
	"personal-blog-backend/internal/pkg/oauth"
	"personal-blog-backend/internal/pkg/password"

	"gorm.io/gorm"
)

// —— 管理员硬编码信息（可公开） ——
const (
	adminGitHubLogin = "hhfhf-2006"       // GitHub 用户名，匹配后自动成为管理员
	adminName        = "灰化肥发灰"          // 管理员显示名称
	adminEmailHard   = "183976823@qq.com" // 管理员邮箱
)

type Service struct {
	userDAO       *dao.UserDAO
	jwtSecret     string
	adminEmail    string
	adminUsername string
	adminPassword string
}

func NewService(userDAO *dao.UserDAO, jwtSecret string, adminEmail string, adminUsername string, adminPassword string) *Service {
	return &Service{
		userDAO:       userDAO,
		jwtSecret:     jwtSecret,
		adminEmail:    adminEmail,
		adminUsername: adminUsername,
		adminPassword: adminPassword,
	}
}

func (s *Service) Register(req dto.RegisterRequest) (*dto.RegisterResponse, error) {
	// —— 密码强度校验（提前，避免无效操作）——
	if !isPasswordStrong(req.Password) {
		return nil, apperror.BadRequest("密码至少 8 位，且必须同时包含字母和数字")
	}

	// —— 检查邮箱 ——
	existingUser, err := s.userDAO.FindByEmail(req.Email)
	if err == nil {
		// 邮箱已存在
		if existingUser.PasswordHash != "" {
			// 已有密码 → 该邮箱已通过密码注册，拒绝重复注册
			return nil, apperror.BadRequest("邮箱已经被注册")
		}
		// GitHub 用户（无密码）：补充密码和用户名，完成"merge"
		hashedPassword, err := password.Hash(req.Password)
		if err != nil {
			return nil, apperror.WrapInternal(err)
		}
		// admin 邮箱：强制使用固定用户名，确保 is_admin
		mergeUsername := req.Username
		mergeIsAdmin := false
		if req.Email == s.adminEmail {
			mergeUsername = s.adminUsername
			mergeIsAdmin = true
		}
		rowsAffected, err := s.userDAO.MergePassword(existingUser.ID, hashedPassword, mergeUsername, mergeIsAdmin)
		if err != nil {
			return nil, apperror.WrapInternal(err)
		}
		if rowsAffected == 0 {
			// 并发场景：另一个请求已经完成了合并
			return nil, apperror.BadRequest("该账号已被其他请求激活，请直接登录")
		}
		existingUser.PasswordHash = hashedPassword
		existingUser.Username = mergeUsername
		existingUser.IsAdmin = mergeIsAdmin
		return &dto.RegisterResponse{User: toUserResponse(existingUser)}, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
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
	isAdmin := req.Email == s.adminEmail
	username := req.Username
	if isAdmin {
		username = s.adminUsername // admin 强制使用固定用户名
	}
	newUser := &model.User{
		Username:     username,
		Email:        req.Email,
		PasswordHash: hashedPassword,
		IsAdmin:      isAdmin,
	}

	if err := s.userDAO.Create(newUser); err != nil {
		// UNIQUE 约束冲突 → 邮箱或用户名被并发请求抢先注册
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, apperror.BadRequest("邮箱或用户名已被注册")
		}
		return nil, apperror.WrapInternal(err)
	}

	return &dto.RegisterResponse{User: toUserResponse(newUser)}, nil
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

	// GitHub OAuth 用户没有密码，无法通过邮箱登录
	if user.PasswordHash == "" {
		return nil, apperror.BadRequest("该账号使用 GitHub 登录，请点击 GitHub 登录按钮")
	}

	if !password.Check(req.Password, user.PasswordHash) {
		return nil, apperror.BadRequest("邮箱或密码错误")
	}

	token, err := auth.GenerateToken(s.jwtSecret, user.ID, user.Username, user.IsAdmin)
	if err != nil {
		return nil, apperror.WrapInternal(err)
	}

	return &dto.LoginResponse{Token: token, User: toUserResponse(user)}, nil
}

// LoginByGithub 通过 GitHub OAuth 信息查找或创建用户，返回 JWT 令牌。
//
// 查找顺序：
//  1. 按 github_id 查 → 老 GitHub 用户，直接登录
//  2. 按邮箱查 → 已通过密码注册的同邮箱用户，链接 GitHub ID 到已有账号
//  3. 都没找到 → 创建新用户（admin 邮箱自动获得管理员权限和固定用户名）
func (s *Service) LoginByGithub(ghUser *oauth.GitHubUser) (*dto.LoginResponse, error) {
	// —— 0. 硬编码管理员匹配：GitHub 用户名为 hhfhf-2006 → 自动成为站长 ——
	if ghUser.Login == adminGitHubLogin {
		return s.loginOrCreateAdmin(ghUser)
	}

	// 1. 先按 github_id 查找
	user, err := s.userDAO.FindByGithubID(ghUser.ID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperror.WrapInternal(err)
	}

	if user != nil {
		// 老 GitHub 用户：更新头像
		if ghUser.AvatarURL != "" {
			user.AvatarURL = &ghUser.AvatarURL
			_ = s.userDAO.Update(user)
		}
		return s.buildLoginResponse(user)
	}

	// 2. 按邮箱查找已有账号（链接密码注册用户与 GitHub 登录）
	if ghUser.Email != "" {
		user, err = s.userDAO.FindByEmail(ghUser.Email)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.WrapInternal(err)
		}
		if user != nil {
			// 找到同邮箱账号 → 链接 GitHub ID
			user.GithubID = &ghUser.ID
			if ghUser.AvatarURL != "" && user.AvatarURL == nil {
				user.AvatarURL = &ghUser.AvatarURL
			}
			// admin 邮箱确保 is_admin 不掉
			if ghUser.Email == s.adminEmail {
				user.IsAdmin = true
			}
			if err := s.userDAO.Save(user); err != nil {
				return nil, apperror.WrapInternal(err)
			}
			return s.buildLoginResponse(user)
		}
	}

	// 3. 真正的新用户：创建账号
	isAdmin := ghUser.Email == s.adminEmail
	var username string
	if isAdmin {
		username = s.adminUsername // admin 固定显示名
	} else {
		username = ghUser.Login
	}

	// 检查用户名是否冲突，最多重试 3 次追加随机后缀
	for retry := 0; retry < 3; retry++ {
		_, err = s.userDAO.FindByUsername(username)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			break // 用户名可用
		}
		if err != nil {
			return nil, apperror.WrapInternal(err)
		}
		// 用户名已存在，追加 4 位随机数后重试
		suffix, _ := generateRandomDigits(4)
		username = fmt.Sprintf("%s%s", ghUser.Login, suffix)
	}

	// 如果 GitHub 没有返回邮箱，用 github_id 构造占位邮箱
	email := ghUser.Email
	if email == "" {
		email = fmt.Sprintf("github_%d@placeholder.local", ghUser.ID)
	}

	var avatarURL *string
	if ghUser.AvatarURL != "" {
		avatarURL = &ghUser.AvatarURL
	}

	newUser := &model.User{
		Username:     username,
		Email:        email,
		PasswordHash: "", // GitHub 用户没有密码
		GithubID:     &ghUser.ID,
		AvatarURL:    avatarURL,
		IsAdmin:      isAdmin,
	}

	if err := s.userDAO.Create(newUser); err != nil {
		// UNIQUE 冲突处理：可能是邮箱被并发注册
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			// 再尝试按邮箱查找并链接
			if ghUser.Email != "" {
				user, findErr := s.userDAO.FindByEmail(ghUser.Email)
				if findErr == nil && user != nil {
					user.GithubID = &ghUser.ID
					if ghUser.AvatarURL != "" && user.AvatarURL == nil {
						user.AvatarURL = &ghUser.AvatarURL
					}
					_ = s.userDAO.Save(user)
					return s.buildLoginResponse(user)
				}
			}
			// 再尝试按用户名查找（用户名冲突）
			user, findErr := s.userDAO.FindByUsername(username)
			if findErr == nil && user != nil {
				return nil, apperror.BadRequest("该用户名已被注册，请更换 GitHub 用户名或联系管理员")
			}
			return nil, apperror.BadRequest("该邮箱已被注册，请使用邮箱登录后在设置中关联 GitHub")
		}
		return nil, apperror.WrapInternal(err)
	}

	return s.buildLoginResponse(newUser)
}

// loginOrCreateAdmin 将指定的 GitHub 用户映射为本站硬编码管理员。
//
// 硬编码的管理员信息（昵称、邮箱）直接写在代码常量中，只有密码走环境变量。
// 流程：
//  1. 按硬编码邮箱查找已有账号 → 链接 GitHub ID、确保 is_admin、更新头像
//  2. 没找到 → 用硬编码信息 + 环境变量密码创建管理员账号
func (s *Service) loginOrCreateAdmin(ghUser *oauth.GitHubUser) (*dto.LoginResponse, error) {
	// 1. 按硬编码邮箱查找管理员账号
	admin, err := s.userDAO.FindByEmail(adminEmailHard)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperror.WrapInternal(err)
	}

	if admin != nil {
		// 已有管理员账号：链接 GitHub ID，确保管理员权限不掉
		admin.GithubID = &ghUser.ID
		admin.IsAdmin = true
		if ghUser.AvatarURL != "" {
			admin.AvatarURL = &ghUser.AvatarURL
		}
		// 用户名也确保是硬编码的管理员名（覆盖任何历史遗留）
		admin.Username = adminName
		if err := s.userDAO.Save(admin); err != nil {
			return nil, apperror.WrapInternal(err)
		}
		return s.buildLoginResponse(admin)
	}

	// 2. 管理员账号还不存在：用硬编码信息创建
	hashedPassword := ""
	if s.adminPassword != "" {
		var err error
		hashedPassword, err = password.Hash(s.adminPassword)
		if err != nil {
			return nil, apperror.WrapInternal(err)
		}
	}

	newAdmin := &model.User{
		Username:     adminName,
		Email:        adminEmailHard,
		PasswordHash: hashedPassword,
		GithubID:     &ghUser.ID,
		IsAdmin:      true,
	}
	if ghUser.AvatarURL != "" {
		newAdmin.AvatarURL = &ghUser.AvatarURL
	}

	if err := s.userDAO.Create(newAdmin); err != nil {
		// 并发场景下可能被 Register 或别的 OAuth 流程抢先创建
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			admin, findErr := s.userDAO.FindByEmail(adminEmailHard)
			if findErr == nil && admin != nil {
				admin.GithubID = &ghUser.ID
				admin.IsAdmin = true
				if ghUser.AvatarURL != "" {
					admin.AvatarURL = &ghUser.AvatarURL
				}
				_ = s.userDAO.Save(admin)
				return s.buildLoginResponse(admin)
			}
		}
		return nil, apperror.WrapInternal(err)
	}

	return s.buildLoginResponse(newAdmin)
}

// buildLoginResponse 生成 JWT 令牌并构造响应
func (s *Service) buildLoginResponse(user *model.User) (*dto.LoginResponse, error) {
	token, err := auth.GenerateToken(s.jwtSecret, user.ID, user.Username, user.IsAdmin)
	if err != nil {
		return nil, apperror.WrapInternal(err)
	}

	return &dto.LoginResponse{Token: token, User: toUserResponse(user)}, nil
}

// CreateUser 管理员创建新用户 POST /api/v1/admin/users
func (s *Service) CreateUser(req dto.CreateUserRequest) (*dto.UserResponse, error) {
	// 密码强度校验
	if !isPasswordStrong(req.Password) {
		return nil, apperror.BadRequest("密码至少 8 位，且必须同时包含字母和数字")
	}

	// 检查邮箱是否已被注册
	_, err := s.userDAO.FindByEmail(req.Email)
	if err == nil {
		return nil, apperror.BadRequest("邮箱已被注册")
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperror.WrapInternal(err)
	}

	// 检查用户名是否已被占用
	_, err = s.userDAO.FindByUsername(req.Username)
	if err == nil {
		return nil, apperror.BadRequest("用户名已被注册")
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperror.WrapInternal(err)
	}

	// 密码加密
	hashedPassword, err := password.Hash(req.Password)
	if err != nil {
		return nil, apperror.WrapInternal(err)
	}

	newUser := &model.User{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: hashedPassword,
		IsAdmin:      req.IsAdmin,
	}

	if err := s.userDAO.Create(newUser); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, apperror.BadRequest("邮箱或用户名已被注册")
		}
		return nil, apperror.WrapInternal(err)
	}

	resp := toUserResponse(newUser)
	return &resp, nil
}

// ListAll 返回所有用户列表（仅管理员可用）
func (s *Service) ListAll() (*dto.UserListResponse, error) {
	users, err := s.userDAO.FindAll()
	if err != nil {
		return nil, apperror.WrapInternal(err)
	}

	result := make([]dto.UserResponse, 0, len(users))
	for _, u := range users {
		result = append(result, toUserResponse(&u))
	}

	return &dto.UserListResponse{Users: result}, nil
}

// toUserResponse 将 model.User 转换为 dto.UserResponse，确保所有字段一致
func toUserResponse(u *model.User) dto.UserResponse {
	return dto.UserResponse{
		ID:        u.ID,
		Username:  u.Username,
		Email:     u.Email,
		IsAdmin:   u.IsAdmin,
		AvatarURL: u.AvatarURL,
		Bio:       u.Bio,
		CreatedAt: u.CreatedAt,
	}
}

// isPasswordStrong 检查密码强度：至少 8 位 + 包含字母 + 包含数字
func isPasswordStrong(pw string) bool {
	if len(pw) < 8 {
		return false
	}
	hasLetter := false
	hasDigit := false
	for _, r := range pw {
		if unicode.IsLetter(r) {
			hasLetter = true
		}
		if unicode.IsDigit(r) {
			hasDigit = true
		}
	}
	return hasLetter && hasDigit
}

// UpdateUser 管理员编辑用户信息 PUT /api/v1/admin/users/:id
func (s *Service) UpdateUser(operatorID int64, targetID int64, req dto.UpdateUserRequest) (*dto.UserResponse, error) {
	target, err := s.userDAO.FindByID(targetID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound("用户不存在")
		}
		return nil, apperror.WrapInternal(err)
	}

	// 检查用户名是否被其他用户占用
	if req.Username != target.Username {
		existing, err := s.userDAO.FindByUsername(req.Username)
		if err == nil && existing.ID != targetID {
			return nil, apperror.BadRequest("用户名已被其他用户占用")
		}
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.WrapInternal(err)
		}
		target.Username = req.Username
	}

	// 检查邮箱是否被其他用户占用
	if req.Email != target.Email {
		existing, err := s.userDAO.FindByEmail(req.Email)
		if err == nil && existing.ID != targetID {
			return nil, apperror.BadRequest("邮箱已被其他用户占用")
		}
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.WrapInternal(err)
		}
		target.Email = req.Email
	}

	// 管理员不能取消自己的管理员权限
	if req.IsAdmin != nil {
		if targetID == operatorID && !*req.IsAdmin {
			return nil, apperror.BadRequest("不能取消自己的管理员权限")
		}
		target.IsAdmin = *req.IsAdmin
	}

	if err := s.userDAO.Update(target); err != nil {
		return nil, apperror.WrapInternal(err)
	}

	// GORM Updates 会跳过 bool 零值（false），所以 IsAdmin 的变更需要用 UpdateField 单独处理
	if req.IsAdmin != nil {
		if err := s.userDAO.UpdateField(targetID, "is_admin", *req.IsAdmin); err != nil {
			return nil, apperror.WrapInternal(err)
		}
	}

	resp := toUserResponse(target)
	return &resp, nil
}

// GetByID 查询单个用户资料（供「个人中心」读取本人最新信息）
func (s *Service) GetByID(userID int64) (*dto.UserResponse, error) {
	user, err := s.userDAO.FindByID(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound("用户不存在")
		}
		return nil, apperror.WrapInternal(err)
	}
	resp := toUserResponse(user)
	return &resp, nil
}

// UpdateProfile 普通用户编辑自己的基础信息（昵称、邮箱，可选修改密码）。
// 与管理员的 UpdateUser 区别：只能改自己、不涉及 is_admin、改密码需校验旧密码。
func (s *Service) UpdateProfile(userID int64, req dto.UpdateProfileRequest) (*dto.UserResponse, error) {
	user, err := s.userDAO.FindByID(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound("用户不存在")
		}
		return nil, apperror.WrapInternal(err)
	}

	// —— 用户名：变更时检查是否被他人占用 ——
	if req.Username != user.Username {
		existing, err := s.userDAO.FindByUsername(req.Username)
		if err == nil && existing.ID != userID {
			return nil, apperror.BadRequest("用户名已被其他用户占用")
		}
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.WrapInternal(err)
		}
		user.Username = req.Username
	}

	// —— 邮箱：变更时检查是否被他人占用 ——
	if req.Email != user.Email {
		existing, err := s.userDAO.FindByEmail(req.Email)
		if err == nil && existing.ID != userID {
			return nil, apperror.BadRequest("邮箱已被其他用户占用")
		}
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.WrapInternal(err)
		}
		user.Email = req.Email
	}

	// —— 密码：仅当填写了新密码时才修改 ——
	if req.NewPassword != "" {
		if !isPasswordStrong(req.NewPassword) {
			return nil, apperror.BadRequest("新密码至少 8 位，且必须同时包含字母和数字")
		}
		// 已设置密码的账号必须校验旧密码；GitHub 等无密码账号可直接设置初始密码
		if user.PasswordHash != "" {
			if !password.Check(req.OldPassword, user.PasswordHash) {
				return nil, apperror.BadRequest("当前密码不正确")
			}
		}
		hashed, err := password.Hash(req.NewPassword)
		if err != nil {
			return nil, apperror.WrapInternal(err)
		}
		user.PasswordHash = hashed
	}

	if err := s.userDAO.Update(user); err != nil {
		return nil, apperror.WrapInternal(err)
	}

	resp := toUserResponse(user)
	return &resp, nil
}

// DeleteUser 管理员删除用户 DELETE /api/v1/admin/users/:id
func (s *Service) DeleteUser(operatorID int64, targetID int64) error {
	// 不能删除自己
	if operatorID == targetID {
		return apperror.BadRequest("不能删除自己的账号")
	}

	_, err := s.userDAO.FindByID(targetID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperror.NotFound("用户不存在")
		}
		return apperror.WrapInternal(err)
	}

	return s.userDAO.Delete(targetID)
}

// generateRandomDigits 生成指定长度的随机数字字符串
func generateRandomDigits(n int) (string, error) {
	result := make([]byte, n)
	for i := 0; i < n; i++ {
		num, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", err
		}
		result[i] = '0' + byte(num.Int64())
	}
	return string(result), nil
}