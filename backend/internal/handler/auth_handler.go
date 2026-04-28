package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"eval-dominator/backend/internal/application"
	"eval-dominator/backend/internal/middleware"
)

type AuthHandler struct {
	authService *application.AuthService
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type changePasswordRequest struct {
	OldPassword string `json:"oldPassword"`
	NewPassword string `json:"newPassword"`
}

func NewAuthHandler(authService *application.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

func (h *AuthHandler) Login(ctx *gin.Context) {
	var req loginRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_ARGUMENT", "message": "请求参数错误"})
		return
	}

	token, expiresAt, err := h.authService.Login(ctx.Request.Context(), req.Username, req.Password)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"code": "LOGIN_FAILED", "message": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"token":     token,
		"expiresAt": expiresAt,
	})
}

// ChangePassword 修改自己的密码。Auth 中间件已注入 username。
func (h *AuthHandler) ChangePassword(ctx *gin.Context) {
	var req changePasswordRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_ARGUMENT", "message": "请求参数错误"})
		return
	}
	username, _ := ctx.Get(middleware.UsernameKey)
	name, _ := username.(string)
	if name == "" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"code": "UNAUTHORIZED", "message": "无法识别当前用户"})
		return
	}
	if err := h.authService.ChangePassword(ctx.Request.Context(), name, req.OldPassword, req.NewPassword); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": "CHANGE_PASSWORD_FAILED", "message": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *AuthHandler) Me(ctx *gin.Context) {
	userID := ctx.GetInt64(middleware.UserIDKey)
	username, _ := ctx.Get(middleware.UsernameKey)
	name, _ := username.(string)
	ctx.JSON(http.StatusOK, gin.H{
		"userId":   userID,
		"username": name,
	})
}

func (h *AuthHandler) Logout(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{})
}
