from __future__ import annotations

import csv
import json
import logging
import os
import re
import shutil
import signal
import subprocess
import sys
import threading
import time

# 非 Windows 下避免子进程继承 gRPC/其它无关 FD，减轻与 fork+exec 相关的 gRPC 告警。
def _subprocess_close_fds() -> bool:
    return os.name != "nt"
from dataclasses import dataclass
from pathlib import Path
from typing import Any

from opencompass_core.config import CoreSettings

# OpenCompass 0.5.x 内置 demo 数据集，每个 config 文件导出的变量名形如 *_datasets。
# 这里默认走 OpenCompass 安装目录下 configs/datasets/demo/<dataset>.py。
# demo 配置常见写法是 `with read_base(): from ..xxx import yyy_datasets`，因此匹配时不锁定行首。
_DATASETS_VAR_RE = re.compile(r"\b([a-zA-Z0-9_]+_datasets)\s*=\s*[\[(]")
_DEMO_IMPORT_RE = re.compile(r"from\s+\.{1,}[a-zA-Z0-9_.]+\s+import\s+([a-zA-Z0-9_]+_datasets)")

logger = logging.getLogger(__name__)


@dataclass(frozen=True)
class ValidationIssue:
    field: str
    message: str


@dataclass(frozen=True)
class ArtifactRef:
    type_name: str
    name: str
    path: str
    description: str


@dataclass(frozen=True)
class AdapterError:
    code: str
    message: str
    detail: str = ""


@dataclass(frozen=True)
class BuildResult:
    ok: bool
    config_path: str = ""
    output_dir: str = ""
    artifacts: tuple[ArtifactRef, ...] = ()
    error: AdapterError | None = None


@dataclass(frozen=True)
class ExecuteResult:
    ok: bool
    output_dir: str = ""
    artifacts: tuple[ArtifactRef, ...] = ()
    error: AdapterError | None = None


@dataclass(frozen=True)
class MetricValue:
    name: str
    value: float
    display_name: str = ""
    description: str = ""
    extra: dict[str, str] | None = None


@dataclass(frozen=True)
class ParsedResult:
    ok: bool
    metrics: tuple[MetricValue, ...] = ()
    artifacts: tuple[ArtifactRef, ...] = ()
    raw_result_path: str = ""
    report_path: str = ""
    log_path: str = ""
    metadata: dict[str, str] | None = None
    error: AdapterError | None = None


