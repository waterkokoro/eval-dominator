package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"eval-dominator/backend/internal/domain"
)

type EvalTaskRepository struct {
	db *sql.DB
}

func NewEvalTaskRepository(db *sql.DB) *EvalTaskRepository {
	return &EvalTaskRepository{db: db}
}

func (r *EvalTaskRepository) Create(ctx context.Context, task domain.EvalTask) error {
	_, err := r.db.ExecContext(
		ctx,
		`INSERT INTO eval_tasks (
			id, task_name, user_id, model_provider, model_name, model_base_url,
			model_preset_id, evaluator_type,
			dataset_type, dataset_name, status, output_dir, error_code, error_message
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		task.ID,
		task.TaskName,
		task.UserID,
		task.ModelProvider,
		task.ModelName,
		task.ModelBaseURL,
		task.ModelPresetID,
		task.EvaluatorType,
		task.DatasetType,
		task.DatasetName,
		string(task.Status),
		task.OutputDir,
		task.ErrorCode,
		task.ErrorMessage,
	)
	if err != nil {
		return fmt.Errorf("创建 EvalTask 失败: %w", err)
	}
	return nil
}

func (r *EvalTaskRepository) UpdateStatus(ctx context.Context, id string, status domain.EvalTaskStatus, outputDir string, errorCode string, errorMessage string) error {
	now := time.Now()
	var startedAt any
	var finishedAt any
	if status == domain.EvalTaskStatusRunning {
		startedAt = now
	}
	if status.IsTerminal() {
		finishedAt = now
	}

	_, err := r.db.ExecContext(
		ctx,
		`UPDATE eval_tasks
		SET status = ?, output_dir = COALESCE(NULLIF(?, ''), output_dir),
			error_code = ?, error_message = ?, updated_at = ?,
			started_at = COALESCE(?, started_at), finished_at = COALESCE(?, finished_at)
		WHERE id = ?`,
		string(status),
		outputDir,
		errorCode,
		errorMessage,
		now,
		startedAt,
		finishedAt,
		id,
	)
	if err != nil {
		return fmt.Errorf("更新 EvalTask 状态失败: %w", err)
	}
	return nil
}

func (r *EvalTaskRepository) GetByID(ctx context.Context, id string) (*domain.EvalTask, error) {
	row := r.db.QueryRowContext(
		ctx,
		`SELECT id, task_name, user_id, model_provider, model_name, model_base_url,
			COALESCE(model_preset_id, 0), COALESCE(evaluator_type, ''),
			dataset_type, dataset_name, status, output_dir, error_code, error_message,
			created_at, updated_at, started_at, finished_at
		FROM eval_tasks WHERE id = ?`,
		id,
	)

	var task domain.EvalTask
	var status string
	if err := row.Scan(
		&task.ID,
		&task.TaskName,
		&task.UserID,
		&task.ModelProvider,
		&task.ModelName,
		&task.ModelBaseURL,
		&task.ModelPresetID,
		&task.EvaluatorType,
		&task.DatasetType,
		&task.DatasetName,
		&status,
		&task.OutputDir,
		&task.ErrorCode,
		&task.ErrorMessage,
		&task.CreatedAt,
		&task.UpdatedAt,
		&task.StartedAt,
		&task.FinishedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("EvalTask 不存在")
		}
		return nil, fmt.Errorf("查询 EvalTask 失败: %w", err)
	}

	task.Status = domain.EvalTaskStatus(status)
	return &task, nil
}

func (r *EvalTaskRepository) List(ctx context.Context, query domain.EvalTaskListQuery) ([]domain.EvalTask, int64, error) {
	conditions := []string{"1 = 1"}
	args := []any{}

	if query.UserID > 0 {
		conditions = append(conditions, "user_id = ?")
		args = append(args, query.UserID)
	}
	if len(query.Statuses) > 0 {
		placeholders := make([]string, 0, len(query.Statuses))
		for _, status := range query.Statuses {
			placeholders = append(placeholders, "?")
			args = append(args, string(status))
		}
		conditions = append(conditions, fmt.Sprintf("status IN (%s)", strings.Join(placeholders, ",")))
	}
	if query.Keyword != "" {
		conditions = append(conditions, "(model_name LIKE ? OR model_provider LIKE ?)")
		like := "%" + query.Keyword + "%"
		args = append(args, like, like)
	}
	if query.DatasetType != "" {
		conditions = append(conditions, "dataset_type = ?")
		args = append(args, query.DatasetType)
	}
	if q := strings.TrimSpace(query.Search); q != "" {
		conditions = append(conditions, "(id LIKE ? OR task_name LIKE ?)")
		like := "%" + q + "%"
		args = append(args, like, like)
	}
	if query.CreatedFrom != nil {
		conditions = append(conditions, "created_at >= ?")
		args = append(args, query.CreatedFrom.Format("2006-01-02 15:04:05"))
	}
	if query.CreatedTo != nil {
		conditions = append(conditions, "created_at <= ?")
		args = append(args, query.CreatedTo.Format("2006-01-02 15:04:05.999999999"))
	}

	where := strings.Join(conditions, " AND ")

	var total int64
	if err := r.db.QueryRowContext(
		ctx,
		fmt.Sprintf("SELECT COUNT(*) FROM eval_tasks WHERE %s", where),
		args...,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("统计 EvalTask 数量失败: %w", err)
	}

	pageSize := query.PageSize
	if pageSize <= 0 {
		pageSize = 10
	}
	page := query.Page
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * pageSize

	listArgs := append([]any{}, args...)
	listArgs = append(listArgs, pageSize, offset)

	rows, err := r.db.QueryContext(
		ctx,
		fmt.Sprintf(`SELECT id, task_name, user_id, model_provider, model_name, model_base_url,
			COALESCE(model_preset_id, 0), COALESCE(evaluator_type, ''),
			dataset_type, dataset_name, status, output_dir, error_code, error_message,
			created_at, updated_at, started_at, finished_at
		FROM eval_tasks WHERE %s ORDER BY created_at DESC LIMIT ? OFFSET ?`, where),
		listArgs...,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("查询 EvalTask 列表失败: %w", err)
	}
	defer rows.Close()

	tasks := make([]domain.EvalTask, 0)
	for rows.Next() {
		var task domain.EvalTask
		var status string
		if err := rows.Scan(
			&task.ID,
			&task.TaskName,
			&task.UserID,
			&task.ModelProvider,
			&task.ModelName,
			&task.ModelBaseURL,
			&task.ModelPresetID,
			&task.EvaluatorType,
			&task.DatasetType,
			&task.DatasetName,
			&status,
			&task.OutputDir,
			&task.ErrorCode,
			&task.ErrorMessage,
			&task.CreatedAt,
			&task.UpdatedAt,
			&task.StartedAt,
			&task.FinishedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("扫描 EvalTask 失败: %w", err)
		}
		task.Status = domain.EvalTaskStatus(status)
		tasks = append(tasks, task)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("遍历 EvalTask 列表失败: %w", err)
	}

	return tasks, total, nil
}

type EvalResultRepository struct {
	db *sql.DB
}

func NewEvalResultRepository(db *sql.DB) *EvalResultRepository {
	return &EvalResultRepository{db: db}
}

func (r *EvalResultRepository) Save(ctx context.Context, result domain.EvalResult) error {
	_, err := r.db.ExecContext(
		ctx,
		`INSERT INTO eval_results (
			eval_task_id, metrics_json, artifacts_json, raw_result_path,
			report_path, log_path, metadata_json
		) VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(eval_task_id) DO UPDATE SET
			metrics_json = excluded.metrics_json,
			artifacts_json = excluded.artifacts_json,
			raw_result_path = excluded.raw_result_path,
			report_path = excluded.report_path,
			log_path = excluded.log_path,
			metadata_json = excluded.metadata_json`,
		result.EvalTaskID,
		result.MetricsJSON,
		result.ArtifactsJSON,
		result.RawResultPath,
		result.ReportPath,
		result.LogPath,
		result.MetadataJSON,
	)
	if err != nil {
		return fmt.Errorf("保存 EvalResult 失败: %w", err)
	}
	return nil
}

func (r *EvalResultRepository) GetByTaskID(ctx context.Context, evalTaskID string) (*domain.EvalResult, error) {
	row := r.db.QueryRowContext(
		ctx,
		`SELECT eval_task_id, metrics_json, artifacts_json, raw_result_path,
			report_path, log_path, metadata_json
		FROM eval_results WHERE eval_task_id = ?`,
		evalTaskID,
	)

	var result domain.EvalResult
	if err := row.Scan(
		&result.EvalTaskID,
		&result.MetricsJSON,
		&result.ArtifactsJSON,
		&result.RawResultPath,
		&result.ReportPath,
		&result.LogPath,
		&result.MetadataJSON,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("EvalResult 不存在")
		}
		return nil, fmt.Errorf("查询 EvalResult 失败: %w", err)
	}
	return &result, nil
}
