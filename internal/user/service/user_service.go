package service

import (
	"context"
	"crypto_trading_system/internal/user/model"
	"crypto_trading_system/internal/user/repository"
	"crypto_trading_system/pkg/auth"
	"crypto_trading_system/pkg/cache"
	"crypto_trading_system/pkg/logger"
	"fmt"
	"time"
)

type UserService interface {
	Register(ctx context.Context, req *RegisterRequest) (*RegisterResponse, error)
	Login(ctx context.Context, req *LoginRequest) (*LoginResponse, error)
	GetProfile(ctx context.Context, req *GetProfileRequest) (*GetProfileResponse, error)
	UpdateProfile(ctx context.Context, req *UpdateProfileRequest) error
	Logout(ctx context.Context, req *LogoutRequest) error
	RefreshToken(ctx context.Context, req *RefreshTokenRequest) (*RefreshTokenResponse, error)
}

type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Password string `json:"password" binding:"required,email"`
	Email    string `json:"email" binding:"required,min=8"`
}
type RegisterResponse struct {
	User  *model.User `json:"user"`
	Token string      `json:"token"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	User  *model.User `json:"user"`
	Token string      `json:"token"`
}

type GetProfileRequest struct {
	UserID uint `json:"user_id"`
}

type GetProfileResponse struct {
	User *model.User `json:"user"`
}

type UpdateProfileRequest struct {
	UserID    uint   `json:"user_id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Phone     string `json:"phone"`
	Country   string `json:"country"`
	City      string `json:"city"`
}

type LogoutRequest struct {
	UserID uint   `json:"user_id"`
	Token  string `json:"token"`
}

type RefreshTokenRequest struct {
	Token string `json:"token"`
}

type RefreshTokenResponse struct {
	Token string `json:"token"`
}

type userService struct {
	userRepo   repository.UserRepository
	jwtService auth.JWTService
	cache      *cache.RedisClient
	logger     *logger.Logger
}

func NewUserService(
	userRepo repository.UserRepository,
	jwtService auth.JWTService,
	cache *cache.RedisClient,
	logger *logger.Logger,
) UserService {
	return &userService{
		userRepo:   userRepo,
		jwtService: jwtService,
		cache:      cache,
		logger:     logger,
	}
}

func (u *userService) Register(ctx context.Context, req *RegisterRequest) (*RegisterResponse, error) {
	// 1. 检查用户是否存在
	if _, err := u.userRepo.GetByUsername(ctx, req.Username); err == nil {
		return nil, repository.ErrUserExists
	}
	// 2. 检查邮箱是否存在
	if _, err := u.userRepo.GetByEmail(ctx, req.Email); err == nil {
		return nil, repository.ErrUserExists
	}
	//3. 创建用户
	user := &model.User{
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password,
	}

	if err := u.userRepo.Create(ctx, user); err != nil {
		u.logger.WithError(err).Error("failed to create user")
		return nil, err
	}

	//4. 生成token
	token, err := u.jwtService.GenerateToken(user.ID, user.Username)
	if err != nil {
		u.logger.WithError(err).Error("failed to generate token")
		return nil, err
	}

	// 5. 缓存用户信息
	u.cacheUser(ctx, user)
	u.logger.WithField("user_id", user.ID).Info("user registered successfully")

	return &RegisterResponse{
		User:  user,
		Token: token,
	}, nil
}

func (u *userService) Login(ctx context.Context, req *LoginRequest) (*LoginResponse, error) {
	// 获取用户
	user, err := u.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		if err == repository.ErrUserNotFound {
			return nil, repository.ErrInvalidCredentials
		}
		return nil, err
	}

	// 检查密码
	if !user.CheckPassword(req.Password) {
		u.logger.WithField("email", req.Email).Warn("Invalid login attempt")
		return nil, repository.ErrInvalidCredentials
	}

	// 检查用户状态
	if !user.IsActive() {
		return nil, fmt.Errorf("user account is not active")
	}

	token, err := u.jwtService.GenerateToken(user.ID, user.Username)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}
	u.cacheUser(ctx, user)

	u.logger.WithField("user_id", user.ID).Info("User logged in successfully")

	return &LoginResponse{
		User:  user,
		Token: token,
	}, nil
}

func (u *userService) GetProfile(ctx context.Context, req *GetProfileRequest) (*GetProfileResponse, error) {

	userID := req.UserID
	// 先从缓存查找
	cacheKey := fmt.Sprintf("user:%d", userID)
	if exists, _ := u.cache.Exists(ctx, cacheKey); exists {
		// 这里可以实现缓存获取逻辑
	}

	// 从数据库获取
	user, err := u.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// 更新缓存
	u.cacheUser(ctx, user)

	return &GetProfileResponse{
		User: user,
	}, nil
}

func (u *userService) UpdateProfile(ctx context.Context, req *UpdateProfileRequest) error {
	userID := req.UserID
	user, err := u.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	// 更新用户资料
	if user.Profile == nil {
		user.Profile = &model.UserProfile{UserID: userID}
	}

	user.Profile.FirstName = req.FirstName
	user.Profile.LastName = req.LastName
	user.Profile.Phone = req.Phone
	user.Profile.Country = req.Country
	user.Profile.City = req.City

	if err := u.userRepo.Update(ctx, user); err != nil {
		return err
	}

	// 清除缓存
	u.clearUserCache(ctx, userID)

	return nil
}

func (u *userService) Logout(ctx context.Context, req *LogoutRequest) error {
	userID := req.UserID
	token := req.Token
	// 将token加入黑名单
	blacklistKey := fmt.Sprintf("blacklist:%s", token)
	if err := u.cache.Set(ctx, blacklistKey, "1", 24*time.Hour); err != nil {
		u.logger.WithError(err).Error("Failed to blacklist token")
	}

	// 清除用户缓存
	u.clearUserCache(ctx, userID)

	u.logger.WithField("user_id", userID).Info("User logged out")
	return nil
}

func (u *userService) RefreshToken(ctx context.Context, req *RefreshTokenRequest) (*RefreshTokenResponse, error) {
	token := req.Token
	newToken, err := u.jwtService.RefreshToken(token)
	if err != nil {
		return nil, err
	}

	return &RefreshTokenResponse{Token: newToken}, nil
}

func (u *userService) cacheUser(ctx context.Context, user *model.User) {
	cacheKey := fmt.Sprintf("user:%d", user.ID)
	if err := u.cache.Set(ctx, cacheKey, user, time.Hour); err != nil {
		u.logger.WithError(err).Error("Failed to cache user")
	}
}

func (u *userService) clearUserCache(ctx context.Context, userID uint) {
	cacheKey := fmt.Sprintf("user:%d", userID)
	if err := u.cache.Del(ctx, cacheKey); err != nil {
		u.logger.WithError(err).Error("Failed to clear user cache")
	}
}
