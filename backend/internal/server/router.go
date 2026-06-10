package server

import (
	"github.com/gin-gonic/gin"

	"eval-dominator/backend/internal/application"
	"eval-dominator/backend/internal/handler"
	"eval-dominator/backend/internal/middleware"
)

type Services struct {
	Auth    *application.AuthService
	Eval    *application.EvalService
	Model   *application.ModelService
	System  *application.SystemService
	Dataset *application.DatasetService
}

func NewRouter(services Services) *gin.Engine {
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery(), middleware.CORS())

	authHandler := handler.NewAuthHandler(services.Auth)
	evalHandler := handler.NewEvalHandler(services.Eval)
	modelHandler := handler.NewModelHandler(services.Model)
	systemHandler := handler.NewSystemHandler(services.System)
	datasetHandler := handler.NewDatasetHandler(services.Dataset)

	api := router.Group("/api")
	api.POST("/auth/login", authHandler.Login)

	protected := api.Group("")
	protected.Use(middleware.Auth(services.Auth))

	protected.GET("/auth/me", authHandler.Me)
	protected.POST("/auth/logout", authHandler.Logout)
	protected.POST("/auth/change-password", authHandler.ChangePassword)

	protected.GET("/eval/tasks", evalHandler.ListTasks)
	protected.POST("/eval/tasks", evalHandler.CreateTask)
	protected.GET("/eval/tasks/:evalTaskId", evalHandler.GetTask)
	protected.GET("/eval/tasks/:evalTaskId/result", evalHandler.GetResult)
	protected.GET("/eval/tasks/:evalTaskId/log", evalHandler.GetTaskLog)
	protected.GET("/eval/tasks/:evalTaskId/logs", evalHandler.ListTaskLogs)
	protected.POST("/eval/tasks/:evalTaskId/cancel", evalHandler.CancelTask)
	protected.POST("/eval/tasks/:evalTaskId/rerun-eval", evalHandler.RerunEvaluateNode)
	protected.GET("/eval/tasks/:evalTaskId/analysis", evalHandler.GetAnalysis)
	protected.GET("/eval/tasks/:evalTaskId/artifacts/preview", evalHandler.PreviewArtifact)
	protected.GET("/eval/tasks/:evalTaskId/artifacts/download", evalHandler.DownloadArtifact)

	protected.GET("/models", modelHandler.List)
	protected.POST("/models", modelHandler.Create)
	protected.PUT("/models/:id", modelHandler.Update)
	protected.DELETE("/models/:id", modelHandler.Delete)

	protected.GET("/datasets", datasetHandler.List)
	protected.POST("/datasets", datasetHandler.Create)
	protected.PUT("/datasets/:id", datasetHandler.Update)
	protected.PATCH("/datasets/:id/enabled", datasetHandler.SetEnabled)
	protected.DELETE("/datasets/:id", datasetHandler.Delete)
	protected.POST("/datasets/sync", datasetHandler.Sync)
	protected.GET("/datasets/search-huggingface", datasetHandler.SearchHuggingFace)
	protected.GET("/datasets/huggingface-detail", datasetHandler.GetHuggingFaceDetail)
	protected.POST("/datasets/pull-huggingface", datasetHandler.PullHuggingFace)
	protected.POST("/datasets/upload", datasetHandler.Upload)
	protected.POST("/datasets/custom-from-path", datasetHandler.CreateFromPath)
	protected.GET("/datasets/demo", datasetHandler.ListDemos)
	protected.GET("/datasets/:id/preview", datasetHandler.Preview)
	protected.GET("/datasets/preview-by-path", datasetHandler.PreviewByPath)

	protected.GET("/system/health", systemHandler.Health)

	return router
}
