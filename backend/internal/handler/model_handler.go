package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"eval-dominator/backend/internal/application"
	"eval-dominator/backend/internal/middleware"
)

type ModelHandler struct {
	modelService *application.ModelService
}

type modelRequest struct {
	Provider    string `json:"provider"`
	ModelName   string `json:"modelName"`
	DisplayName string `json:"displayName"`
	Version     string `json:"version"`
	BaseURL     string `json:"baseUrl"`
	APIKey      string `json:"apiKey"`
}

func NewModelHandler(modelService *application.ModelService) *ModelHandler {
	return &ModelHandler{modelService: modelService}
}

func (h *ModelHandler) List(ctx *gin.Context) {
	userID := ctx.GetInt64(middleware.UserIDKey)
	models, err := h.modelService.List(ctx.Request.Context(), userID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": "LIST_MODEL_FAILED", "message": err.Error()})
		return
	}
	items := make([]modelItem, 0, len(models))
	for _, m := range models {
		items = append(items, toModelItem(m))
	}
	ctx.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *ModelHandler) Create(ctx *gin.Context) {
	var req modelRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_ARGUMENT", "message": "请求参数错误"})
		return
	}
	userID := ctx.GetInt64(middleware.UserIDKey)
	model, err := h.modelService.Create(ctx.Request.Context(), application.CreateModelInput{
		UserID:      userID,
		Provider:    req.Provider,
		ModelName:   req.ModelName,
		DisplayName: req.DisplayName,
		Version:     req.Version,
		BaseURL:     req.BaseURL,
		APIKey:      req.APIKey,
	})
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": "CREATE_MODEL_FAILED", "message": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, toModelItem(*model))
}

func (h *ModelHandler) Update(ctx *gin.Context) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_ARGUMENT", "message": "ID 不合法"})
		return
	}
	var req modelRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_ARGUMENT", "message": "请求参数错误"})
		return
	}
	userID := ctx.GetInt64(middleware.UserIDKey)
	model, err := h.modelService.Update(ctx.Request.Context(), id, userID, application.UpdateModelInput{
		Provider:    req.Provider,
		ModelName:   req.ModelName,
		DisplayName: req.DisplayName,
		Version:     req.Version,
		BaseURL:     req.BaseURL,
		APIKey:      req.APIKey,
	})
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": "UPDATE_MODEL_FAILED", "message": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, toModelItem(*model))
}

func (h *ModelHandler) Delete(ctx *gin.Context) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_ARGUMENT", "message": "ID 不合法"})
		return
	}
	userID := ctx.GetInt64(middleware.UserIDKey)
	if err := h.modelService.Delete(ctx.Request.Context(), id, userID); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": "DELETE_MODEL_FAILED", "message": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{})
}
