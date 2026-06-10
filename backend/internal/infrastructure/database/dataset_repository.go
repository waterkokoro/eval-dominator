package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"eval-dominator/backend/internal/domain"
)

type DatasetRepository struct {
	db *sql.DB
}

func NewDatasetRepository(db *sql.DB) *DatasetRepository {
	return &DatasetRepository{db: db}
}

func (r *DatasetRepository) Create(ctx context.Context, dataset domain.Dataset) (*domain.Dataset, error) {
	res, err := r.db.ExecContext(
		ctx,
		`INSERT INTO datasets (code, display_name, description, type, source, sample_count, enabled, inference_mode, config_path, extra_json, hf_repo, hf_subset, local_path, file_format)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		dataset.Code,
		dataset.DisplayName,
		dataset.Description,
		dataset.Type,
		string(dataset.Source),
		dataset.SampleCount,
		boolToInt(dataset.Enabled),
		dataset.InferenceMode,
		dataset.ConfigPath,
		nullableJSON(dataset.ExtraJSON),
		dataset.HuggingFaceRepo,
		dataset.HuggingFaceSubset,
		dataset.LocalPath,
		dataset.FileFormat,
	)
	if err != nil {
		return nil, fmt.Errorf("创建 Dataset 失败: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("获取插入 ID 失败: %w", err)
	}
	return r.GetByID(ctx, id)
}

func (r *DatasetRepository) Update(ctx context.Context, id int64, dataset domain.Dataset) (*domain.Dataset, error) {
	now := time.Now()
	_, err := r.db.ExecContext(
		ctx,
		`UPDATE datasets
		SET display_name = ?, description = ?, type = ?, sample_count = ?, enabled = ?, inference_mode = ?, config_path = ?, extra_json = ?, hf_repo = ?, hf_subset = ?, local_path = ?, file_format = ?, updated_at = ?
		WHERE id = ?`,
		dataset.DisplayName,
		dataset.Description,
		dataset.Type,
		dataset.SampleCount,
		boolToInt(dataset.Enabled),
		dataset.InferenceMode,
		dataset.ConfigPath,
		nullableJSON(dataset.ExtraJSON),
		dataset.HuggingFaceRepo,
		dataset.HuggingFaceSubset,
		dataset.LocalPath,
		dataset.FileFormat,
		now,
		id,
	)
	if err != nil {
		return nil, fmt.Errorf("更新 Dataset 失败: %w", err)
	}
	return r.GetByID(ctx, id)
}

func (r *DatasetRepository) UpdateEnabled(ctx context.Context, id int64, enabled bool) error {
	_, err := r.db.ExecContext(
		ctx,
		`UPDATE datasets SET enabled = ?, updated_at = ? WHERE id = ?`,
		boolToInt(enabled),
		time.Now(),
		id,
	)
	if err != nil {
		return fmt.Errorf("更新 Dataset 启用状态失败: %w", err)
	}
	return nil
}

func (r *DatasetRepository) Delete(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM datasets WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("删除 Dataset 失败: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("获取删除结果失败: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("Dataset 不存在")
	}
	return nil
}

func (r *DatasetRepository) GetByID(ctx context.Context, id int64) (*domain.Dataset, error) {
	row := r.db.QueryRowContext(
		ctx,
		`SELECT id, code, display_name, description, type, source, sample_count, enabled, inference_mode, config_path, extra_json, hf_repo, hf_subset, local_path, file_format, created_at, updated_at
		FROM datasets WHERE id = ?`,
		id,
	)
	return scanDataset(row)
}

func (r *DatasetRepository) GetByCode(ctx context.Context, code string) (*domain.Dataset, error) {
	row := r.db.QueryRowContext(
		ctx,
		`SELECT id, code, display_name, description, type, source, sample_count, enabled, inference_mode, config_path, extra_json, hf_repo, hf_subset, local_path, file_format, created_at, updated_at
		FROM datasets WHERE code = ?`,
		code,
	)
	return scanDataset(row)
}

func (r *DatasetRepository) List(ctx context.Context, includeDisabled bool) ([]domain.Dataset, error) {
	query := `SELECT id, code, display_name, description, type, source, sample_count, enabled, inference_mode, config_path, extra_json, hf_repo, hf_subset, local_path, file_format, created_at, updated_at
		FROM datasets`
	if !includeDisabled {
		query += " WHERE enabled = 1"
	}
	query += " ORDER BY source ASC, code ASC"

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("查询 Dataset 列表失败: %w", err)
	}
	defer rows.Close()

	items := make([]domain.Dataset, 0)
	for rows.Next() {
		var ds domain.Dataset
		if err := scanRowsInto(rows, &ds); err != nil {
			return nil, err
		}
		items = append(items, ds)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历 Dataset 列表失败: %w", err)
	}
	return items, nil
}

// UpsertBuiltin 仅在 code 不存在时插入；已存在时只刷新 config_path 和 inference_mode（处理 venv 路径变化或新加字段 backfill）。
func (r *DatasetRepository) UpsertBuiltin(ctx context.Context, dataset domain.Dataset) error {
	_, err := r.db.ExecContext(
		ctx,
		`INSERT INTO datasets (code, display_name, description, type, source, sample_count, enabled, inference_mode, config_path, extra_json)
		VALUES (?, ?, ?, ?, 'builtin', ?, 1, ?, ?, ?)
		ON CONFLICT(code) DO UPDATE SET
			config_path = excluded.config_path,
			inference_mode = excluded.inference_mode,
			updated_at = CURRENT_TIMESTAMP`,
		dataset.Code,
		dataset.DisplayName,
		dataset.Description,
		dataset.Type,
		dataset.SampleCount,
		dataset.InferenceMode,
		dataset.ConfigPath,
		nullableJSON(dataset.ExtraJSON),
	)
	if err != nil {
		return fmt.Errorf("Upsert builtin Dataset 失败: %w", err)
	}
	return nil
}

// UpsertHuggingFace 插入或更新 HuggingFace 数据集。已存在时更新 local_path、sample_count 等。
func (r *DatasetRepository) UpsertHuggingFace(ctx context.Context, dataset domain.Dataset) error {
	_, err := r.db.ExecContext(
		ctx,
		`INSERT INTO datasets (code, display_name, description, type, source, sample_count, enabled, inference_mode, config_path, extra_json, hf_repo, hf_subset, local_path, file_format)
		VALUES (?, ?, ?, ?, 'huggingface', ?, 1, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(code) DO UPDATE SET
			display_name = excluded.display_name,
			description = excluded.description,
			sample_count = excluded.sample_count,
			local_path = excluded.local_path,
			file_format = excluded.file_format,
			hf_repo = excluded.hf_repo,
			hf_subset = excluded.hf_subset,
			config_path = excluded.config_path,
			updated_at = CURRENT_TIMESTAMP`,
		dataset.Code,
		dataset.DisplayName,
		dataset.Description,
		dataset.Type,
		dataset.SampleCount,
		dataset.InferenceMode,
		dataset.ConfigPath,
		nullableJSON(dataset.ExtraJSON),
		dataset.HuggingFaceRepo,
		dataset.HuggingFaceSubset,
		dataset.LocalPath,
		dataset.FileFormat,
	)
	if err != nil {
		return fmt.Errorf("Upsert HuggingFace Dataset 失败: %w", err)
	}
	return nil
}

// ListHuggingFaceRepos 返回已拉取的 HuggingFace 数据集仓库 ID 列表。
func (r *DatasetRepository) ListHuggingFaceRepos(ctx context.Context) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT hf_repo FROM datasets WHERE source = 'huggingface' AND hf_repo != ''`)
	if err != nil {
		return nil, fmt.Errorf("查询 HuggingFace repos 失败: %w", err)
	}
	defer rows.Close()
	var repos []string
	for rows.Next() {
		var repo string
		if err := rows.Scan(&repo); err != nil {
			continue
		}
		repos = append(repos, repo)
	}
	return repos, nil
}

func scanDataset(row *sql.Row) (*domain.Dataset, error) {
	var ds domain.Dataset
	var source string
	var enabledInt int
	if err := row.Scan(
		&ds.ID,
		&ds.Code,
		&ds.DisplayName,
		&ds.Description,
		&ds.Type,
		&source,
		&ds.SampleCount,
		&enabledInt,
		&ds.InferenceMode,
		&ds.ConfigPath,
		&ds.ExtraJSON,
		&ds.HuggingFaceRepo,
		&ds.HuggingFaceSubset,
		&ds.LocalPath,
		&ds.FileFormat,
		&ds.CreatedAt,
		&ds.UpdatedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("Dataset 不存在")
		}
		return nil, fmt.Errorf("查询 Dataset 失败: %w", err)
	}
	ds.Source = domain.DatasetSource(source)
	ds.Enabled = enabledInt != 0
	return &ds, nil
}

func scanRowsInto(rows *sql.Rows, ds *domain.Dataset) error {
	var source string
	var enabledInt int
	if err := rows.Scan(
		&ds.ID,
		&ds.Code,
		&ds.DisplayName,
		&ds.Description,
		&ds.Type,
		&source,
		&ds.SampleCount,
		&enabledInt,
		&ds.InferenceMode,
		&ds.ConfigPath,
		&ds.ExtraJSON,
		&ds.HuggingFaceRepo,
		&ds.HuggingFaceSubset,
		&ds.LocalPath,
		&ds.FileFormat,
		&ds.CreatedAt,
		&ds.UpdatedAt,
	); err != nil {
		return fmt.Errorf("扫描 Dataset 失败: %w", err)
	}
	ds.Source = domain.DatasetSource(source)
	ds.Enabled = enabledInt != 0
	return nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func nullableJSON(s string) string {
	if s == "" {
		return "{}"
	}
	return s
}
