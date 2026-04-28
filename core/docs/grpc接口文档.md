# Core gRPC 接口文档

## 调用方向

`Go backend -> Python Core`

## 真实契约

- `proto/eval_service.proto`

本文档只解释协议，不替代 `.proto` 文件。

## 接口列表

### HealthCheck

确认 Core 服务是否可用。

### ValidateEvalConfig

校验远程 API 模型、数据集、运行目录、超时等配置。

### BuildEvalConfig

根据标准化 `EvalConfig` 构建 OpenCompass 可执行配置。

### ExecuteEval

执行 OpenCompass 评测原子能力。

注意：该接口不代表业务任务生命周期，不维护状态。

### ParseEvalResult

解析 OpenCompass 输出结果，返回指标、产物路径和错误信息。

## 敏感信息

- `api_key` 只用于调用远程 API 模型。
- Core 不保存 API Key。
- Core 日志不得输出 API Key 明文。
