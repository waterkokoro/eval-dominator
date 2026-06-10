package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"eval-dominator/backend/internal/application"
	"eval-dominator/backend/internal/domain"
)

type DatasetHandler struct {
	datasetService *application.DatasetService
}

func NewDatasetHandler(datasetService *application.DatasetService) *DatasetHandler {
	return &DatasetHandler{datasetService: datasetService}
}

type datasetItem struct {
	ID                int64  `json:"id"`
	Code              string `json:"code"`
	DisplayName       string `json:"displayName"`
	Description       string `json:"description"`
	Type              string `json:"type"`
	Source            string `json:"source"`
	SampleCount       int    `json:"sampleCount"`
	Enabled           bool   `json:"enabled"`
	InferenceMode     string `json:"inferenceMode"`
	ConfigPath        string `json:"configPath"`
	HuggingFaceRepo   string `json:"huggingFaceRepo,omitempty"`
	HuggingFaceSubset string `json:"huggingFaceSubset,omitempty"`
	LocalPath         string `json:"localPath,omitempty"`
	FileFormat        string `json:"fileFormat,omitempty"`
	CreatedAt         string `json:"createdAt"`
	UpdatedAt         string `json:"updatedAt"`
}

func toDatasetItem(ds domain.Dataset) datasetItem {
	return datasetItem{
		ID:                ds.ID,
		Code:              ds.Code,
		DisplayName:       ds.DisplayName,
		Description:       ds.Description,
		Type:              ds.Type,
		Source:            string(ds.Source),
		SampleCount:       ds.SampleCount,
		Enabled:           ds.Enabled,
		InferenceMode:     ds.InferenceMode,
		ConfigPath:        ds.ConfigPath,
		HuggingFaceRepo:   ds.HuggingFaceRepo,
		HuggingFaceSubset: ds.HuggingFaceSubset,
		LocalPath:         ds.LocalPath,
		FileFormat:        ds.FileFormat,
		CreatedAt:         fmtTime(&ds.CreatedAt),
		UpdatedAt:         fmtTime(&ds.UpdatedAt),
	}
}

type createDatasetRequest struct {
	Code          string `json:"code"`
	DisplayName   string `json:"displayName"`
	Description   string `json:"description"`
	Type          string `json:"type"`
	SampleCount   int    `json:"sampleCount"`
	InferenceMode string `json:"inferenceMode"`
	ConfigPath    string `json:"configPath"`
}

type updateDatasetRequest struct {
	DisplayName   string `json:"displayName"`
	Description   string `json:"description"`
	Type          string `json:"type"`
	SampleCount   int    `json:"sampleCount"`
	Enabled       *bool  `json:"enabled"`
	InferenceMode string `json:"inferenceMode"`
	ConfigPath    string `json:"configPath"`
}

func (h *DatasetHandler) List(ctx *gin.Context) {
	includeDisabled := ctx.Query("includeDisabled") == "1" || ctx.Query("includeDisabled") == "true"
	items, err := h.datasetService.List(ctx.Request.Context(), includeDisabled)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": "LIST_DATASET_FAILED", "message": err.Error()})
		return
	}
	dtos := make([]datasetItem, 0, len(items))
	for _, ds := range items {
		dtos = append(dtos, toDatasetItem(ds))
	}
	ctx.JSON(http.StatusOK, gin.H{"items": dtos})
}

func (h *DatasetHandler) Create(ctx *gin.Context) {
	var req createDatasetRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_ARGUMENT", "message": "请求参数错误"})
		return
	}
	created, err := h.datasetService.Create(ctx.Request.Context(), application.CreateDatasetInput{
		Code:          req.Code,
		DisplayName:   req.DisplayName,
		Description:   req.Description,
		Type:          req.Type,
		SampleCount:   req.SampleCount,
		InferenceMode: req.InferenceMode,
		ConfigPath:    req.ConfigPath,
	})
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": "CREATE_DATASET_FAILED", "message": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, toDatasetItem(*created))
}

func (h *DatasetHandler) Update(ctx *gin.Context) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_ARGUMENT", "message": "ID 不合法"})
		return
	}
	var req updateDatasetRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_ARGUMENT", "message": "请求参数错误"})
		return
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	updated, err := h.datasetService.Update(ctx.Request.Context(), id, application.UpdateDatasetInput{
		DisplayName:   req.DisplayName,
		Description:   req.Description,
		Type:          req.Type,
		SampleCount:   req.SampleCount,
		Enabled:       enabled,
		InferenceMode: req.InferenceMode,
		ConfigPath:    req.ConfigPath,
	})
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": "UPDATE_DATASET_FAILED", "message": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, toDatasetItem(*updated))
}

type setEnabledRequest struct {
	Enabled bool `json:"enabled"`
}

func (h *DatasetHandler) SetEnabled(ctx *gin.Context) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_ARGUMENT", "message": "ID 不合法"})
		return
	}
	var req setEnabledRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_ARGUMENT", "message": "请求参数错误"})
		return
	}
	if err := h.datasetService.SetEnabled(ctx.Request.Context(), id, req.Enabled); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": "UPDATE_DATASET_FAILED", "message": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"id": id, "enabled": req.Enabled})
}

