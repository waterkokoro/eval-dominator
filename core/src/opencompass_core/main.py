from __future__ import annotations

import argparse

from opencompass_core.config import load_settings
from opencompass_core.server import serve
from opencompass_core.utils import setup_logging


def main() -> None:
    parser = argparse.ArgumentParser(description="OpenCompass Core gRPC Server")
    parser.add_argument("--config", default="config/config.yaml", help="Core 配置文件路径")
    args = parser.parse_args()

    settings = load_settings(args.config)
    setup_logging(settings.log.level, settings.runtime.log_dir)
    serve(settings)


if __name__ == "__main__":
    main()
