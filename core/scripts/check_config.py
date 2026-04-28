from __future__ import annotations

import argparse

from opencompass_core.config import load_settings
from opencompass_core.utils import setup_logging


def main() -> None:
    parser = argparse.ArgumentParser(description="检查 Core 配置是否可加载")
    parser.add_argument("--config", default="config/config.example.yaml", help="Core 配置文件路径")
    args = parser.parse_args()

    settings = load_settings(args.config)
    setup_logging(settings.log.level, settings.runtime.log_dir)
    print(f"Core 配置加载成功，gRPC 地址: {settings.grpc_address}")
    print(f"运行输出目录: {settings.runtime.work_dir}")


if __name__ == "__main__":
    main()
