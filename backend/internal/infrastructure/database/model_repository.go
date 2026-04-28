package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"eval-dominator/backend/internal/domain"
)

type ModelRepository struct {
	db *sql.DB
}

func NewModelRepository(db *sql.DB) *ModelRepository {
	return &ModelRepository{db: db}
}

func (r *ModelRepository) Create(ctx context.Context, model domain.Model) (*domain.Model, error) {
	res, err := r.db.ExecContext(
		ctx,
		`INSERT INTO models (user_id, provider, model_name, display_name, version, api_key, base_url, masked_key)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		model.UserID,
		model.Provider,
		model.ModelName,
		model.DisplayName,
		model.Version,
		model.APIKey,
		model.BaseURL,
		model.MaskedKey,
	)
	if err != nil {
		return nil, fmt.Errorf("创建模型预设失败: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("获取插入 ID 失败: %w", err)
	}
	return r.GetByID(ctx, id)
}

func (r *ModelRepository) Update(ctx context.Context, id int64, userID int64, fields domain.Model, updateAPIKey bool) (*domain.Model, error) {
	now := time.Now()
	if updateAPIKey {
		_, err := r.db.ExecContext(
			ctx,
			`UPDATE models SET provider = ?, model_name = ?, display_name = ?, version = ?, base_url = ?, api_key = ?, masked_key = ?, updated_at = ?
			WHERE id = ? AND user_id = ?`,
			fields.Provider,
			fields.ModelName,
			fields.DisplayName,
			fields.Version,
			fields.BaseURL,
			fields.APIKey,
			fields.MaskedKey,
			now,
			id,
			userID,
		)
		if err != nil {
			return nil, fmt.Errorf("更新模型预设失败: %w", err)
		}
	} else {
		_, err := r.db.ExecContext(
			ctx,
			`UPDATE models SET provider = ?, model_name = ?, display_name = ?, version = ?, base_url = ?, updated_at = ?
			WHERE id = ? AND user_id = ?`,
			fields.Provider,
			fields.ModelName,
			fields.DisplayName,
			fields.Version,
			fields.BaseURL,
			now,
			id,
			userID,
		)
		if err != nil {
			return nil, fmt.Errorf("更新模型预设失败: %w", err)
		}
	}
	return r.GetByID(ctx, id)
}

func (r *ModelRepository) Delete(ctx context.Context, id int64, userID int64) error {
	res, err := r.db.ExecContext(
		ctx,
		`DELETE FROM models WHERE id = ? AND user_id = ?`,
		id,
		userID,
	)
	if err != nil {
		return fmt.Errorf("删除模型预设失败: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("获取删除结果失败: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("模型预设不存在")
	}
	return nil
}

func (r *ModelRepository) GetByID(ctx context.Context, id int64) (*domain.Model, error) {
	row := r.db.QueryRowContext(
		ctx,
		`SELECT id, user_id, provider, model_name, display_name, version, api_key, base_url, masked_key, created_at, updated_at
		FROM models WHERE id = ?`,
		id,
	)
	var m domain.Model
	if err := row.Scan(
		&m.ID,
		&m.UserID,
		&m.Provider,
		&m.ModelName,
		&m.DisplayName,
		&m.Version,
		&m.APIKey,
		&m.BaseURL,
		&m.MaskedKey,
		&m.CreatedAt,
		&m.UpdatedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("模型预设不存在")
		}
		return nil, fmt.Errorf("查询模型预设失败: %w", err)
	}
	return &m, nil
}

func (r *ModelRepository) ListByUser(ctx context.Context, userID int64) ([]domain.Model, error) {
	rows, err := r.db.QueryContext(
		ctx,
		`SELECT id, user_id, provider, model_name, display_name, version, api_key, base_url, masked_key, created_at, updated_at
		FROM models WHERE user_id = ? ORDER BY created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("查询模型预设列表失败: %w", err)
	}
	defer rows.Close()

	items := make([]domain.Model, 0)
	for rows.Next() {
		var m domain.Model
		if err := rows.Scan(
			&m.ID,
			&m.UserID,
			&m.Provider,
			&m.ModelName,
			&m.DisplayName,
			&m.Version,
			&m.APIKey,
			&m.BaseURL,
			&m.MaskedKey,
			&m.CreatedAt,
			&m.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("扫描模型预设失败: %w", err)
		}
		items = append(items, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历模型预设列表失败: %w", err)
	}
	return items, nil
}

// Save 写入一条新模型预设（提交评测时勾选「保存模型」走这里）。
func (r *ModelRepository) Save(ctx context.Context, model domain.Model) error {
	_, err := r.db.ExecContext(
		ctx,
		`INSERT INTO models (user_id, provider, model_name, display_name, version, api_key, base_url, masked_key)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		model.UserID,
		model.Provider,
		model.ModelName,
		model.DisplayName,
		model.Version,
		model.APIKey,
		model.BaseURL,
		model.MaskedKey,
	)
	if err != nil {
		return fmt.Errorf("保存模型预设失败: %w", err)
	}
	return nil
}
