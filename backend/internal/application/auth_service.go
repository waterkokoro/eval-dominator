package application

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"eval-dominator/backend/internal/config"
	"eval-dominator/backend/internal/domain"
	"eval-dominator/backend/internal/infrastructure/database"
)

// UserRepository 抽象，便于替换实现 / 测试。
type UserRepository interface {
	GetByUsername(ctx context.Context, username string) (*domain.User, error)
	Create(ctx context.Context, user *domain.User) error
	UpdatePassword(ctx context.Context, userID int64, passwordHash string) error
}

type AuthService struct {
	jwtConfig  config.JWTConfig
	authConfig config.AuthConfig
	userRepo   UserRepository
}

func NewAuthService(jwtConfig config.JWTConfig, authConfig config.AuthConfig, userRepo UserRepository) *AuthService {
	return &AuthService{jwtConfig: jwtConfig, authConfig: authConfig, userRepo: userRepo}
}

// ErrInvalidCredentials 用户名 / 密码错误时返回；不区分"用户不存在"与"密码错误"以避免账号枚举。
var ErrInvalidCredentials = errors.New("用户名或密码错误")

func (s *AuthService) Login(ctx context.Context, username string, password string) (string, time.Time, error) {
	username = strings.TrimSpace(username)
	if username == "" || password == "" {
		return "", time.Time{}, fmt.Errorf("用户名和密码不能为空")
	}

	user, err := s.userRepo.GetByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, database.ErrUserNotFound) {
			return "", time.Time{}, ErrInvalidCredentials
		}
		return "", time.Time{}, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", time.Time{}, ErrInvalidCredentials
	}

	expiresAt := time.Now().Add(s.jwtConfig.ExpireDuration())
	claims := jwt.MapClaims{
		"iss":      s.jwtConfig.Issuer,
		"sub":      fmt.Sprintf("%d", user.ID),
		"username": user.Username,
		"exp":      expiresAt.Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(s.jwtConfig.Secret))
	if err != nil {
		return "", time.Time{}, fmt.Errorf("生成 JWT 失败: %w", err)
	}
	return tokenString, expiresAt, nil
}

// EnsureDefaultUser 确保至少存在一个可登录账号。如果默认账号不存在则按配置创建；
// 若配置仍是默认占位密码，启动时打印警告，提醒上线前 ChangePassword。
func (s *AuthService) EnsureDefaultUser(ctx context.Context) error {
	username := strings.TrimSpace(s.authConfig.DefaultAdminUsername)
	password := s.authConfig.DefaultAdminPassword
	if username == "" || password == "" {
		return fmt.Errorf("auth.default_admin_username / default_admin_password 不能为空")
	}

	if _, err := s.userRepo.GetByUsername(ctx, username); err == nil {
		// 已存在：不主动覆盖密码，避免管理员手动改过又被启动重置。
		if password == config.DefaultAdminPasswordPlaceholder {
			log.Printf("[auth] 注意：当前使用占位密码 %q 登录。强烈建议通过 POST /auth/change-password 修改并把 backend/config/config.yaml 同步更新。", config.DefaultAdminPasswordPlaceholder)
		}
		return nil
	} else if !errors.Is(err, database.ErrUserNotFound) {
		return err
	}

	hash, err := hashPassword(password)
	if err != nil {
		return err
	}
	user := &domain.User{Username: username, PasswordHash: hash}
	if err := s.userRepo.Create(ctx, user); err != nil {
		return err
	}
	if password == config.DefaultAdminPasswordPlaceholder {
		log.Printf("[auth] 已 seed 默认账号 username=%q password=%q ── 上线前请改！", username, password)
	} else {
		log.Printf("[auth] 已 seed 账号 username=%q（密码来自 backend/config/config.yaml）", username)
	}
	return nil
}

// ChangePassword 校验旧密码后更新。userID/username 由 Auth 中间件从 JWT 注入。
func (s *AuthService) ChangePassword(ctx context.Context, username string, oldPassword string, newPassword string) error {
	if newPassword == "" || len(newPassword) < 6 {
		return fmt.Errorf("新密码至少 6 位")
	}
	if oldPassword == newPassword {
		return fmt.Errorf("新密码不能与旧密码相同")
	}

	user, err := s.userRepo.GetByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, database.ErrUserNotFound) {
			return fmt.Errorf("用户不存在")
		}
		return err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(oldPassword)); err != nil {
		return fmt.Errorf("旧密码错误")
	}
	hash, err := hashPassword(newPassword)
	if err != nil {
		return err
	}
	return s.userRepo.UpdatePassword(ctx, user.ID, hash)
}

type TokenClaims struct {
	UserID   int64
	Username string
}

func (s *AuthService) ParseTokenClaims(tokenString string) (*TokenClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("不支持的 JWT 签名方法")
		}
		return []byte(s.jwtConfig.Secret), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("无效 token")
	}
	sub, err := claims.GetSubject()
	if err != nil {
		return nil, err
	}
	var userID int64
	if _, err := fmt.Sscan(sub, &userID); err != nil {
		return nil, fmt.Errorf("解析用户 ID 失败: %w", err)
	}
	username, _ := claims["username"].(string)
	return &TokenClaims{UserID: userID, Username: username}, nil
}

func (s *AuthService) ParseToken(tokenString string) (int64, error) {
	claims, err := s.ParseTokenClaims(tokenString)
	if err != nil {
		return 0, err
	}
	return claims.UserID, nil
}

// WarnIfDefaultJWTSecret 启动时调用：JWT secret 仍是 example 占位符则打 WARNING，
// 而非 fail-fast，避免新用户首次启动卡住。README 已强调要改。
func (s *AuthService) WarnIfDefaultJWTSecret() {
	if s.jwtConfig.Secret == config.DefaultJWTSecretPlaceholder {
		log.Printf("[auth] 注意：JWT secret 仍是默认占位符 %q，公网部署前必须修改 backend/config/config.yaml::jwt.secret。", config.DefaultJWTSecretPlaceholder)
	}
}

func hashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("生成密码哈希失败: %w", err)
	}
	return string(hash), nil
}
