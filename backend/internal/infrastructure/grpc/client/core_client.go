package client

import (
	"context"
	"fmt"

	evalv1 "eval-dominator/backend/internal/infrastructure/grpc/gen"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"eval-dominator/backend/internal/config"
)

type CoreClient struct {
	conn   *grpc.ClientConn
	client evalv1.EvalServiceClient
	config config.CoreConfig
}

func NewCoreClient(ctx context.Context, cfg config.CoreConfig) (*CoreClient, error) {
	dialCtx, cancel := context.WithTimeout(ctx, cfg.ConnectTimeout())
	defer cancel()

	conn, err := grpc.DialContext(
		dialCtx,
		cfg.GRPCAddress,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, fmt.Errorf("连接 Python Core 失败: %w", err)
	}

	return &CoreClient{conn: conn, client: evalv1.NewEvalServiceClient(conn), config: cfg}, nil
}

func (c *CoreClient) Close() error {
	if c == nil || c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

func (c *CoreClient) HealthCheck(ctx context.Context) error {
	callCtx, cancel := context.WithTimeout(ctx, c.config.CallTimeout())
	defer cancel()

	resp, err := c.client.HealthCheck(callCtx, &evalv1.HealthCheckRequest{RequestId: "backend-health-check"})
	if err != nil {
		return fmt.Errorf("调用 Core HealthCheck 失败: %w", err)
	}
	if !resp.GetOk() {
		return fmt.Errorf("Core HealthCheck 返回不可用: %s", resp.GetMessage())
	}
	return nil
}

func (c *CoreClient) ValidateEvalConfig(ctx context.Context, config *evalv1.EvalConfig) error {
	callCtx, cancel := context.WithTimeout(ctx, c.config.CallTimeout())
	defer cancel()

	resp, err := c.client.ValidateEvalConfig(callCtx, &evalv1.ValidateEvalConfigRequest{
		RequestId: "validate-eval-config",
		Config:    config,
	})
	if err != nil {
		return fmt.Errorf("调用 Core ValidateEvalConfig 失败: %w", err)
	}
	if !resp.GetValid() {
		return fmt.Errorf("Core 配置校验失败: %s", formatValidationErrors(resp.GetErrors()))
	}
	return nil
}

func (c *CoreClient) BuildEvalConfig(ctx context.Context, config *evalv1.EvalConfig) (*evalv1.BuildEvalConfigResponse, error) {
	callCtx, cancel := context.WithTimeout(ctx, c.config.CallTimeout())
	defer cancel()

	resp, err := c.client.BuildEvalConfig(callCtx, &evalv1.BuildEvalConfigRequest{
		RequestId: "build-eval-config",
		Config:    config,
	})
	if err != nil {
		return nil, fmt.Errorf("调用 Core BuildEvalConfig 失败: %w", err)
	}
	if !resp.GetOk() {
		return nil, coreError(resp.GetError())
	}
	return resp, nil
}

func (c *CoreClient) ExecuteEval(ctx context.Context, evalTaskID string, config *evalv1.EvalConfig, configPath string, outputDir string) (*evalv1.ExecuteEvalResponse, error) {
	callCtx, cancel := context.WithTimeout(ctx, c.config.CallTimeout())
	defer cancel()

	resp, err := c.client.ExecuteEval(callCtx, &evalv1.ExecuteEvalRequest{
		RequestId:  "execute-eval",
		EvalTaskId: evalTaskID,
		Config:     config,
		ConfigPath: configPath,
		OutputDir:  outputDir,
	})
	if err != nil {
		return nil, fmt.Errorf("调用 Core ExecuteEval 失败: %w", err)
	}
	if !resp.GetOk() {
		return nil, coreError(resp.GetError())
	}
	return resp, nil
}

// CancelEval 让 Core 终止指定任务的 OpenCompass 子进程组。
// running=false 表示当时无运行中进程；此函数仅在 RPC 层面失败时返回 error。
func (c *CoreClient) CancelEval(ctx context.Context, evalTaskID string) (running bool, err error) {
	callCtx, cancel := context.WithTimeout(ctx, c.config.CallTimeout())
	defer cancel()

	resp, err := c.client.CancelEval(callCtx, &evalv1.CancelEvalRequest{
		RequestId:  "cancel-eval",
		EvalTaskId: evalTaskID,
	})
	if err != nil {
		return false, fmt.Errorf("调用 Core CancelEval 失败: %w", err)
	}
	return resp.GetRunning(), nil
}

func (c *CoreClient) ParseEvalResult(ctx context.Context, evalTaskID string, outputDir string) (*evalv1.EvalResult, error) {
	callCtx, cancel := context.WithTimeout(ctx, c.config.CallTimeout())
	defer cancel()

	resp, err := c.client.ParseEvalResult(callCtx, &evalv1.ParseEvalResultRequest{
		RequestId:  "parse-eval-result",
		EvalTaskId: evalTaskID,
		OutputDir:  outputDir,
	})
	if err != nil {
		return nil, fmt.Errorf("调用 Core ParseEvalResult 失败: %w", err)
	}
	if !resp.GetOk() {
		return nil, coreError(resp.GetError())
	}
	return resp.GetResult(), nil
}

func formatValidationErrors(errors []*evalv1.ValidationError) string {
	if len(errors) == 0 {
		return "未知错误"
	}
	message := ""
	for i, item := range errors {
		if i > 0 {
			message += "; "
		}
		message += item.GetField() + ": " + item.GetMessage()
	}
	return message
}

func coreError(err *evalv1.CoreError) error {
	if err == nil || err.GetMessage() == "" {
		return fmt.Errorf("Core 返回未知错误")
	}
	msg := err.GetMessage()
	if detail := err.GetDetail(); detail != "" {
		msg = msg + " | " + detail
	}
	if err.GetCode() == "" {
		return fmt.Errorf("%s", msg)
	}
	return fmt.Errorf("%s: %s", err.GetCode(), msg)
}
