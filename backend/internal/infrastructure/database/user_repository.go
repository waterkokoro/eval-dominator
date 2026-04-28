package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"eval-dominator/backend/internal/domain"
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

// ErrUserNotFound 由 GetByUsername 返回，便于上层区分"不存在"与"读取失败"。
var ErrUserNotFound = errors.New("user not found")

func (r *UserRepository) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	row := r.db.QueryRowContext(
		ctx,
		`SELECT id, username, password_hash, created_at, updated_at
		FROM users WHERE username = ?`,
		username,
	)
	var u domain.User
	if err := row.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.CreatedAt, &u.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}
	return &u, nil
}

func (r *UserRepository) Create(ctx context.Context, user *domain.User) error {
	res, err := r.db.ExecContext(
		ctx,
		`INSERT INTO users (username, password_hash) VALUES (?, ?)`,
		user.Username, user.PasswordHash,
	)
	if err != nil {
		return fmt.Errorf("创建用户失败: %w", err)
	}
	id, err := res.LastInsertId()
	if err == nil {
		user.ID = id
	}
	return nil
}

func (r *UserRepository) UpdatePassword(ctx context.Context, userID int64, passwordHash string) error {
	_, err := r.db.ExecContext(
		ctx,
		`UPDATE users SET password_hash = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		passwordHash, userID,
	)
	if err != nil {
		return fmt.Errorf("更新用户密码失败: %w", err)
	}
	return nil
}
