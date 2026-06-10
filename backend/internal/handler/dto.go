package handler

import (
	"time"

	"eval-dominator/backend/internal/domain"
)

// 直接面向前端用户阅读，不带 ISO 8601 的 T/Z；时区按 Go 进程本地时区。
const dtoTimeLayout = "2006-01-02 15:04:05"

type evalTaskItem struct {
	EvalTaskID    string `json:"evalTaskId"`
	TaskName      string `json:"taskName"`
	ModelProvider string `json:"modelProvider"`
	ModelName     string `json:"modelName"`
	ModelBaseURL  string `json:"modelBaseUrl"`
	ModelPresetID int64  `json:"modelPresetId"`
	EvaluatorType string `json:"evaluatorType"`
	DatasetType   string `json:"datasetType"`
	DatasetName   string `json:"datasetName"`
	Status        string `json:"status"`
	Progress      int    `json:"progress,omitempty"`     // tqdm progress percentage during running, -1 if unavailable
	ProgressText  string `json:"progressText,omitempty"` // e.g. "62/139"
	RunningPhase  string `json:"runningPhase,omitempty"` // "infer" / "eval" / "" — 仅 status=running 时有意义
	OutputDir     string `json:"outputDir,omitempty"`
	ErrorCode     string `json:"errorCode"`
	ErrorMessage  string `json:"errorMessage"`
	CreatedAt     string `json:"createdAt"`
	UpdatedAt     string `json:"updatedAt"`
	StartedAt     string `json:"startedAt,omitempty"`
	FinishedAt    string `json:"finishedAt,omitempty"`
}

func toEvalTaskItem(task domain.EvalTask) evalTaskItem {
	return evalTaskItem{
		EvalTaskID:    task.ID,
		TaskName:      task.TaskName,
		ModelProvider: task.ModelProvider,
		ModelName:     task.ModelName,
		ModelBaseURL:  task.ModelBaseURL,
		ModelPresetID: task.ModelPresetID,
		EvaluatorType: task.EvaluatorType,
		DatasetType:   task.DatasetType,
		DatasetName:   task.DatasetName,
		Status:        string(task.Status),
		Progress:      -1,
		OutputDir:     task.OutputDir,
		ErrorCode:     task.ErrorCode,
		ErrorMessage:  task.ErrorMessage,
		CreatedAt:     fmtTime(&task.CreatedAt),
		UpdatedAt:     fmtTime(&task.UpdatedAt),
		StartedAt:     fmtTime(task.StartedAt),
		FinishedAt:    fmtTime(task.FinishedAt),
	}
}

func toEvalTaskItemWithProgress(task domain.EvalTask, progress int, progressText string, runningPhase string) evalTaskItem {
	item := toEvalTaskItem(task)
	item.Progress = progress
	item.ProgressText = progressText
	item.RunningPhase = runningPhase
	return item
}

type modelItem struct {
	ID          int64  `json:"id"`
	Provider    string `json:"provider"`
	ModelName   string `json:"modelName"`
	DisplayName string `json:"displayName"`
	Version     string `json:"version"`
	BaseURL     string `json:"baseUrl"`
	MaskedKey   string `json:"maskedKey"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
}

func toModelItem(model domain.Model) modelItem {
	return modelItem{
		ID:          model.ID,
		Provider:    model.Provider,
		ModelName:   model.ModelName,
		DisplayName: model.DisplayName,
		Version:     model.Version,
		BaseURL:     model.BaseURL,
		MaskedKey:   model.MaskedKey,
		CreatedAt:   fmtTime(&model.CreatedAt),
		UpdatedAt:   fmtTime(&model.UpdatedAt),
	}
}

func fmtTime(t *time.Time) string {
	if t == nil || t.IsZero() {
		return ""
	}
	return t.Local().Format(dtoTimeLayout)
}
