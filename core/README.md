# Core 工程说明

## 技术栈

- Python
- gRPC Server
- OpenCompass

## 职责边界

Python Core 是无状态能力服务：

- 不维护业务任务状态。
- 不连接 SQLite。
- 不做用户登录和权限。
- 不做前端字段适配。
- 只封装 OpenCompass 原子能力。

Go 后端负责流程编排、状态推进、结果入库和前端查询。

## 建议目录

后续初始化 Python 工程时建议使用：

- `src/opencompass_core/server/`：gRPC Server 启动与服务注册。
- `src/opencompass_core/service/`：协议入参出参转换。
- `src/opencompass_core/adapter/`：OpenCompass 适配层。
- `src/opencompass_core/runner/`：评测执行封装。
- `src/opencompass_core/parser/`：结果解析。
- `src/opencompass_core/config/`：配置加载。
- `src/opencompass_core/utils/`：日志、路径、错误处理。
- `tests/`：单元测试和最小能力测试。

## 原子能力

Core 优先提供最小维度能力：

- `HealthCheck`
- `ValidateEvalConfig`
- `BuildEvalConfig`
- `ExecuteEval`
- `ParseEvalResult`

除非多步封装能明显提升性能或维护性，否则不把流程编排封装到 Core。

## 配置

示例配置：

- `config/config.example.yaml`

真实本地配置：

- `config/config.yaml`

真实配置不得提交 Git。

## 本地启动

Core 必须使用独立虚拟环境运行。OpenCompass 当前依赖不适合直接安装在 Python 3.13 环境，建议使用 Python 3.10。

初始化虚拟环境：

```bash
./scripts/init_core_venv.sh
```

生成 gRPC 代码：

```bash
./scripts/generate_proto.sh
```

检查配置：

```bash
PYTHONPATH=src .venv/bin/python scripts/check_config.py --config config/config.example.yaml
```

启动服务：

```bash
./scripts/start_core.sh
```

## OpenCompass MVP

MVP 固定先支持：

- 远程 API 模型
- OpenCompass demo 数据集

后续需要能扩展：

- OpenCompass 标准数据集
- 自定义数据集
- 本地 HuggingFace 模型

## 返回数据

Core 对结果采取尽量全量返回：

- 指标列表
- 原始结果路径
- 报告路径
- 日志路径
- 产物列表
- 错误详情
- 补充元数据

## 命名规范

- 文件名、模块名、函数名、变量名使用 `snake_case`。
- 类名使用 `PascalCase`。
- 常量使用 `UPPER_SNAKE_CASE`。
- 司南统一写作 `OpenCompass`。
- 评测相关命名统一使用 `Eval`、`EvalTask`、`EvalResult`。

## 注释规范

中文注释用于解释：

- OpenCompass 调用边界
- 配置映射
- 结果解析
- 异常处理
- 原子能力为什么这样拆分
