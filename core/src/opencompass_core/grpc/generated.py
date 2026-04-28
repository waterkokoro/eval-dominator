from __future__ import annotations

import sys
from pathlib import Path


def ensure_generated_path() -> None:
    # grpcio-tools 生成的 pb2_grpc 文件默认使用同目录绝对导入，这里补充路径保证可导入。
    gen_dir = Path(__file__).resolve().parent / "gen"
    gen_dir_str = str(gen_dir)
    if gen_dir_str not in sys.path:
        sys.path.insert(0, gen_dir_str)
