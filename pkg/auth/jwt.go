package auth

import (
	"crypto_trading_system/pkg/config"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type JWTService interface {
	GenerateToken(userID uint, username string) (string, error)
	ValidateToken(tokenString string) (*Claims, error)
	RefreshToken(tokenString string) (string, error)
}

type Claims struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

type jwtService struct {
	secretKey []byte
	issuer    string
	expiresIn time.Duration
	refreshIn time.Duration
}

func NewJWTService(cfg *config.Config) JWTService {
	return &jwtService{
		secretKey: []byte(cfg.JWT.Secret),
		issuer:    cfg.JWT.Issuer,
		expiresIn: time.Duration(cfg.JWT.ExpiresIn) * time.Minute,
		refreshIn: time.Duration(cfg.JWT.RefreshIn) * time.Minute,
	}
}

func (j *jwtService) GenerateToken(userID uint, username string) (string, error) {
	now := time.Now()
	claims := &Claims{
		UserID:   userID,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    j.issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(j.expiresIn)),
			Subject:   username,
			NotBefore: jwt.NewNumericDate(now),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString(j.secretKey)
}

func (j *jwtService) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// 优先匹配 token 签名算法
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("token signed method is not HS256")
		}
		return j.secretKey, nil
	})

	if err != nil {
		return nil, err
	}

	if cliams, ok := token.Claims.(*Claims); ok && token.Valid {
		return cliams, nil
	}

	return nil, errors.New("invalid token")
}

func (j *jwtService) RefreshToken(tokenString string) (string, error) {
	claims, err := j.ValidateToken(tokenString)
	if err != nil {
		return "", err
	}

	if time.Until(claims.ExpiresAt.Time) > j.refreshIn {
		return "", errors.New("token is not expired")
	}

	return j.GenerateToken(claims.UserID, claims.Username)
}
