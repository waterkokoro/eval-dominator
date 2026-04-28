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
	Params        map[string]string `json:"params"`
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
		SaveModel:     req.SaveModel,
		Params:        req.Params,
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
	ctx.JSON(http.StatusOK, toEvalTaskItem(*task))
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
	content, err := h.evalService.GetTaskLog(ctx.Request.Context(), ctx.Param("evalTaskId"), tail)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": "READ_LOG_FAILED", "message": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"content": content, "tail": tail})
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
