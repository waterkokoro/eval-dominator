# HTTP 接口文档

## 调用方向

`Frontend -> Go backend`

## 通用规则

- Base URL 由前端环境变量 `VUE_APP_API_BASE_URL` 配置。
- 除登录接口外，所有接口使用 `Authorization: Bearer <token>`。
- 请求和响应字段使用 `camelCase`。
- API Key 返回前端时必须脱敏。

## POST /auth/login

用途：账号密码登录，返回 JWT。

请求：

```json
{
  "username": "admin",
  "password": "password"
}
```

响应：

```json
{
  "token": "jwt-token",
  "expiresAt": "2026-04-28T12:00:00+08:00"
}
```

## POST /eval/tasks

用途：创建 EvalTask。

请求：

两种二选一：手动填写模型 (provider/modelName/baseUrl/apiKey) 或使用预设模型 (modelPresetId)。

```json
{
  "taskName": "可选，列表搜索用",
  "modelPresetId": 1,
  "datasetType": "opencompass_demo",
  "datasetName": "demo_gsm8k_chat_gen",
  "params": {}
}
```

或：

```json
{
  "taskName": "可选",
  "provider": "openai-compatible",
  "modelName": "qwen-plus",
  "displayName": "我的 Qwen",
  "version": "v1",
  "baseUrl": "https://dashscope.aliyuncs.com/compatible-mode/v1",
  "apiKey": "sk-xxx",
  "saveModel": true,
  "datasetType": "opencompass_demo",
  "datasetName": "demo_gsm8k_chat_gen",
  "params": { "temperature": "0.0" }
}
```

- `taskName`：可选，最多 200 字。
- `saveModel`：手动模式下若为 true，会把当次填写的模型存为预设。
- `params`：透传到 OpenCompass `OpenAISDK`，可覆盖 `temperature`、`max_out_len`、`max_seq_len`、`batch_size`、`max_workers`、`query_per_second`。

响应：

```json
{
  "evalTaskId": "ev-1a2b3c4d5e",
  "status": "pending"
}
```

任务 ID 为短 ID（`ev-` 前缀 + 10 位十六进制，约 13 字符）。

## GET /eval/tasks/{evalTaskId}

用途：查询 EvalTask 状态。

响应：

```json
{
  "evalTaskId": "ev-1a2b3c4d5e",
  "taskName": "我的实验",
  "status": "running",
  "errorMessage": ""
}
```

## GET /eval/tasks/{evalTaskId}/result

用途：查询 EvalResult。

响应：

```json
{
  "evalTaskId": "eval-task-id",
  "metrics": [
    {
      "name": "accuracy",
      "value": 0.95,
      "displayName": "Accuracy"
    }
  ],
  "rawResultPath": "../runtime/outputs/eval-task-id/raw.json",
  "reportPath": "../runtime/outputs/eval-task-id/report.md",
  "logPath": "../runtime/logs/core/eval-task-id.log"
}
```

## GET /eval/tasks

用途：任务历史列表，一期增强实现。

请求 query：

- `page`：默认 1
- `pageSize`：默认 10
- `status`：逗号分隔，对应任务状态枚举
- `search`：任务显示名称、任务 ID 模糊匹配（与下方 `keyword` 可同时使用）
- `keyword`：模型名 / Provider 模糊匹配
- `datasetType`：精确匹配数据集类型
- `createdFrom`、`createdTo`：创建日期，格式 `YYYY-MM-DD`（按本地时区，含起止日全天）

响应：

```json
{
  "items": [
    {
      "evalTaskId": "ev-1a2b3c4d5e",
      "taskName": "我的对比实验",
      "modelProvider": "openai-compatible",
      "modelName": "demo-model",
      "modelBaseUrl": "https://api.example.com/v1",
      "datasetType": "opencompass_demo",
      "datasetName": "demo",
      "status": "succeeded",
      "createdAt": "2026-04-27 12:00:00",
      "startedAt": "2026-04-27 12:00:01",
      "finishedAt": "2026-04-27 12:00:30",
      "errorCode": "",
      "errorMessage": ""
    }
  ],
  "total": 1,
  "page": 1,
  "pageSize": 10
}
```

