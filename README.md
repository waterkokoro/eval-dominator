<p align="center">
  <img src="./eval-dominator.svg" alt="Eval Dominator" width="480"/>
</p>

<h1 align="center">Eval Dominator</h1>

<p align="center">
  一个轻量、本地优先、面向 OpenAI 兼容接口的大模型评测平台<br/>
  把 <a href="https://github.com/open-compass/opencompass">OpenCompass</a> 的能力包装成一个能用浏览器跑的小工具
</p>

<p align="center">
  <img src="https://img.shields.io/badge/version-v0.1.0--mvp-blue" alt="version"/>
  <img src="https://img.shields.io/badge/status-MVP%20%C2%B7%20iterating-orange" alt="status"/>
  <a href="https://github.com/open-compass/opencompass"><img src="https://img.shields.io/badge/OpenCompass-0.5.2-2c3e50" alt="OpenCompass 0.5.2"/></a>
  <a href="./LICENSE"><img src="https://img.shields.io/badge/License-MIT-green" alt="license"/></a>
  <br/>
  <img src="https://img.shields.io/badge/Go-1.21%2B-00ADD8?logo=go&logoColor=white" alt="Go"/>
  <img src="https://img.shields.io/badge/Python-3.10-3776AB?logo=python&logoColor=white" alt="Python"/>
  <img src="https://img.shields.io/badge/Node.js-18%2B-339933?logo=node.js&logoColor=white" alt="Node"/>
  <img src="https://img.shields.io/badge/Vue-2.x-4FC08D?logo=vue.js&logoColor=white" alt="Vue"/>
  <img src="https://img.shields.io/badge/Backend-Gin-00ACD7" alt="Gin"/>
  <img src="https://img.shields.io/badge/Storage-SQLite-003B57?logo=sqlite&logoColor=white" alt="SQLite"/>
  <img src="https://img.shields.io/badge/Transport-gRPC-4D4D4D?logo=grpc&logoColor=white" alt="gRPC"/>
  <img src="https://img.shields.io/badge/UI-ElementUI-409EFF" alt="Element UI"/>
</p>

<p align="center">
  <a href="./README.md">中文</a> · <a href="./README.en.md">English</a>
</p>

> **状态：MVP（v0.1.0-mvp），迭代中。** 目前只在本机单用户场景验证，不建议直接公网部署。

## 它是什么

把"评测一个 LLM"的流程拆成三层，每一层之间都有稳定的契约：

- **Frontend** (Vue2 + ElementUI)：登录、提交评测、跟踪进度、看指标和产物。
- **Backend** (Go + Gin + SQLite)：账号、任务编排与持久化、对外暴露 REST API。
- **Core** (Python + gRPC + OpenCompass)：实际驱动 OpenCompass 跑模型评测，被 Backend 通过 gRPC 调用。

```mermaid
flowchart LR
    UI["浏览器<br/>Vue 2 + ElementUI"]
    BE["Go Backend :8080<br/>Gin + JWT"]
    Core["Python Core :50051<br/>gRPC EvalService"]
    OC["OpenCompass CLI"]
    DB[("SQLite<br/>users / models / datasets / tasks")]
    FS["runtime/<br/>summary · log · predictions"]

    UI -->|HTTP| BE
    BE <--> DB
    BE -->|gRPC| Core
    Core -->|subprocess| OC
    OC -->|写入产物| FS
    BE -.->|预览 / 下载| FS
```

## 界面预览

<table>
  <tr>
    <td width="33%" align="center">
      <a href="./eval-img/zh/task-list.png"><img src="./eval-img/zh/task-list.png" alt="任务列表" width="100%"/></a>
      <br/><sub><b>任务列表</b><br/>创建 · 查询 · 筛选</sub>
    </td>
    <td width="33%" align="center">
      <a href="./eval-img/zh/task-detail01.png"><img src="./eval-img/zh/task-detail01.png" alt="任务详情" width="100%"/></a>
      <br/><sub><b>任务详情</b><br/>阶段进度 · 指标 · 产物</sub>
    </td>
    <td width="33%" align="center">
      <a href="./eval-img/zh/task-detail-log.png"><img src="./eval-img/zh/task-detail-log.png" alt="实时日志" width="100%"/></a>
      <br/><sub><b>实时日志</b><br/>自动跟最新 infer 子集</sub>
    </td>
  </tr>
  <tr>
    <td width="33%" align="center">
      <a href="./eval-img/zh/submit.png"><img src="./eval-img/zh/submit.png" alt="提交评测 · 模型" width="100%"/></a>
      <br/><sub><b>提交评测 · 模型</b><br/>API / 本地 · 预设</sub>
    </td>
    <td width="33%" align="center">
      <a href="./eval-img/zh/submit02.png"><img src="./eval-img/zh/submit02.png" alt="提交评测 · 数据集" width="100%"/></a>
      <br/><sub><b>提交评测 · 数据集</b><br/>GEN / PPL · 运行参数</sub>
    </td>
    <td width="33%" align="center">
      <a href="./eval-img/zh/model.png"><img src="./eval-img/zh/model.png" alt="模型管理" width="100%"/></a>
      <br/><sub><b>模型管理</b><br/>OpenAI 兼容预设 · 脱敏 Key</sub>
    </td>
  </tr>
