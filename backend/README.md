# Backend 工程说明

## 技术栈

- Go
- Gin
- SQLite
- gRPC Client
- JWT

## 职责边界

Go 后端是唯一业务编排服务：

- 用户登录和 JWT 鉴权
- EvalTask 创建与状态推进
- SQLite 持久化
- API Key 保存、脱敏和敏感日志规避
- 调用 Python Core 的 gRPC 原子能力
- 对前端提供 HTTP API

后端不直接实现 OpenCompass 评测逻辑。

## 分层规范

固定使用轻量 DDD 分层：

- `internal/handler/`：HTTP 入参绑定、鉴权上下文提取、响应返回。
- `internal/application/`：业务用例、任务编排、事务边界。
- `internal/domain/`：实体、枚举、状态流转、领域规则。
- `internal/infrastructure/`：SQLite、gRPC Client、配置、日志。
- `internal/config/`：配置结构和加载逻辑。
- `internal/middleware/`：JWT、日志、错误恢复等中间件。

## 配置

示例配置：

- `config/config.example.yaml`

真实本地配置：

- `config/config.yaml`

真实配置不得提交 Git。

## 数据库

初始化脚本：

- `migrations/001_init.sql`

核心表：

- `users`
- `api_keys`
- `eval_tasks`
- `eval_results`

## 任务状态

Go 后端维护全部业务状态：

- `pending`
- `validating`
- `building`
- `running`
- `parsing`
- `succeeded`
- `failed`
- `timeout`

Python Core 不维护这些状态。

## 性能与资源约束

- 数据库连接集中初始化并复用。
- gRPC Client 集中初始化并复用。
- 配置和日志实例集中初始化。
- 所有外部调用必须设置 `context timeout`。
- 不在 HTTP handler 中写业务逻辑。

## 命名规范

- 文件名使用小写下划线，例如 `eval_service.go`。
- 包名使用小写短名。
- 导出类型和方法使用 `PascalCase`。
- 非导出变量和方法使用 `camelCase`。
- JSON 字段使用 `camelCase`。
- 数据库字段使用 `snake_case`。
- 评测相关命名统一使用 `Eval`、`EvalTask`、`EvalResult`。

## 注释规范

中文注释用于解释：

- 状态流转
- 异步任务编排
- gRPC 调用边界
- 错误映射
- API Key 脱敏和敏感信息处理
