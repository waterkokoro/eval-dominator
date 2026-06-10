package handler

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"eval-dominator/backend/internal/application"
	"eval-dominator/backend/internal/domain"
	"eval-dominator/backend/internal/middleware"
)

type EvalHandler struct {
	evalService *application.EvalService
}

type createEvalTaskRequest struct {
	TaskName      string            `json:"taskName"`
	Provider      string            `json:"provider"`
	ModelName     string            `json:"modelName"`
	DisplayName   string            `json:"displayName"`
	Version       string            `json:"version"`
	BaseURL       string            `json:"baseUrl"`
	APIKey        string            `json:"apiKey"`
	ModelPresetID int64             `json:"modelPresetId"`
	SaveModel     bool              `json:"saveModel"`
	DatasetType   string            `json:"datasetType"`
	DatasetName   string            `json:"datasetName"`
	EvaluatorType string            `json:"evaluatorType"` // rouge / accuracy / keyword_match / em / bleu / jieba_rouge
	Params        map[string]string `json:"params"`
	Runtime       *struct {
		TimeoutSeconds int  `json:"timeoutSeconds"`
		MaxWorkers     int  `json:"maxWorkers"`
		KeepRawOutputs bool `json:"keepRawOutputs"`
	} `json:"runtime"`
}

func NewEvalHandler(evalService *application.EvalService) *EvalHandler {
	return &EvalHandler{evalService: evalService}
}

func (h *EvalHandler) CreateTask(ctx *gin.Context) {
	var req createEvalTaskRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_ARGUMENT", "message": "请求参数错误"})
		return
	}

	userID := ctx.GetInt64(middleware.UserIDKey)
	task, err := h.evalService.CreateTask(ctx.Request.Context(), application.CreateEvalTaskInput{
		TaskName:      req.TaskName,
		UserID:        userID,
		Provider:      req.Provider,
		ModelName:     req.ModelName,
		DisplayName:   req.DisplayName,
		Version:       req.Version,
		BaseURL:       req.BaseURL,
		APIKey:        req.APIKey,
		ModelPresetID: req.ModelPresetID,
		DatasetType:   req.DatasetType,
		DatasetName:   req.DatasetName,
		EvaluatorType: req.EvaluatorType,
		SaveModel:     req.SaveModel,
		Params:        req.Params,
		Runtime:       toRuntimeInput(req.Runtime),
	})
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": "CREATE_EVAL_TASK_FAILED", "message": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"evalTaskId": task.ID,
		"status":     task.Status,
	})
}

func (h *EvalHandler) GetTask(ctx *gin.Context) {
	task, err := h.evalService.GetTask(ctx.Request.Context(), ctx.Param("evalTaskId"))
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"code": "EVAL_TASK_NOT_FOUND", "message": err.Error()})
		return
	}
	progress, progressText, runningPhase := h.evalService.GetTaskProgress(ctx.Request.Context(), task.ID)
	ctx.JSON(http.StatusOK, toEvalTaskItemWithProgress(*task, progress, progressText, runningPhase))
}

func (h *EvalHandler) GetResult(ctx *gin.Context) {
	result, err := h.evalService.GetResult(ctx.Request.Context(), ctx.Param("evalTaskId"))
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"code": "EVAL_RESULT_NOT_FOUND", "message": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"evalTaskId":    result.EvalTaskID,
		"metricsJson":   result.MetricsJSON,
		"artifactsJson": result.ArtifactsJSON,
		"rawResultPath": result.RawResultPath,
		"reportPath":    result.ReportPath,
		"logPath":       result.LogPath,
		"metadataJson":  result.MetadataJSON,
	})
}

func (h *EvalHandler) ListTasks(ctx *gin.Context) {
	userID := ctx.GetInt64(middleware.UserIDKey)
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("pageSize", "10"))

	statusParam := strings.TrimSpace(ctx.Query("status"))
	statuses := make([]domain.EvalTaskStatus, 0)
	if statusParam != "" {
		for _, s := range strings.Split(statusParam, ",") {
			s = strings.TrimSpace(s)
			if s != "" {
				statuses = append(statuses, domain.EvalTaskStatus(s))
			}
		}
	}

	query := domain.EvalTaskListQuery{
		UserID:      userID,
		Statuses:    statuses,
		Keyword:     strings.TrimSpace(ctx.Query("keyword")),
		DatasetType: strings.TrimSpace(ctx.Query("datasetType")),
		Search:      strings.TrimSpace(ctx.Query("search")),
		Page:        page,
		PageSize:    pageSize,
	}
	if t, ok := parseDateStartLocal(ctx.Query("createdFrom")); ok {
		query.CreatedFrom = &t
	}
	if t, ok := parseDateEndLocal(ctx.Query("createdTo")); ok {
		query.CreatedTo = &t
	}

	result, err := h.evalService.ListTasks(ctx.Request.Context(), query)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": "LIST_EVAL_TASK_FAILED", "message": err.Error()})
		return
	}

	items := make([]evalTaskItem, 0, len(result.Items))
	for _, task := range result.Items {
		items = append(items, toEvalTaskItem(task))
	}
	ctx.JSON(http.StatusOK, gin.H{
		"items":    items,
		"total":    result.Total,
		"page":     result.Page,
		"pageSize": result.PageSize,
	})
}