</table>

## 主要特性（MVP）

- ✅ 账号 + 密码（bcrypt） + JWT 登录；首次启动 seed 一个默认账号 `admin / admin123`，登录后请立刻通过 `POST /auth/change-password` 修改
- ✅ OpenAI 兼容远程模型评测（DashScope / OpenAI / DeepSeek / 自部署 vLLM 等）
- ✅ 模型预设（保存 / 复用 / 脱敏回显 API Key）
- ✅ 数据集中心：自动同步 OpenCompass demo（`demo_gsm8k`、`demo_math`、`demo_cmmlu` 等），按 `gen` / `ppl` 模式打标签并做兼容性拦截
- ✅ 任务列表（搜索 / 时间筛选 / 多状态过滤）
- ✅ 任务详情：阶段进度、指标可视化（自动百分比识别）、产物在线预览 & 下载、实时日志（自动跟最新 infer 子集）
- ✅ 任务终止（SIGTERM/SIGKILL 整个 OpenCompass 进程组）
- ✅ 前端多语言（中文 / English）：基于 vue-i18n + ElementUI 内置 locale，文案集中在 `frontend/src/locales/{zh-CN,en-US}/*.json`，新增语种只需注册一项即可
- 🚧 评测角色 / 模板（多模型 + judge 编排，规划中，详见 [`md/评测角色与模板规划-2026-04-27-v1.md`](./md/评测角色与模板规划-2026-04-27-v1.md)）
- 🚧 本地 HuggingFace 模型 + PPL 数据集
- 🚧 用户系统、权限、多人协作

## 基于的 OpenCompass 版本

`opencompass==0.5.2`（在 `core/.venv` 中安装；不与系统 Python 共享）。

> 我们与 OpenCompass 的耦合点很薄：只生成它能消费的 mmengine `.py` config，再以 subprocess 调起其 CLI，结果通过解析 `summary/*.csv` 拿回。理论上随 OpenCompass 升级 minor 版本可平滑替换；如果上游改了 demo 数据集变量命名规则可能需要更新 `core/src/opencompass_core/adapter/opencompass_adapter.py` 里的两个正则。

## Quick Start

### 0. 前置依赖

| 依赖 | 版本 | 用途 |
| --- | --- | --- |
| Python | **3.10**（不要用 3.11+，OpenCompass 0.5.x 还未官方支持） | Core / OpenCompass |
| Go | 1.21+ | Backend |
| Node.js | 18+ | Frontend |
| protoc 工具链 | 由 `scripts/generate_proto.sh` 自动安装 | 生成 gRPC 代码 |

### 1. 克隆 + 拉一份本地配置

```bash
git clone <your-fork-url> eval-dominator
cd eval-dominator

cp backend/config/config.example.yaml backend/config/config.yaml
cp core/config/config.example.yaml    core/config/config.yaml
cp frontend/.env.development.example  frontend/.env.development  # 如有需要
```

> 一定要把 `backend/config/config.yaml` 里的 `jwt.secret` 改成自己的值，不要保留默认占位符。

### 2. 初始化 Python venv（含 OpenCompass）

```bash
./scripts/init_core_venv.sh
```

这一步会创建 `core/.venv`、安装 OpenCompass 0.5.2 与运行时依赖。第一次跑会比较慢。

### 3. 生成 gRPC 代码（首次或修改 proto 后）

```bash
./scripts/generate_proto.sh
```

会自动 `go install` 安装 `buf` / `protoc-gen-go` / `protoc-gen-go-grpc`，并用 venv 里的 `grpcio-tools` 生成 Python 代码。

### 4. 三个终端跑起来

```bash
# 终端 1
./scripts/start_core.sh        # gRPC :50051

# 终端 2
./scripts/start_backend.sh     # HTTP :8080

# 终端 3
./scripts/start_frontend.sh    # Vue dev server :8081/8080
```

打开浏览器，使用默认账号 **`admin / admin123`** 登录；首次登录后建议立即修改：

```bash
curl -X POST http://127.0.0.1:8080/api/auth/change-password \
  -H "Authorization: Bearer <你的 token>" \
  -H "Content-Type: application/json" \
  -d '{"oldPassword":"admin123","newPassword":"<你的新密码>"}'
```

或同步修改 `backend/config/config.yaml::auth.default_admin_password` 后重启（首次成功登录前才会"按配置 seed"，已存在用户不会被覆盖）。

### 5. 跑你的第一次评测

