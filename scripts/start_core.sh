#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT_DIR}/core"
VENV_PYTHON="${ROOT_DIR}/core/.venv/bin/python"

if [ ! -f "config/config.yaml" ]; then
  cp "config/config.example.yaml" "config/config.yaml"
  echo "已创建 core/config/config.yaml，请按需修改 OpenCompass 路径和运行目录。"
fi

if [ ! -x "${VENV_PYTHON}" ]; then
  echo "未找到 core/.venv，请先执行: ./scripts/init_core_venv.sh"
  exit 1
fi

"${VENV_PYTHON}" scripts/generate_proto.py

# 把 venv/bin 注入 PATH，确保 opencompass 等子进程可以被 shutil.which 命中。
export PATH="$(dirname "${VENV_PYTHON}"):${PATH}"
# 关键：拉 OpenCompass 时是 fork+exec，子进程立刻被 exec 替换，不需要 Python gRPC。
# 开 GRPC_ENABLE_FORK_SUPPORT=1 会触发 fork handler 在子进程 exec 前清理 gRPC 状态，
# 在 macOS 上经常失败并 abort 子进程（表现：opencompass.log 0 字节，错误为
# "Failed to shutdown gRPC Core after fork()" / OPENCOMPASS_NON_ZERO_EXIT）。
# 因此默认禁用，配合 GRPC_VERBOSITY=error 压住父进程刚 fork 后 gRPC C 核心的 INFO 噪音。
export GRPC_ENABLE_FORK_SUPPORT="${GRPC_ENABLE_FORK_SUPPORT:-0}"
export GRPC_VERBOSITY="${GRPC_VERBOSITY:-error}"
# OpenCompass 会基于 model.path 去 huggingface.co 探测 tokenizer/config.json：
# 远程 API 模型（如 qwen3-plus）这条路径并非真实 HF 仓库，会触发 HTTP 429，
# 每个子集白白等 50s 后重试。这里强制 huggingface_hub / transformers 走离线模式，
# 让其直接落到本地 fallback tokenizer，不再发 HEAD 到 HF。
export HF_HUB_OFFLINE="${HF_HUB_OFFLINE:-1}"
export TRANSFORMERS_OFFLINE="${TRANSFORMERS_OFFLINE:-1}"

PYTHONPATH=src "${VENV_PYTHON}" -m opencompass_core --config config/config.yaml
