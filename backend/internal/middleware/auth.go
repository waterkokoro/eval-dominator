package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"eval-dominator/backend/internal/application"
)

const (
	UserIDKey   = "userID"
	UsernameKey = "username"
)

func Auth(authService *application.AuthService) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		header := ctx.GetHeader("Authorization")
		if header == "" || !strings.HasPrefix(header, "Bearer ") {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    "UNAUTHORIZED",
				"message": "缺少认证信息",
			})
			return
		}

		claims, err := authService.ParseTokenClaims(strings.TrimPrefix(header, "Bearer "))
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    "UNAUTHORIZED",
				"message": "认证信息无效",
			})
			return
		}

		ctx.Set(UserIDKey, claims.UserID)
		ctx.Set(UsernameKey, claims.Username)
		ctx.Next()
	}
}
