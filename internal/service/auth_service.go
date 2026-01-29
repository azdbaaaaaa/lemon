package service

import (
	"context"
	"errors"
	"time"

	"github.com/rs/zerolog/log"

	"lemon/internal/model/auth"
	"lemon/internal/pkg/id"
	"lemon/internal/pkg/jwt"
	"lemon/internal/pkg/password"
	authRepo "lemon/internal/repository/auth"
)

var (
	ErrUserNotFound      = errors.New("用户不存在")
	ErrUserAlreadyExists = errors.New("用户已存在")
	ErrInvalidPassword   = errors.New("密码错误")
	ErrUserInactive      = errors.New("用户未激活，请联系管理员")
	ErrUserBanned        = errors.New("用户已被禁用")
	ErrInvalidToken      = errors.New("Token无效")
	ErrExpiredToken      = errors.New("Token已过期")
)

// AuthService 认证服务
type AuthService struct {
	userRepo         *authRepo.UserRepo
	refreshTokenRepo *authRepo.RefreshTokenRepo
	jwt              *jwt.JWT
	refreshExpiry    time.Duration // Refresh Token过期时间
}

// NewAuthService 创建认证服务
func NewAuthService(
	userRepo *authRepo.UserRepo,
	refreshTokenRepo *authRepo.RefreshTokenRepo,
	jwtSecret string,
	accessTokenExpiry time.Duration,
	refreshTokenExpiry time.Duration,
) *AuthService {
	return &AuthService{
		userRepo:         userRepo,
		refreshTokenRepo: refreshTokenRepo,
		jwt:              jwt.NewJWT(jwtSecret, accessTokenExpiry),
		refreshExpiry:    refreshTokenExpiry,
	}
}

// RegisterResult 注册结果
type RegisterResult struct {
	UserID   string
	Username string
	Status   string
}

// Register 用户注册
// 使用基本类型参数，不依赖Handler层的Request类型
func (s *AuthService) Register(ctx context.Context, username, email, pwd, nickname string) (*RegisterResult, error) {
	// 检查用户名是否已存在
	existing, _ := s.userRepo.FindByUsername(ctx, username)
	if existing != nil {
		return nil, ErrUserAlreadyExists
	}

	// 检查邮箱是否已存在
	existing, _ = s.userRepo.FindByEmail(ctx, email)
	if existing != nil {
		return nil, errors.New("邮箱已被注册")
	}

	// 加密密码
	hashedPassword, err := password.Hash(pwd)
	if err != nil {
		log.Error().Err(err).Msg("failed to hash password")
		return nil, errors.New("密码加密失败")
	}

	// 创建用户
	user := &auth.User{
		ID:       id.New(), // 生成UUID
		Username: username,
		Email:    email,
		Password: hashedPassword,
		Role:     auth.RoleEditor,         // 新注册用户默认为editor
		Status:   auth.UserStatusInactive, // 新注册用户需要管理员审核
	}

	if nickname != "" {
		user.Profile = &auth.UserProfile{
			Nickname: nickname,
		}
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		log.Error().Err(err).Msg("failed to create user")
		return nil, errors.New("创建用户失败")
	}

	return &RegisterResult{
		UserID:   user.ID,
		Username: user.Username,
		Status:   string(user.Status),
	}, nil
}

// LoginResult 登录结果
type LoginResult struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int
	TokenType    string
	User         *auth.User
}

