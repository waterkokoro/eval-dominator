# Scripts 说明

`scripts/` 用于放置本地开发、初始化、联调和构建脚本。

## 后续脚本规划

- `init_db.sh`：初始化 SQLite 数据库。
- `start_core.sh`：启动 Python Core。
- `start_backend.sh`：启动 Go 后端。
- `start_frontend.sh`：启动 Vue 前端。
- `dev_all.sh`：本地联调时按顺序启动全部服务。

脚本命名使用英文小写下划线。

## 约束

- 脚本不得写入真实 API Key。
- 脚本读取本地配置文件，不硬编码端口和路径。
- 需要输出清晰的中文提示。
