import os

# 须在首次 import grpc 之前设置环境变量：
# - 本进程一边跑 gRPC 服务、一边 subprocess 启 OpenCompass（fork+exec），
#   开 GRPC_ENABLE_FORK_SUPPORT=1 反而会触发 Python gRPC 的 fork handler，
#   在子进程 exec 之前尝试清理父进程的 gRPC 状态，常常失败并 abort 子进程
#   （表现：opencompass.log 0 字节、exit code 非 0、Core 日志看到
#    "Failed to shutdown gRPC Core after fork()"）。fork+exec 下子进程不会再用
#   Python gRPC，所以禁用 fork support 是更稳的选择，配合 GRPC_VERBOSITY=error
#   可以压住父进程刚 fork 后 gRPC C 核心打出的 INFO 噪音。
os.environ.setdefault("GRPC_ENABLE_FORK_SUPPORT", "0")
os.environ.setdefault("GRPC_VERBOSITY", "error")
# OpenCompass 会基于 model.path 去 huggingface.co 探测 tokenizer/config.json：
# 远程 API 模型（如 qwen3-plus）这条路径并非真实 HF 仓库，会触发 HTTP 429（每个子集白等 50s）。
# 这里强制走离线模式，让 huggingface_hub / transformers 直接落到本地 fallback tokenizer。
os.environ.setdefault("HF_HUB_OFFLINE", "1")
os.environ.setdefault("TRANSFORMERS_OFFLINE", "1")

from opencompass_core.main import main

if __name__ == "__main__":
    main()