// Login 用户登录
// 使用基本类型参数，不依赖Handler层的Request类型
func (s *AuthService) Login(ctx context.Context, username, pwd string) (*LoginResult, error) {
	// 查找用户
	user, err := s.userRepo.FindByUsername(ctx, username)
	if err != nil {
		return nil, ErrUserNotFound
	}

	// 验证密码
	if !password.Verify(pwd, user.Password) {
		return nil, ErrInvalidPassword
	}

	// 检查用户状态
	if user.Status == auth.UserStatusInactive {
		return nil, ErrUserInactive
	}
	if user.Status == auth.UserStatusBanned {
		return nil, ErrUserBanned
	}

	// 生成Access Token
	accessToken, err := s.jwt.GenerateToken(user.ID, user.Username, string(user.Role))
	if err != nil {
		log.Error().Err(err).Msg("failed to generate access token")
		return nil, errors.New("生成Token失败")
	}

	// 生成Refresh Token
	refreshTokenValue := jwt.GenerateRefreshToken()
	refreshToken := &auth.RefreshToken{
		ID:        id.New(), // 生成UUID
		UserID:    user.ID,
		Token:     refreshTokenValue,
		ExpiresAt: time.Now().Add(s.refreshExpiry),
	}

	if err := s.refreshTokenRepo.Create(ctx, refreshToken); err != nil {
		log.Error().Err(err).Msg("failed to create refresh token")
		return nil, errors.New("创建Refresh Token失败")
	}

	// 更新最后登录时间
	if err := s.userRepo.UpdateLastLoginAt(ctx, user.ID); err != nil {
		log.Warn().Err(err).Msg("failed to update last login time")
		// 不影响登录流程，只记录警告
	}

	// 获取过期时间（秒）
	expiresIn := int(s.jwt.GetExpiration().Seconds())

	return &LoginResult{
		AccessToken:  accessToken,
		RefreshToken: refreshTokenValue,
		ExpiresIn:    expiresIn,
		TokenType:    "Bearer",
		User:         user,
	}, nil
}

// RefreshTokenResult 刷新Token结果
type RefreshTokenResult struct {
	AccessToken string
	ExpiresIn   int
	TokenType   string
}

// RefreshToken 刷新Access Token
func (s *AuthService) RefreshToken(ctx context.Context, refreshTokenValue string) (*RefreshTokenResult, error) {
	// 查找Refresh Token
	refreshToken, err := s.refreshTokenRepo.FindByToken(ctx, refreshTokenValue)
	if err != nil {
		return nil, ErrInvalidToken
	}

	// 检查Token是否过期
	if refreshToken.IsExpired() {
		// 删除过期的Token
		_ = s.refreshTokenRepo.DeleteByToken(ctx, refreshTokenValue)
		return nil, ErrExpiredToken
	}

	// 查找用户
	user, err := s.userRepo.FindByID(ctx, refreshToken.UserID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	// 检查用户状态
	if user.Status == auth.UserStatusBanned {
		return nil, ErrUserBanned
	}

	// 生成新的Access Token
	accessToken, err := s.jwt.GenerateToken(user.ID, user.Username, string(user.Role))
	if err != nil {
		log.Error().Err(err).Msg("failed to generate access token")
		return nil, errors.New("生成Token失败")
	}

	// 获取Access Token过期时间（秒）
	expiresIn := int(s.jwt.GetExpiration().Seconds())

	return &RefreshTokenResult{
		AccessToken: accessToken,
		ExpiresIn:   expiresIn,
		TokenType:   "Bearer",
	}, nil
}

// Logout 退出登录
func (s *AuthService) Logout(ctx context.Context, refreshTokenValue string) error {
	// 删除Refresh Token
	return s.refreshTokenRepo.DeleteByToken(ctx, refreshTokenValue)
}

// GetUserByID 根据ID获取用户信息
func (s *AuthService) GetUserByID(ctx context.Context, userID string) (*auth.User, error) {
	return s.userRepo.FindByID(ctx, userID)
}

// ValidateToken 验证Access Token并返回用户信息
func (s *AuthService) ValidateToken(tokenString string) (*auth.User, error) {
	claims, err := s.jwt.ValidateToken(tokenString)
	if err != nil {
		return nil, err
	}

	// 查找用户
	user, err := s.userRepo.FindByID(context.Background(), claims.UserID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	// 检查用户状态
	if user.Status == auth.UserStatusBanned {
		return nil, ErrUserBanned
	}

	return user, nil
}
