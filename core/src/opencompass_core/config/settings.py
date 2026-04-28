from __future__ import annotations

from dataclasses import dataclass
from pathlib import Path
from typing import Any

import yaml


@dataclass(frozen=True)
class ServerSettings:
    host: str
    port: int
    max_workers: int


@dataclass(frozen=True)
class OpenCompassSettings:
    source_dir: Path
    dataset_dir: Path
    default_dataset: str
    default_model_type: str


@dataclass(frozen=True)
class RuntimeSettings:
    work_dir: Path
    log_dir: Path
    timeout_seconds: int
    keep_raw_outputs: bool


@dataclass(frozen=True)
class LogSettings:
    level: str


@dataclass(frozen=True)
class CoreSettings:
    server: ServerSettings
    opencompass: OpenCompassSettings
    runtime: RuntimeSettings
    log: LogSettings

    @property
    def grpc_address(self) -> str:
        return f"{self.server.host}:{self.server.port}"


def load_settings(config_path: str | Path) -> CoreSettings:
    path = Path(config_path).expanduser().resolve()
    with path.open("r", encoding="utf-8") as config_file:
        raw = yaml.safe_load(config_file) or {}

    base_dir = path.parent.parent
    settings = CoreSettings(
        server=ServerSettings(
            host=_get(raw, "server.host", "127.0.0.1"),
            port=int(_get(raw, "server.port", 50051)),
            max_workers=int(_get(raw, "server.max_workers", 4)),
        ),
        opencompass=OpenCompassSettings(
            source_dir=_resolve_path(base_dir, _get(raw, "opencompass.source_dir", "../resources/opencompass")),
            dataset_dir=_resolve_path(base_dir, _get(raw, "opencompass.dataset_dir", "../resources/opencompass-data")),
            default_dataset=str(_get(raw, "opencompass.default_dataset", "demo")),
            default_model_type=str(_get(raw, "opencompass.default_model_type", "remote_api")),
        ),
        runtime=RuntimeSettings(
            work_dir=_resolve_path(base_dir, _get(raw, "runtime.work_dir", "../runtime/outputs")),
            log_dir=_resolve_path(base_dir, _get(raw, "runtime.log_dir", "../runtime/logs/core")),
            timeout_seconds=int(_get(raw, "runtime.timeout_seconds", 3600)),
            keep_raw_outputs=bool(_get(raw, "runtime.keep_raw_outputs", True)),
        ),
        log=LogSettings(level=str(_get(raw, "log.level", "info"))),
    )

    settings.runtime.work_dir.mkdir(parents=True, exist_ok=True)
    settings.runtime.log_dir.mkdir(parents=True, exist_ok=True)
    return settings


def _get(raw: dict[str, Any], dotted_key: str, default: Any) -> Any:
    current: Any = raw
    for key in dotted_key.split("."):
        if not isinstance(current, dict) or key not in current:
            return default
        current = current[key]
    return current


def _resolve_path(base_dir: Path, value: str) -> Path:
    path = Path(value).expanduser()
    if path.is_absolute():
        return path.resolve()
    return (base_dir / path).resolve()
