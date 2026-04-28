# Proto 说明

`proto/` 是 Go 后端和 Python Core 之间的唯一真实接口契约。所有 gRPC 方法、请求字段、响应字段和枚举都必须先在 `.proto` 文件中定义，再生成两侧代码。

## 固定规则

- 服务名统一使用 `EvalService`。
- 字段名使用 `snake_case`。
- message、service、enum 使用 `PascalCase`。
- 不定义同步跑完整个长任务的 `RunEvaluation` 大接口。
- Core 只暴露 OpenCompass 原子能力，业务任务状态由 Go 后端维护。
- 删除字段后不得复用字段编号。
- 修改 `.proto` 后必须同步更新 `评测服务协议.md`。

## 生成代码

标准生成方式：

```bash
./scripts/generate_proto.sh
```

生成方式不依赖 macOS 全局 `protoc`：

- Go 侧使用 `buf generate`，脚本会通过 `go install github.com/bufbuild/buf/cmd/buf@latest` 安装 `buf`。
- Go 侧插件使用 `protoc-gen-go` 和 `protoc-gen-go-grpc`，脚本会通过 `go install` 安装。
- Python 侧使用 pip 依赖 `grpcio-tools`，脚本会执行 `python3 -m pip install -r core/requirements.txt`。

相关配置：

- `buf.yaml`
- `buf.gen.yaml`

生成位置：

- Go：`backend/internal/infrastructure/grpc/gen/`
- Python：`core/src/opencompass_core/grpc/gen/`

## 当前协议文件

- `eval_service.proto`：Go 后端调用 Python Core 的 gRPC 契约。