## POST /eval/tasks/{evalTaskId}/cancel

终止运行中的任务（状态置为 `cancelled`，Core 端 SIGTERM/SIGKILL OpenCompass 子进程组）。
对处于终态的任务返回 400。

```json
{ "ok": true }
```

## GET /eval/tasks/{evalTaskId}/log

任务日志查看，query `tail`（默认 200）。后端会在 `runtime/outputs/<task>/...` 下挑「最近更新」的 OpenCompass log（顶层 opencompass.log 或最新的子集 infer .out）。

```json
{ "content": "2026-04-27 12:00:00 INFO ...", "tail": 200 }
```

## GET /eval/tasks/{evalTaskId}/artifacts/preview?path=...

文本类产物在线预览（最多 512KB），路径强制限制在 runtime/ 输出目录内、且与该任务相关。

```json
{
  "path": "/abs/path/to/file",
  "relativePath": "<taskId>/.../summary.csv",
  "size": 12345,
  "isText": true,
  "truncated": false,
  "content": "...",
  "contentType": "text/csv; charset=utf-8"
}
```

## GET /eval/tasks/{evalTaskId}/artifacts/download?path=...

直接以 `Content-Disposition: attachment` 流式下载，路径校验同上。

## /models CRUD（模型预设）

- `GET /models`
- `POST /models`：新增
- `PUT /models/:id`：更新（apiKey 为空时保留原值）
- `DELETE /models/:id`

列表项已脱敏：

```json
{
  "id": 1,
  "provider": "openai-compatible",
  "modelName": "qwen-plus",
  "displayName": "我的 Qwen",
  "version": "v1",
  "baseUrl": "https://api.example.com/v1",
  "maskedKey": "sk-9adb****4bb7",
  "createdAt": "2026-04-27 12:00:00"
}
```

## /datasets CRUD

- `GET /datasets`：列表（参数 `enabled=true` 仅启用项）
- `POST /datasets`：新增自定义数据集
- `PUT /datasets/:id`：更新
- `PATCH /datasets/:id/enabled`：启用/禁用
- `DELETE /datasets/:id`：删除（builtin 数据集禁止删除）
- `POST /datasets/sync`：扫描 OpenCompass demo 配置，幂等同步内置项

```json
{
  "id": 1,
  "code": "demo_gsm8k_chat_gen",
  "displayName": "GSM8K (Chat · GEN) · Demo",
  "type": "opencompass_demo",
  "source": "builtin",
  "sampleCount": 64,
  "enabled": true,
  "inferenceMode": "gen"
}
```

## GET /system/health

聚合 backend + Core 健康状态：

```json
{
  "backend": { "ok": true, "message": "ok" },
  "core": { "ok": true, "message": "ok", "version": "0.1.0" }
}
```

## POST /auth/logout / GET /auth/me

`logout` 清前端登录态（服务端无 session）；`me` 返回 `{"userId":1,"username":"admin"}`。

## POST /auth/change-password

修改当前登录用户的密码。需在 Authorization 头里带有效 JWT。

```json
{ "oldPassword": "admin123", "newPassword": "your-new-password" }
```

- `newPassword` 至少 6 位、且不能与 `oldPassword` 相同。
- 旧密码错误返回 400。
- 成功返回 `{ "ok": true }`。Token 不会自动失效；需要立刻强制下线请前端清掉本地 token。

## 错误响应

```json
{
  "code": "INVALID_ARGUMENT",
  "message": "参数错误"
}
```

常见错误 code：`INVALID_ARGUMENT` / `EVAL_TASK_NOT_FOUND` / `CREATE_EVAL_TASK_FAILED` / `CANCEL_TASK_FAILED` / `PREVIEW_ARTIFACT_FAILED` / `DOWNLOAD_ARTIFACT_FAILED` / `LIST_EVAL_TASK_FAILED`。
