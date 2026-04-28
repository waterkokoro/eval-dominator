package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"eval-dominator/backend/internal/application"
)

type SystemHandler struct {
	systemService *application.SystemService
}

func NewSystemHandler(systemService *application.SystemService) *SystemHandler {
	return &SystemHandler{systemService: systemService}
}

func (h *SystemHandler) Health(ctx *gin.Context) {
	report := h.systemService.Health(ctx.Request.Context())
	ctx.JSON(http.StatusOK, report)
}