func (h *DatasetHandler) Delete(ctx *gin.Context) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_ARGUMENT", "message": "ID 不合法"})
		return
	}
	if err := h.datasetService.Delete(ctx.Request.Context(), id); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": "DELETE_DATASET_FAILED", "message": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{})
}

func (h *DatasetHandler) Sync(ctx *gin.Context) {
	result, err := h.datasetService.Sync(ctx.Request.Context())
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": "SYNC_DATASET_FAILED", "message": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, result)
}

// ------------------------------------------------------------------
// HuggingFace 搜索与拉取
// ------------------------------------------------------------------

func (h *DatasetHandler) SearchHuggingFace(ctx *gin.Context) {
	limit := 20
	if l := ctx.Query("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			limit = n
		}
	}
	results, err := h.datasetService.SearchHuggingFace(ctx.Request.Context(), application.HuggingFaceSearchParams{
		Keyword: ctx.Query("keyword"),
		Sort:    ctx.Query("sort"),
		Tag:     ctx.Query("tag"),
		Limit:   limit,
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": "HF_SEARCH_FAILED", "message": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"items": results})
}

func (h *DatasetHandler) GetHuggingFaceDetail(ctx *gin.Context) {
	repoID := ctx.Query("repo")
	if repoID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_ARGUMENT", "message": "repo 参数不能为空"})
		return
	}
	detail, err := h.datasetService.GetHuggingFaceDetail(ctx.Request.Context(), repoID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": "HF_DETAIL_FAILED", "message": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, detail)
}

type pullHuggingFaceRequest struct {
	Repo   string `json:"repo"`
	Subset string `json:"subset"`
	Split  string `json:"split"`
}

func (h *DatasetHandler) PullHuggingFace(ctx *gin.Context) {
	var req pullHuggingFaceRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_ARGUMENT", "message": "请求参数错误"})
		return
	}
	ds, err := h.datasetService.PullHuggingFace(ctx.Request.Context(), application.PullHuggingFaceInput{
		Repo:   req.Repo,
		Subset: req.Subset,
		Split:  req.Split,
	})
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": "HF_PULL_FAILED", "message": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, toDatasetItem(*ds))
}

// ------------------------------------------------------------------
// 自定义数据集（文件上传 / 本地路径）
// ------------------------------------------------------------------

func (h *DatasetHandler) Upload(ctx *gin.Context) {
	file, header, err := ctx.Request.FormFile("file")
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_ARGUMENT", "message": "请上传文件"})
		return
	}
	defer file.Close()

	displayName := ctx.PostForm("displayName")
	description := ctx.PostForm("description")
	taskType := ctx.PostForm("taskType")
	if taskType == "" {
		taskType = "qa"
	}

	ds, err := h.datasetService.CreateCustomFromFile(ctx.Request.Context(), application.CreateCustomFromFileInput{
		DisplayName: displayName,
		Description: description,
		TaskType:    taskType,
		FileName:    header.Filename,
		FileData:    file,
	})
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": "UPLOAD_DATASET_FAILED", "message": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, toDatasetItem(*ds))
}

type customFromPathRequest struct {
	Code        string `json:"code"`
	DisplayName string `json:"displayName"`
	Description string `json:"description"`
	LocalPath   string `json:"localPath"`
	TaskType    string `json:"taskType"`
}

func (h *DatasetHandler) CreateFromPath(ctx *gin.Context) {
	var req customFromPathRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_ARGUMENT", "message": "请求参数错误"})
		return
	}
	ds, err := h.datasetService.CreateCustomFromPath(ctx.Request.Context(), application.CreateCustomFromPathInput{
		Code:        req.Code,
		DisplayName: req.DisplayName,
		Description: req.Description,
		LocalPath:   req.LocalPath,
		TaskType:    req.TaskType,
	})
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": "CREATE_DATASET_FAILED", "message": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, toDatasetItem(*ds))
}

// ------------------------------------------------------------------
// Demo 数据集
// ------------------------------------------------------------------

func (h *DatasetHandler) ListDemos(ctx *gin.Context) {
	demos := h.datasetService.ListDemoDatasets()
	ctx.JSON(http.StatusOK, gin.H{"items": demos})
}

// ------------------------------------------------------------------
// 数据集预览
// ------------------------------------------------------------------

func (h *DatasetHandler) Preview(ctx *gin.Context) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_ARGUMENT", "message": "ID 不合法"})
		return
	}
	limit := 10
	if l := ctx.Query("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			limit = n
		}
	}
	preview, err := h.datasetService.PreviewDataset(ctx.Request.Context(), id, limit)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": "PREVIEW_DATASET_FAILED", "message": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, preview)
}

func (h *DatasetHandler) PreviewByPath(ctx *gin.Context) {
	filePath := ctx.Query("path")
	if filePath == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_ARGUMENT", "message": "path 参数不能为空"})
		return
	}
	limit := 10
	if l := ctx.Query("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			limit = n
		}
	}
	preview, err := h.datasetService.PreviewByPath(ctx.Request.Context(), filePath, limit)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": "PREVIEW_DATASET_FAILED", "message": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, preview)
}
