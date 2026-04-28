package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func CORS() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		origin := ctx.GetHeader("Origin")
		if origin == "" {
			origin = "*"
		}

		ctx.Header("Access-Control-Allow-Origin", origin)
		ctx.Header("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
		ctx.Header("Access-Control-Allow-Headers", "Authorization,Content-Type")
		ctx.Header("Access-Control-Allow-Credentials", "true")

		// 浏览器跨端口调用会先发 OPTIONS 预检请求，这里直接放行。
		if ctx.Request.Method == http.MethodOptions {
			ctx.AbortWithStatus(http.StatusNoContent)
			return
		}

		ctx.Next()
	}
}