class OpenCompassAdapter:
    def __init__(self, settings: CoreSettings) -> None:
        self._settings = settings
        # task_id -> Popen，用于 CancelEval 时杀子进程组。
        self._running: dict[str, subprocess.Popen] = {}
        self._cancelled: set[str] = set()
        self._lock = threading.Lock()

    def validate_eval_config(self, config: Any) -> list[ValidationIssue]:
        issues: list[ValidationIssue] = []

        if not config.model.provider:
            issues.append(ValidationIssue("model.provider", "模型服务商不能为空"))
        if not config.model.model_name:
            issues.append(ValidationIssue("model.model_name", "模型名称不能为空"))
        if not config.model.api_key:
            issues.append(ValidationIssue("model.api_key", "API Key 不能为空"))
        if not config.dataset.name:
            issues.append(ValidationIssue("dataset.name", "数据集名称不能为空"))

        return issues

    def build_eval_config(self, config: Any) -> BuildResult:
        issues = self.validate_eval_config(config)
        if issues:
            return BuildResult(
                ok=False,
                error=AdapterError(
                    code="INVALID_EVAL_CONFIG",
                    message="评测配置校验失败",
                    detail="; ".join(f"{issue.field}: {issue.message}" for issue in issues),
                ),
            )

        output_dir = self._resolve_output_dir(config)
        output_dir.mkdir(parents=True, exist_ok=True)

        # 留一份 JSON 描述供排错，不参与执行。
        descriptor_path = output_dir / "opencompass_eval_config.json"
        descriptor_path.write_text(
            json.dumps(self._to_config_payload(config, output_dir), ensure_ascii=False, indent=2),
            encoding="utf-8",
        )

        # 生成 OpenCompass 真正消费的 mmengine Python 配置。
        try:
            config_path = self._write_mmengine_config(config, output_dir)
        except FileNotFoundError as exc:
            return BuildResult(
                ok=False,
                output_dir=str(output_dir),
                error=AdapterError(
                    code="DATASET_NOT_FOUND",
                    message="未找到对应数据集配置",
                    detail=str(exc),
                ),
            )

        return BuildResult(
            ok=True,
            config_path=str(config_path),
            output_dir=str(output_dir),
            artifacts=(
                ArtifactRef(
                    type_name="config",
                    name="opencompass_eval_config",
                    path=str(config_path),
                    description="Core 生成的 OpenCompass 评测配置（mmengine .py）",
                ),
                ArtifactRef(
                    type_name="config",
                    name="opencompass_eval_descriptor",
                    path=str(descriptor_path),
                    description="原始评测配置描述（不含明文 Key 的话需自行处理）",
                ),
            ),
        )

    def execute_eval(self, eval_task_id: str, config: Any, config_path: str, output_dir: str) -> ExecuteResult:
        target_output_dir = Path(output_dir).expanduser().resolve() if output_dir else self._resolve_output_dir(config)
        target_output_dir.mkdir(parents=True, exist_ok=True)

        opencompass_command = self._resolve_opencompass_command(config)
        if opencompass_command is None:
            return ExecuteResult(
                ok=False,
                output_dir=str(target_output_dir),
                error=AdapterError(
                    code="OPENCOMPASS_NOT_AVAILABLE",
                    message="未找到 OpenCompass 命令",
                    detail="请安装 opencompass，或在 extra_params.opencompass_command 中配置可执行命令",
                ),
            )

        command = self._build_command(opencompass_command, config, config_path, target_output_dir)
        log_path = target_output_dir / "opencompass.log"

        logger.info("开始执行 OpenCompass 评测，eval_task_id=%s, output_dir=%s", eval_task_id, target_output_dir)

        # 重置 cancel 标记（允许同一 task_id 重新跑）。
        with self._lock:
            self._cancelled.discard(eval_task_id)

        # 关键改造：用 Popen 启动并把它放进新的 process group，便于 cancel 时整组发信号。
        # OpenCompass 内部会再 fork 多个 worker 子进程（如 openicl_infer.py），
        # 仅 kill 顶层进程不够，必须 killpg。
        # cwd 选择：默认让 OpenCompass 的临时文件（./tmp、./icl_inference_output 等）
        # 写在任务的 output 目录下，避免污染源代码目录。如果用户显式配了 source_dir
        # （可能依赖那里的某些 OpenCompass 资源），就尊重该配置。
        if self._settings.opencompass.source_dir.exists():
            run_cwd = str(self._settings.opencompass.source_dir)
        else:
            run_cwd = str(target_output_dir)
        try:
            with log_path.open("w", encoding="utf-8") as log_file:
                proc = subprocess.Popen(
                    command,
                    cwd=run_cwd,
                    stdout=log_file,
                    stderr=subprocess.STDOUT,
                    text=True,
                    close_fds=_subprocess_close_fds(),
                    preexec_fn=os.setsid if os.name != "nt" else None,
                )
                with self._lock:
                    self._running[eval_task_id] = proc
                try:
                    completed_returncode = proc.wait(timeout=self._resolve_timeout(config))
                finally:
                    with self._lock:
                        self._running.pop(eval_task_id, None)

            class _C:  # 兼容下游 completed.returncode 调用
                def __init__(self, rc): self.returncode = rc
            completed = _C(completed_returncode)
        except subprocess.TimeoutExpired as exc:
            return ExecuteResult(
                ok=False,
                output_dir=str(target_output_dir),
                artifacts=self._collect_artifacts(target_output_dir),
                error=AdapterError(code="OPENCOMPASS_TIMEOUT", message="OpenCompass 执行超时", detail=str(exc)),
            )
        except OSError as exc:
            return ExecuteResult(
                ok=False,
                output_dir=str(target_output_dir),
                artifacts=self._collect_artifacts(target_output_dir),
                error=AdapterError(code="OPENCOMPASS_EXEC_ERROR", message="OpenCompass 执行失败", detail=str(exc)),
            )

        with self._lock:
            cancelled = eval_task_id in self._cancelled
        if cancelled:
            return ExecuteResult(
                ok=False,
                output_dir=str(target_output_dir),
                artifacts=self._collect_artifacts(target_output_dir),
                error=AdapterError(
                    code="OPENCOMPASS_CANCELLED",
                    message="评测已被用户终止",
                    detail=f"exit_code={completed.returncode}",
                ),
            )

        if completed.returncode != 0:
            # 把 log 末尾贴一段进 detail；如果 log 为空（典型于子进程在 import 阶段就崩、
            # 来不及写 stdout，例如 GRPC fork handler abort、PYTHONPATH 缺失等），
            # 在 message 里直接写明，避免上层只看到 "返回非零退出码" 这种没诊断价值的错。
            tail_text = self._tail_log(log_path, max_bytes=4096)
            if not tail_text:
                message = (
                    "OpenCompass 启动后立刻退出且没有任何输出，常见原因："
                    "GRPC fork handler abort（请确保 GRPC_ENABLE_FORK_SUPPORT=0）、"
                    "依赖缺失，或 opencompass CLI 不可用。"
                    f" exit_code={completed.returncode}"
                )
                detail = f"opencompass.log 为 0 字节，命令: {' '.join(command)}"
            else:
                message = f"OpenCompass 返回非零退出码（exit_code={completed.returncode}）"
                detail = f"日志末尾:\n{tail_text}"
            return ExecuteResult(
                ok=False,
                output_dir=str(target_output_dir),
                artifacts=self._collect_artifacts(target_output_dir),
                error=AdapterError(
                    code="OPENCOMPASS_NON_ZERO_EXIT",
                    message=message,
                    detail=detail,
                ),
            )

        return ExecuteResult(ok=True, output_dir=str(target_output_dir), artifacts=self._collect_artifacts(target_output_dir))

    @staticmethod
    def _tail_log(path: Path, *, max_bytes: int = 4096) -> str:
        try:
            stat = path.stat()
        except OSError:
            return ""
        if stat.st_size == 0:
            return ""
        try:
            with path.open("rb") as f:
                if stat.st_size > max_bytes:
                    f.seek(-max_bytes, 2)
                data = f.read()
        except OSError:
            return ""
        try:
            return data.decode("utf-8", errors="replace").strip()
        except Exception:  # pragma: no cover
            return ""

    def cancel_eval(self, eval_task_id: str) -> bool:
        """对指定任务的子进程组发 SIGTERM，必要时升级到 SIGKILL。返回当时是否有运行中的进程。"""
        with self._lock:
            proc = self._running.get(eval_task_id)
            self._cancelled.add(eval_task_id)
        if proc is None:
            return False
        if proc.poll() is not None:
            return False
        try:
            if os.name != "nt":
                pgid = os.getpgid(proc.pid)
                os.killpg(pgid, signal.SIGTERM)
            else:  # pragma: no cover
                proc.terminate()
        except ProcessLookupError:
            return False
        except PermissionError as exc:
            logger.warning("终止 OpenCompass 进程组失败 task=%s: %s", eval_task_id, exc)
            return False

        # 5 秒优雅退出，再不行就 SIGKILL
        deadline = time.time() + 5
        while time.time() < deadline:
            if proc.poll() is not None:
                return True
            time.sleep(0.2)
        try:
            if os.name != "nt":
                os.killpg(os.getpgid(proc.pid), signal.SIGKILL)
            else:  # pragma: no cover
                proc.kill()
        except ProcessLookupError:
            pass
        return True

    def parse_eval_result(self, output_dir: str) -> ParsedResult:
        target_output_dir = Path(output_dir).expanduser().resolve()
        if not target_output_dir.exists():
            return ParsedResult(
                ok=False,
                error=AdapterError(code="OUTPUT_DIR_NOT_FOUND", message="输出目录不存在", detail=str(target_output_dir)),
            )

        run_dir = self._latest_run_dir(target_output_dir)
        artifacts = self._collect_artifacts(target_output_dir)
        log_path = self._resolve_log_path(target_output_dir, run_dir)
        metadata: dict[str, str] = {
            "output_dir": str(target_output_dir),
            "run_dir": str(run_dir) if run_dir else "",
        }

        if run_dir is None:
            metadata["valid_metric_count"] = "0"
            metadata["failure_reason"] = "OpenCompass 未生成运行产物目录"
            return ParsedResult(
                ok=True,
                artifacts=artifacts,
                log_path=log_path,
                metadata=metadata,
            )

        summary_csv = self._latest_summary_file(run_dir, ".csv")
        summary_md = self._latest_summary_file(run_dir, ".md")
        metrics, valid_count = self._load_metrics_from_csv(summary_csv) if summary_csv else ([], 0)

        report_path = str(summary_md) if summary_md else ""
        raw_result_path = str(summary_csv) if summary_csv else ""

        metadata["valid_metric_count"] = str(valid_count)
        if valid_count == 0:
            metadata["failure_reason"] = self._diagnose_failure(run_dir)

        return ParsedResult(
            ok=True,
            metrics=tuple(metrics),
            artifacts=artifacts,
            raw_result_path=raw_result_path,
            report_path=report_path,
            log_path=log_path,
            metadata=metadata,
        )

    def _resolve_output_dir(self, config: Any) -> Path:
        work_dir = config.runtime.work_dir or str(self._settings.runtime.work_dir)
        task_name = config.extra_params.get("eval_task_id") or "manual_eval"
        return Path(work_dir).expanduser().resolve() / task_name

    def _to_config_payload(self, config: Any, output_dir: Path) -> dict[str, Any]:
        return {
            "model": {
                "type": "remote_api",
                "provider": config.model.provider,
                "model_name": config.model.model_name,
                "base_url": config.model.base_url,
                "api_key": config.model.api_key,
                "params": dict(config.model.params),
            },
            "dataset": {
                "name": config.dataset.name,
                "path": config.dataset.path,
                "params": dict(config.dataset.params),
            },
            "runtime": {
                "work_dir": str(output_dir),
                "timeout_seconds": config.runtime.timeout_seconds or self._settings.runtime.timeout_seconds,
                "max_workers": config.runtime.max_workers or self._settings.server.max_workers,
                "keep_raw_outputs": config.runtime.keep_raw_outputs,
            },
            "extra_params": dict(config.extra_params),
        }

    def _resolve_opencompass_command(self, config: Any) -> str | None:
        configured = config.extra_params.get("opencompass_command")
        if configured:
            return configured

        # 优先使用当前 Python 解释器同目录下的可执行文件（venv 安装的入口一定在这里）。
        python_dir = Path(sys.executable).parent
        for name in ("opencompass", "opencompass.exe"):
            candidate = python_dir / name
            if candidate.is_file() and os.access(candidate, os.X_OK):
                return str(candidate)

        # 回退到 PATH 查找，兼容全局安装。
        return shutil.which("opencompass")

    def _build_command(self, opencompass_command: str, config: Any, config_path: str, output_dir: Path) -> list[str]:
        custom_args = config.extra_params.get("opencompass_args")
        if custom_args:
            return [opencompass_command, *custom_args.split()]

        # 用 build 阶段生成的 mmengine .py 配置驱动 OpenCompass，避免 CLI 拼模型/数据集参数。
        return [
            opencompass_command,
            str(config_path),
            "--work-dir",
            str(output_dir),
            "--dump-eval-details",
        ]

    def _write_mmengine_config(self, config: Any, output_dir: Path) -> Path:
        dataset_name = config.dataset.name or self._settings.opencompass.default_dataset
        dataset_module, dataset_var = self._resolve_dataset_module(dataset_name)
        model_block = self._render_model_block(config)

        config_text = (
            "from mmengine.config import read_base\n"
            "from opencompass.models import OpenAISDK\n"
            "\n"
            "with read_base():\n"
            f"    from {dataset_module} import {dataset_var}\n"
            "\n"
            f"datasets = {dataset_var}\n"
            "\n"
            f"models = [\n{model_block}]\n"
        )

        config_path = output_dir / "opencompass_eval_config.py"
        config_path.write_text(config_text, encoding="utf-8")
        return config_path

    def _resolve_dataset_module(self, dataset_name: str) -> tuple[str, str]:
        """根据数据集名定位 OpenCompass 安装目录下的 dataset 配置模块，并返回导出变量名。"""
        if not dataset_name:
            raise FileNotFoundError("数据集名称不能为空")

        # 优先把 dataset_name 当作 OpenCompass 自带 demo 配置（demo_*）。
        try:
            import opencompass

            base = Path(opencompass.__file__).resolve().parent / "configs" / "datasets"
        except ImportError as exc:  # pragma: no cover
            raise FileNotFoundError(f"未安装 opencompass: {exc}") from exc

        candidates: list[Path] = []
        if dataset_name.startswith("demo_"):
            candidates.append(base / "demo" / f"{dataset_name}.py")
        candidates.extend(base.glob(f"**/{dataset_name}.py"))

        target: Path | None = next((path for path in candidates if path.is_file()), None)
        if target is None:
            available = sorted(p.stem for p in (base / "demo").glob("demo_*.py"))
            raise FileNotFoundError(
                f"未找到数据集配置 {dataset_name}.py，可用 demo 数据集: {', '.join(available) or '空'}"
            )

        text = target.read_text(encoding="utf-8")
        match = _DEMO_IMPORT_RE.search(text) or _DATASETS_VAR_RE.search(text)
        if not match:
            raise FileNotFoundError(f"无法在 {target} 中识别 *_datasets 变量")
        var_name = match.group(1)

        rel = target.relative_to(base.parent.parent.parent).with_suffix("")
        module = ".".join(rel.parts)
        return module, var_name

    def _render_model_block(self, config: Any) -> str:
        api_key = config.model.api_key or ""
        base_url = config.model.base_url or ""
        model_name = config.model.model_name or ""
        params = dict(config.model.params or {})

        max_seq_len = int(params.get("max_seq_len") or 4096)
        max_out_len = int(params.get("max_out_len") or 1024)
        batch_size = int(params.get("batch_size") or 4)
        # 默认从 1 上调到 5：OpenAI 兼容厂商（DashScope 等）一般容忍这个量级，
        # 配合 max_workers=8 大幅缩短推理总时长。可由 model.params 覆盖。
        query_per_second = int(params.get("query_per_second") or 5)
        max_workers = int(params.get("max_workers") or 8)
        # OpenCompass OpenAISDK 默认 temperature=None，会被原样透传到 OpenAI 兼容端点。
        # DashScope（qwen 兼容接口）严格校验 temperature 为 Float，None 会触发 400
        # "'temperature' must be Float"。这里默认给 0.0，允许 model.params.temperature 覆盖。
        temperature = self._coerce_float(params.get("temperature"), default=0.0)

        fields = [
            f"        abbr={self._py_repr(model_name or 'remote-model')},",
            "        type=OpenAISDK,",
            f"        path={self._py_repr(model_name)},",
            f"        key={self._py_repr(api_key)},",
        ]
        if base_url:
            fields.append(f"        openai_api_base={self._py_repr(base_url)},")
        fields.extend(
            [
                f"        query_per_second={query_per_second},",
                f"        max_out_len={max_out_len},",
                f"        max_seq_len={max_seq_len},",
                f"        batch_size={batch_size},",
                f"        max_workers={max_workers},",
                f"        temperature={self._py_float(temperature)},",
            ]
        )
        return "    dict(\n" + "\n".join(fields) + "\n    ),\n"

    @staticmethod
    def _coerce_float(value: Any, *, default: float) -> float:
        if value is None or value == "":
            return float(default)
        try:
            return float(value)
        except (TypeError, ValueError):
            return float(default)

    @staticmethod
    def _py_float(value: float) -> str:
        # 保证 Python 字面量带小数点，避免被解析为 int 后 JSON 序列化为整数。
        if float(value).is_integer():
            return f"{float(value):.1f}"
        return repr(float(value))

    @staticmethod
    def _py_repr(value: str) -> str:
        return repr(value)

    def _resolve_timeout(self, config: Any) -> int:
        return int(config.runtime.timeout_seconds or self._settings.runtime.timeout_seconds)

    def _collect_artifacts(self, output_dir: Path) -> tuple[ArtifactRef, ...]:
        artifacts: list[ArtifactRef] = []
        for path in output_dir.rglob("*"):
            if not path.is_file():
                continue
            artifacts.append(
                ArtifactRef(
                    type_name=self._artifact_type_name(path),
                    name=path.name,
                    path=str(path),
                    description="OpenCompass 运行产物",
                )
            )
        return tuple(artifacts)

    def _artifact_type_name(self, path: Path) -> str:
        if path.suffix == ".log":
            return "log"
        if path.suffix in {".md", ".html"}:
            return "report"
        if path.suffix == ".json":
            return "raw_result"
        return "other"

    def _latest_run_dir(self, output_dir: Path) -> Path | None:
        """OpenCompass 0.5.x 把每次运行产物放在 <output_dir>/<YYYYMMDD_HHMMSS>/ 下。"""
        pattern = re.compile(r"^\d{8}_\d{6}$")
        runs = [p for p in output_dir.iterdir() if p.is_dir() and pattern.match(p.name)]
        if not runs:
            return None
        runs.sort(key=lambda p: p.name, reverse=True)
        return runs[0]

    def _latest_summary_file(self, run_dir: Path, suffix: str) -> Path | None:
        summary_dir = run_dir / "summary"
        if not summary_dir.is_dir():
            return None
        candidates = sorted(summary_dir.glob(f"summary_*{suffix}"), key=lambda p: p.name, reverse=True)
        return candidates[0] if candidates else None

    def _resolve_log_path(self, output_dir: Path, run_dir: Path | None) -> str:
        # 优先用顶层重定向 stdout 的 opencompass.log（包含完整运行输出）。
        top_log = output_dir / "opencompass.log"
        if top_log.is_file():
            return str(top_log)
        if run_dir is None:
            return ""
        infer_logs = list((run_dir / "logs" / "infer").rglob("*.out"))
        if infer_logs:
            infer_logs.sort(key=lambda p: p.stat().st_mtime, reverse=True)
            return str(infer_logs[0])
        return ""

    def _load_metrics_from_csv(self, csv_path: Path) -> tuple[list[MetricValue], int]:
        """解析 summary_*.csv 取最后一列为得分，返回 (指标列表, 有效指标数)。"""
        try:
            with csv_path.open("r", encoding="utf-8") as fp:
                reader = csv.reader(fp)
                rows = [row for row in reader if row]
        except OSError:
            return [], 0

        if len(rows) < 2:
            return [], 0

        header = rows[0]
        if len(header) < 2:
            return [], 0
        model_name = header[-1]

        metrics: list[MetricValue] = []
        valid = 0
        for row in rows[1:]:
            if len(row) < 2:
                continue
            dataset = row[0] or "unknown"
            metric_name = row[2] if len(row) >= 3 and row[2] not in ("", "-") else "score"
            mode = row[3] if len(row) >= 4 else ""
            value_str = row[-1]

            display_name = f"{dataset} · {metric_name}" if metric_name != "score" else dataset
            extra = {"dataset": dataset, "model": model_name, "mode": mode}

            try:
                value = float(value_str)
            except (TypeError, ValueError):
                metrics.append(
                    MetricValue(
                        name=f"{dataset}.{metric_name}",
                        value=0.0,
                        display_name=display_name,
                        description="未生成有效得分",
                        extra=extra,
                    )
                )
                continue

            metrics.append(
                MetricValue(
                    name=f"{dataset}.{metric_name}",
                    value=value,
                    display_name=display_name,
                    description=f"OpenCompass 输出 ({mode})" if mode and mode != "-" else "OpenCompass 输出",
                    extra=extra,
                )
            )
            valid += 1

        return metrics, valid

    def _diagnose_failure(self, run_dir: Path) -> str:
        """从 logs/infer/ 抽取最后一行错误，给前端一个可读的失败原因。"""
        infer_logs = list((run_dir / "logs" / "infer").rglob("*.out"))
        for log in infer_logs:
            try:
                tail = log.read_text(encoding="utf-8", errors="ignore").strip().splitlines()
            except OSError:
                continue
            for line in reversed(tail[-50:]):
                if "Error" in line or "ERROR" in line or "Traceback" in line:
                    return line.strip()[:300]
        return "推理日志未发现明确错误，请查看 log_path 详细排查"
