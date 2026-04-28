from __future__ import annotations

import logging
from pathlib import Path


def setup_logging(level: str, log_dir: Path) -> None:
    log_dir.mkdir(parents=True, exist_ok=True)
    log_level = getattr(logging, level.upper(), logging.INFO)
    log_file = log_dir / "core.log"

    logging.basicConfig(
        level=log_level,
        format="%(asctime)s %(levelname)s [%(name)s] %(message)s",
        handlers=[
            logging.StreamHandler(),
            logging.FileHandler(log_file, encoding="utf-8"),
        ],
    )