1. 「模型管理」→ 新增一个预设：例如 DashScope 的 `qwen-plus`，base url = `https://dashscope.aliyuncs.com/compatible-mode/v1`，api key 填你自己的 `sk-...`。
2. 「数据集」页面会列出 OpenCompass 内置 demo；建议先用 **`demo_gsm8k_chat_gen`**（4 题，30 秒左右出结果）跑通流程；不要一上来就 `demo_cmmlu_chat_gen`（67 个子学科 ≈ 30 分钟）。
3. 「提交评测」→ 选预设模型 + 数据集 → 创建。

任务详情页能看到：状态机进度条、实时日志（自动跟最新 infer log）、指标表格（自动判断百分比并画进度条）、产物预览/下载。

## 项目结构

```
.
├── frontend/             # Vue2 + ElementUI
├── backend/              # Go + Gin
│   ├── cmd/server/
│   ├── internal/{config,domain,application,handler,middleware,server,infrastructure}
│   ├── migrations/       # SQLite 初始化脚本
│   └── docs/             # http接口文档.md / 数据库设计.md
├── core/                 # Python gRPC 服务
│   ├── src/opencompass_core/
│   ├── config/           # config.example.yaml
│   └── scripts/          # generate_proto.py
├── proto/                # 协议文件（gRPC 契约）
├── runtime/              # 本地 SQLite + 评测产物（已 gitignored）
├── md/                   # 中文设计文档（架构 / 实施 / 规范 / 评测角色规划）
├── scripts/              # 启动 / 初始化 / generate proto
└── deploy/               # 本地部署说明
```

## 文档索引

- 架构：[`md/整体架构说明-2026-04-27-v1.md`](./md/整体架构说明-2026-04-27-v1.md)
- 实施步骤：[`md/实施步骤-2026-04-27-v1.md`](./md/实施步骤-2026-04-27-v1.md)
- 命名/接口规范：[`md/命名与接口规范-2026-04-27-v1.md`](./md/命名与接口规范-2026-04-27-v1.md)
- 评测角色 / 模板规划：[`md/评测角色与模板规划-2026-04-27-v1.md`](./md/评测角色与模板规划-2026-04-27-v1.md)
- HTTP 接口：[`backend/docs/http接口文档.md`](./backend/docs/http接口文档.md)
- gRPC 协议：[`proto/评测服务协议.md`](./proto/评测服务协议.md)
- 数据库：[`backend/docs/数据库设计.md`](./backend/docs/数据库设计.md)
- 本地部署：[`deploy/本地部署说明.md`](./deploy/本地部署说明.md)

## 运行环境优化点（MVP 期间踩过的坑，已固化进 scripts）

- `HF_HUB_OFFLINE=1` / `TRANSFORMERS_OFFLINE=1`：避免 OpenCompass 把远程 model 名当 HF 仓库去探测，否则每个子集会被 HF 限流停 50s。
- `GRPC_ENABLE_FORK_SUPPORT=0` / `GRPC_VERBOSITY=error`：Core 是 gRPC 进程，subprocess 拉 OpenCompass 时 fork+exec，开 fork support 会让子进程在 exec 前 abort。
- DashScope（qwen 兼容模式）严格校验 `temperature` 必须是 Float，因此生成的 OpenAISDK 配置默认 `temperature=0.0`（可在 task 的 `params` 覆盖）。
- 默认 `query_per_second=5`、`max_workers=8`，在大多数 OpenAI 兼容厂商里更合理。

## 安全提醒

- ⚠️ 默认 JWT secret 是占位符 `replace-with-local-secret`，**部署前必须改 `backend/config/config.yaml::jwt.secret`**（推荐 `openssl rand -hex 32`）。
- ⚠️ 默认 admin 密码 `admin123` 是 seed，方便本地开机即用；**禁止把这套默认值挂到公网**。
- ⚠️ API Key 在 SQLite 里**明文存储**，回显时脱敏。
- ⚠️ 产物预览/下载接口已限定路径必须落在 `runtime/` 输出目录内，但仍建议不要把 backend 直接挂到公网。

## 路线图（粗）

- [ ] 用户注册接口、多用户、空间隔离
- [ ] 评测角色 / 模板（详见规划文档）
- [ ] 本地 HuggingFace 模型 + PPL 数据集
- [ ] 多任务并发与资源调度
- [ ] 多用户、空间隔离

## 致谢与引用

本项目基于 [OpenCompass 0.5.2](https://github.com/open-compass/opencompass) 构建——感谢 OpenCompass 提供的稳定、丰富的评测工具链与数据集。

- 代码仓库：<https://github.com/open-compass/opencompass>
- 在线文档：<https://opencompass.readthedocs.io/>

如果你在研究 / 报告 / 文章中引用了本项目的评测结果，请同时引用 OpenCompass：

```bibtex
@misc{2023opencompass,
  title        = {OpenCompass: A Universal Evaluation Platform for Foundation Models},
  author       = {OpenCompass Contributors},
  howpublished = {\url{https://github.com/open-compass/opencompass}},
  year         = {2023}
}
```

## License

[MIT](./LICENSE)
