# Frontend 工程说明

## 技术栈

- Vue2
- ElementUI
- Axios
- Vue Router

## 职责边界

前端只负责页面交互和 HTTP 调用：

- 登录
- 提交评测任务
- 查询任务状态 / 历史列表
- 展示评测结果（指标、产物、日志）
- 管理 API Key（脱敏展示，不保存明文）
- 数据集中心、系统健康状态查看

前端不直接调用 Python Core，不保存 API Key 明文。

## 信息架构

```text
LoginView                      （独立页面，无 Layout）
└── AppLayout                  （AppHeader 顶栏 + AppSidebar 侧边栏 + Main）
    ├── 评测中心
    │   ├── 任务列表           /eval/tasks
    │   ├── 提交评测           /eval/submit
    │   └── 任务详情           /eval/tasks/:evalTaskId
    ├── 模型管理
    │   └── 模型预设           /models
    ├── 数据集
    │   └── 数据集中心          /datasets
    └── 系统
        └── 关于                /about
```

## 目录约定

- `src/api/`：HTTP API 封装（`auth` / `eval-task` / `model` / `dataset` / `artifact` / `system`）。
- `src/router/`：路由表与登录态拦截。
- `src/layout/`：`AppLayout` / `AppSidebar` / `AppHeader`。
- `src/views/`：页面级组件，按功能拆子目录（`eval/`、`model/`、`dataset/`、`system/`）。
- `src/components/`：通用展示组件（`StatusTag`、`PageHeader`、`EmptyState`、`MetricsTable`、`ArtifactList`、`LogViewer`、`KeyValueEditor`）。
- `src/store/`：基于 `Vue.observable` 的轻量状态（`user`、`app`），不引入 Vuex。
- `src/utils/`：请求、token 等工具。
- `src/constants/`：任务状态、数据集类型等常量映射。

## 配置

配置文件：

- `.env.development.example`
- `.env.production.example`

可配置项：

- `VUE_APP_API_BASE_URL`
- `VUE_APP_REQUEST_TIMEOUT`
- `VUE_APP_TOKEN_STORAGE_KEY`

## 命名规范

- Vue 组件文件使用 `PascalCase.vue`，例如 `EvalForm.vue`。
- 目录名使用小写短横线，例如 `eval-task`。
- 变量和方法使用 `camelCase`。
- 评测相关命名统一使用 `Eval`、`EvalTask`、`EvalResult`。

## 代码约束

- API 请求必须统一放在 `src/api/`。
- 页面组件不直接拼接后端 URL。
- 任务状态文案/颜色必须来自 `constants/eval-status.js`，并通过 `StatusTag` 渲染。
- 敏感字段不写入 localStorage。
- 中文注释只解释复杂交互和业务意图。
- HTTP 拦截器：
  - 默认 `Message.error` 全局兜底；如需静默处理（例如未就绪接口、空态判断），调用时传 `{ silent: true }`。
  - 401 会触发 `eval-dominator:unauthorized` 事件，自动清 token 并跳登录。

## 后端契约

所有后端 API 均已在 Go 后端实现，详见 `backend/docs/http接口文档.md`。如新增接口建议遵循以下约定：

- 字段统一 `camelCase`；时间统一 `YYYY-MM-DD HH:MM:SS` 字符串。
- 4xx 返回 `{ code, message }`，前端拦截器会 `Message.error(message)`，调用方可传 `{ silent: true }` 跳过全局提示。
- 401 由前端拦截器统一清 token + 派发 `eval-dominator:unauthorized` 事件回登录页。
