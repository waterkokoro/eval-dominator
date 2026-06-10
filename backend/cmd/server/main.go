package main

import (
	"context"
	"flag"
	"log"
	"net/http"

	"eval-dominator/backend/internal/application"
	"eval-dominator/backend/internal/config"
	"eval-dominator/backend/internal/infrastructure/database"
	coreclient "eval-dominator/backend/internal/infrastructure/grpc/client"
	"eval-dominator/backend/internal/server"
)

func main() {
	configPath := flag.String("config", "config/config.example.yaml", "后端配置文件路径")
	migrationPath := flag.String("migration", "migrations/001_init.sql", "数据库初始化脚本路径")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	db, err := database.Open(cfg.Database)
	if err != nil {
		log.Fatalf("初始化数据库失败: %v", err)
	}
	defer db.Close()

	if err := database.Migrate(db, *migrationPath); err != nil {
		log.Fatalf("执行数据库迁移失败: %v", err)
	}

	userRepo := database.NewUserRepository(db)
	authService := application.NewAuthService(cfg.JWT, cfg.Auth, userRepo)
	authService.WarnIfDefaultJWTSecret()
	if err := authService.EnsureDefaultUser(context.Background()); err != nil {
		log.Fatalf("初始化默认账号失败: %v", err)
	}

	evalRepo := database.NewEvalTaskRepository(db)
	evalResultRepo := database.NewEvalResultRepository(db)
	modelRepo := database.NewModelRepository(db)
	datasetRepo := database.NewDatasetRepository(db)
	coreClient, err := coreclient.NewCoreClient(context.Background(), cfg.Core)
	if err != nil {
		log.Fatalf("初始化 Core Client 失败: %v", err)
	}
	defer coreClient.Close()
	if err := coreClient.HealthCheck(context.Background()); err != nil {
		log.Fatalf("Core HealthCheck 失败: %v", err)
	}

	evalService := application.NewEvalService(evalRepo, evalResultRepo, modelRepo, datasetRepo, coreClient, cfg.Eval)
	modelService := application.NewModelService(modelRepo)
	systemService := application.NewSystemService(coreClient)
	datasetService := application.NewDatasetService(datasetRepo, cfg.Dataset)
	datasetService.SetCoreClient(&coreclient.DatasetCoreClientAdapter{Client: coreClient})
	datasetService.SyncOnStartup(context.Background())

	router := server.NewRouter(server.Services{
		Auth:    authService,
		Eval:    evalService,
		Model:   modelService,
		System:  systemService,
		Dataset: datasetService,
	})

	httpServer := &http.Server{
		Addr:         cfg.Server.Address(),
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout(),
		WriteTimeout: cfg.Server.WriteTimeout(),
	}

	log.Printf("Go Backend 已启动，address=%s", cfg.Server.Address())
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("启动 HTTP 服务失败: %v", err)
	}
}