func (h *EvalHandler) GetTaskLog(ctx *gin.Context) {
	tail, _ := strconv.Atoi(ctx.DefaultQuery("tail", "200"))
	logID := strings.TrimSpace(ctx.Query("logId"))
	content, err := h.evalService.GetTaskLog(ctx.Request.Context(), ctx.Param("evalTaskId"), logID, tail)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": "READ_LOG_FAILED", "message": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"content": content, "tail": tail, "logId": logID})
}

// ListTaskLogs GET /eval/tasks/:id/logs
// 列举该任务全部可用日志文件，前端用于渲染左侧日志菜单。
func (h *EvalHandler) ListTaskLogs(ctx *gin.Context) {
	logs := h.evalService.ListTaskLogs(ctx.Request.Context(), ctx.Param("evalTaskId"))
	// 不返回绝对路径给前端，避免泄露服务器目录
	items := make([]map[string]interface{}, 0, len(logs))
	for _, lg := range logs {
		items = append(items, map[string]interface{}{
			"id":          lg.ID,
			"displayName": lg.DisplayName,
			"type":        lg.Type,
			"mtime":       lg.ModTime,
			"size":        lg.Size,
		})
	}
	ctx.JSON(http.StatusOK, gin.H{"items": items})
}

// CancelTask POST /eval/tasks/:id/cancel
func (h *EvalHandler) CancelTask(ctx *gin.Context) {
	userID := ctx.GetInt64(middleware.UserIDKey)
	if err := h.evalService.CancelTask(ctx.Request.Context(), ctx.Param("evalTaskId"), userID); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": "CANCEL_TASK_FAILED", "message": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"ok": true})
}

// RerunEvaluateNode POST /eval/tasks/:id/rerun-eval
// 仅重跑 evaluate 节点（复用旧的 predictions），不重新调用 LLM。
func (h *EvalHandler) RerunEvaluateNode(ctx *gin.Context) {
	userID := ctx.GetInt64(middleware.UserIDKey)
	if err := h.evalService.RerunEvaluateNode(ctx.Request.Context(), ctx.Param("evalTaskId"), userID); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": "RERUN_EVAL_FAILED", "message": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"ok": true})
}

// GetAnalysis GET /eval/tasks/:id/analysis
// 返回该任务的逐题分析数据（prompt / prediction / 关键词命中 / 评分等）。
func (h *EvalHandler) GetAnalysis(ctx *gin.Context) {
	userID := ctx.GetInt64(middleware.UserIDKey)
	data, err := h.evalService.GetAnalysis(ctx.Request.Context(), ctx.Param("evalTaskId"), userID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": "ANALYSIS_NOT_AVAILABLE", "message": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, data)
}

// PreviewArtifact GET /eval/tasks/:id/artifacts/preview?path=...
// 读取文本类产物前 ~512KB 用于在线预览。
func (h *EvalHandler) PreviewArtifact(ctx *gin.Context) {
	evalTaskID := ctx.Param("evalTaskId")
	path := ctx.Query("path")
	file, content, truncated, err := h.evalService.PreviewArtifactFile(ctx.Request.Context(), evalTaskID, path)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": "PREVIEW_ARTIFACT_FAILED", "message": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{
		"path":         file.AbsolutePath,
		"relativePath": file.RelativePath,
		"size":         file.Size,
		"isText":       file.IsText,
		"truncated":    truncated,
		"content":      content,
		"contentType":  file.ContentType,
	})
}

// DownloadArtifact GET /eval/tasks/:id/artifacts/download?path=...
// 直接以二进制流返回，强制 Content-Disposition: attachment。
func (h *EvalHandler) DownloadArtifact(ctx *gin.Context) {
	evalTaskID := ctx.Param("evalTaskId")
	path := ctx.Query("path")
	file, err := h.evalService.ResolveArtifactFile(ctx.Request.Context(), evalTaskID, path)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": "DOWNLOAD_ARTIFACT_FAILED", "message": err.Error()})
		return
	}
	filename := strings.ReplaceAll(file.RelativePath, "/", "_")
	ctx.Header("Content-Type", file.ContentType)
	ctx.Header("Content-Disposition", "attachment; filename=\""+filename+"\"")
	ctx.File(file.AbsolutePath)
}

func parseDateStartLocal(s string) (time.Time, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, false
	}
	t, err := time.ParseInLocation("2006-01-02", s, time.Local)
	if err != nil {
		return time.Time{}, false
	}
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location()), true
}

func parseDateEndLocal(s string) (time.Time, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, false
	}
	t, err := time.ParseInLocation("2006-01-02", s, time.Local)
	if err != nil {
		return time.Time{}, false
	}
	return time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 999999999, t.Location()), true
}

// toRuntimeInput 将前端传来的 runtime 参数转换为 application 层结构。
func toRuntimeInput(r *struct {
	TimeoutSeconds int  `json:"timeoutSeconds"`
	MaxWorkers     int  `json:"maxWorkers"`
	KeepRawOutputs bool `json:"keepRawOutputs"`
}) application.RuntimeInput {
	if r == nil {
		return application.RuntimeInput{}
	}
	return application.RuntimeInput{
		TimeoutSeconds: r.TimeoutSeconds,
		MaxWorkers:     r.MaxWorkers,
		KeepRawOutputs: r.KeepRawOutputs,
		HasValues:      true,
	}
}
