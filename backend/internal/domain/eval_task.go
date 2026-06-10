package domain

import "time"

type EvalTaskStatus string

const (
	EvalTaskStatusPending    EvalTaskStatus = "pending"
	EvalTaskStatusValidating EvalTaskStatus = "validating"
	EvalTaskStatusBuilding   EvalTaskStatus = "building"
	EvalTaskStatusRunning    EvalTaskStatus = "running"
	EvalTaskStatusParsing    EvalTaskStatus = "parsing"
	EvalTaskStatusSucceeded  EvalTaskStatus = "succeeded"
	EvalTaskStatusFailed     EvalTaskStatus = "failed"
	EvalTaskStatusTimeout    EvalTaskStatus = "timeout"
	EvalTaskStatusCancelled  EvalTaskStatus = "cancelled"
)

type EvalTask struct {
	ID            string
	TaskName      string
	UserID        int64
	ModelProvider string
	ModelName     string
	ModelBaseURL  string
	ModelPresetID int64  // 创建时使用的预设 ID（0 表示手动填写）
	EvaluatorType string // rouge / accuracy / keyword_match / em / bleu / jieba_rouge
	DatasetType   string
	DatasetName   string
	Status        EvalTaskStatus
	OutputDir     string
	ErrorCode     string
	ErrorMessage  string
	CreatedAt     time.Time
	UpdatedAt     time.Time
	StartedAt     *time.Time
	FinishedAt    *time.Time
}

func (s EvalTaskStatus) IsTerminal() bool {
	return s == EvalTaskStatusSucceeded ||
		s == EvalTaskStatusFailed ||
		s == EvalTaskStatusTimeout ||
		s == EvalTaskStatusCancelled
}

type EvalTaskListQuery struct {
	UserID      int64
	Statuses    []EvalTaskStatus
	Keyword     string
	DatasetType string
	// Search 在任务显示名称、任务 ID 上模糊匹配（与 Keyword 的模型名筛选可同时生效）。
	Search      string
	CreatedFrom *time.Time
	CreatedTo   *time.Time
	Page        int
	PageSize    int
}
