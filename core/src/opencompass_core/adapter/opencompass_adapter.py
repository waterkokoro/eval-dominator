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

# 确保自定义 evaluator 注册到 OpenCompass ICL_EVALUATORS registry（触发 @register_module 装饰器）
import opencompass_core.evaluators as _evaluators_mod  # noqa: F401

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

        # 确保数据集缓存目录存在
        self._settings.opencompass.dataset_cache_dir.mkdir(parents=True, exist_ok=True)

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

    def execute_eval(
        self,
        eval_task_id: str,
        config: Any,
        config_path: str,
        output_dir: str,
        reuse_timestamp: str = "",
    ) -> ExecuteResult:
        target_output_dir = Path(output_dir).expanduser().resolve() if output_dir else self._resolve_output_dir(config)
        target_output_dir.mkdir(parents=True, exist_ok=True)

        # 仅重跑评测节点（reuse 模式）：复用 output_dir/<reuse_timestamp>/ 下的 predictions/，
        # 由 OpenCompass `-r <ts>` 跳过推理直接复用预测结果，仅重新生成 results/summary。
        # 在执行前清理旧的 results/summary/logs/eval/，避免 OpenICLEval 把残留文件当成
        # 已完成 task 而跳过；保留 predictions/ 与 configs/。
        if reuse_timestamp:
            run_dir = target_output_dir / reuse_timestamp
            if not run_dir.is_dir():
                return ExecuteResult(
                    ok=False,
                    output_dir=str(target_output_dir),
                    error=AdapterError(
                        code="REUSE_RUN_DIR_NOT_FOUND",
                        message="未找到要复用的运行目录",
                        detail=f"reuse_timestamp={reuse_timestamp} 对应目录不存在: {run_dir}",
                    ),
                )
            predictions_dir = run_dir / "predictions"
            if not predictions_dir.is_dir() or not any(predictions_dir.rglob("*.json")):
                return ExecuteResult(
                    ok=False,
                    output_dir=str(target_output_dir),
                    error=AdapterError(
                        code="REUSE_PREDICTIONS_MISSING",
                        message="预测结果不存在，无法复用",
                        detail=f"{predictions_dir} 下未找到任何 *.json 预测文件",
                    ),
                )
            # 清理上一轮 evaluate 的产物，避免被 OpenCompass 误认为已完成
            for sub in ("results", "summary"):
                target = run_dir / sub
                if target.exists():
                    shutil.rmtree(target, ignore_errors=True)
            eval_log_dir = run_dir / "logs" / "eval"
            if eval_log_dir.exists():
                shutil.rmtree(eval_log_dir, ignore_errors=True)
            logger.info(
                "evaluate 节点重跑模式：output_dir=%s, reuse_timestamp=%s",
                target_output_dir,
                reuse_timestamp,
            )

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

        command = self._build_command(
            opencompass_command, config, config_path, target_output_dir, reuse_timestamp=reuse_timestamp
        )
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

        # 构建子进程环境：注入 opencompass_core 的 src/ 绝对路径到 PYTHONPATH，
        # 使 opencompass 子进程加载 config 时能 import opencompass_core.evaluators。
        _src_dir = str(Path(__file__).resolve().parent.parent.parent)  # adapter → opencompass_core → src
        sub_env = os.environ.copy()
        existing_pp = sub_env.get("PYTHONPATH", "")
        sub_env["PYTHONPATH"] = f"{_src_dir}:{existing_pp}" if existing_pp else _src_dir

        try:
            with log_path.open("w", encoding="utf-8") as log_file:
                proc = subprocess.Popen(
                    command,
                    cwd=run_cwd,
                    env=sub_env,
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

    # ------------------------------------------------------------------
    # HuggingFace 数据集拉取
    # ------------------------------------------------------------------

    @dataclass(frozen=True)
    class _HFPullResult:
        ok: bool
        local_path: str = ""
        sample_count: int = 0
        error: "AdapterError | None" = None

    def pull_huggingface_dataset(self, repo: str, subset: str, split: str, cache_dir: str) -> _HFPullResult:
        """通过 Python datasets 库下载 HuggingFace 数据集到本地。"""
        if not repo:
            return self._HFPullResult(
                ok=False,
                error=AdapterError(code="INVALID_REPO", message="HuggingFace 仓库名不能为空"),
            )

        target_dir = Path(cache_dir) if cache_dir else self._settings.opencompass.dataset_cache_dir
        target_dir.mkdir(parents=True, exist_ok=True)

        try:
            from datasets import load_dataset

            logger.info("开始下载 HuggingFace 数据集: repo=%s, subset=%s, split=%s", repo, subset, split)

            # 临时禁用离线模式以允许网络下载。
            # huggingface_hub 在首次 import 时就缓存了 HF_HUB_OFFLINE 的值，
            # 仅修改 os.environ 不够，还需要 patch 已缓存的常量。
            import importlib
            old_hf_offline = os.environ.get("HF_HUB_OFFLINE")
            old_tf_offline = os.environ.get("TRANSFORMERS_OFFLINE")
            old_ds_offline = os.environ.get("HF_DATASETS_OFFLINE")
            os.environ["HF_HUB_OFFLINE"] = "0"
            os.environ["TRANSFORMERS_OFFLINE"] = "0"
            os.environ["HF_DATASETS_OFFLINE"] = "0"

            try:
                import huggingface_hub as huggingface_hub_mod
                hf_constants = importlib.import_module("huggingface_hub.constants")
                old_constants_offline = getattr(hf_constants, "HF_HUB_OFFLINE", None)
                hf_constants.HF_HUB_OFFLINE = False
                # 同时 patch huggingface_hub 顶层属性（部分版本从此处读取）
                old_hub_offline = getattr(huggingface_hub_mod, "HF_HUB_OFFLINE", None)
                huggingface_hub_mod.HF_HUB_OFFLINE = False
                # patch datasets 库自身的离线缓存常量
                import datasets.config as ds_config
                old_ds_offline_const = getattr(ds_config, "HF_DATASETS_OFFLINE", None)
                old_ds_hub_offline_const = getattr(ds_config, "HF_HUB_OFFLINE", None)
                ds_config.HF_DATASETS_OFFLINE = False
                ds_config.HF_HUB_OFFLINE = False
            except (ImportError, AttributeError):
                old_constants_offline = None
                old_hub_offline = None
                old_ds_offline_const = None
                old_ds_hub_offline_const = None
                hf_constants = None
                huggingface_hub_mod = None
                ds_config = None

            try:
                load_kwargs = {"path": repo, "cache_dir": str(target_dir)}
                if subset:
                    load_kwargs["name"] = subset
                if split:
                    load_kwargs["split"] = split

                fallback_df = None  # 回退路径用 pandas DataFrame

                # 尝试正常加载；若遇 pyarrow 类型冲突等 DatasetGenerationError，
                # 则回退到 snapshot_download + pandas 加载以容忍混合类型列。
                try:
                    ds = load_dataset(**load_kwargs)
                except Exception as first_exc:
                    logger.warning("load_dataset 首次加载失败 (%s: %s)，尝试回退策略", type(first_exc).__name__, first_exc)
                    # 回退：使用 snapshot_download 下载原始文件，然后用 pandas 读取 JSON/JSONL
                    try:
                        from huggingface_hub import snapshot_download
                        import pandas as pd
                        from datasets import Dataset, DatasetDict

                        snap_dir = snapshot_download(
                            repo_id=repo,
                            repo_type="dataset",
                            cache_dir=str(target_dir),
                            allow_patterns=["*.json", "*.jsonl", "*.csv", "*.parquet"],
                        )
                        snap_path = Path(snap_dir)
                        data_files = sorted(snap_path.rglob("*.json")) + sorted(snap_path.rglob("*.jsonl")) + sorted(snap_path.rglob("*.csv"))

                        if not data_files:
                            raise RuntimeError(f"未找到可用数据文件: {snap_path}")

                        # 统计所有文件行数，并选择最大文件做预览（避免选到模板配置文件）
                        first_df = None
                        best_df = None
                        best_size = 0
                        total_rows = 0
                        for f in data_files:
                            try:
                                if f.suffix == ".csv":
                                    tmp_df = pd.read_csv(f)
                                else:
                                    tmp_df = pd.read_json(f, lines=f.suffix == ".jsonl" or f.name.endswith(".jsonl"), orient="records")
                                total_rows += len(tmp_df)
                                if first_df is None:
                                    first_df = tmp_df
                                fsize = f.stat().st_size
                                if fsize > best_size:
                                    best_size = fsize
                                    best_df = tmp_df
                                    logger.info("回退加载选用文件: %s (%d 行, %d 列, %d bytes)", f.name, len(tmp_df), len(tmp_df.columns), fsize)
                            except Exception as read_exc:
                                logger.warning("跳过文件 %s: %s", f.name, read_exc)

                        if first_df is None:
                            raise RuntimeError("所有数据文件均无法解析")

                        df = best_df if best_df is not None else first_df

                        # 回退路径直接通过 pandas 保存 JSONL，避免 Dataset.to_json 在超宽数据集上失败
                        # 注意：不再做 pyarrow 类型清洗，因为不再创建 Dataset 对象，
                        # pandas.to_json 可直接序列化 dict/list 等复杂类型。
                        fallback_df = df
                        sample_count = total_rows
                        logger.info("回退加载成功: 预览 %d 行 %d 列，总计 %d 行", len(df), len(df.columns), total_rows)
                    except Exception as fallback_exc:
                        raise first_exc from fallback_exc
            finally:
                # 恢复原有的离线模式设置
                if old_hf_offline is not None:
                    os.environ["HF_HUB_OFFLINE"] = old_hf_offline
                else:
                    os.environ.pop("HF_HUB_OFFLINE", None)
                if old_tf_offline is not None:
                    os.environ["TRANSFORMERS_OFFLINE"] = old_tf_offline
                else:
                    os.environ.pop("TRANSFORMERS_OFFLINE", None)
                if old_ds_offline is not None:
                    os.environ["HF_DATASETS_OFFLINE"] = old_ds_offline
                else:
                    os.environ.pop("HF_DATASETS_OFFLINE", None)
                # 恢复 huggingface_hub 缓存的常量
                try:
                    if hf_constants is not None and old_constants_offline is not None:
                        hf_constants.HF_HUB_OFFLINE = old_constants_offline
                    if huggingface_hub_mod is not None and old_hub_offline is not None:
                        huggingface_hub_mod.HF_HUB_OFFLINE = old_hub_offline
                    if ds_config is not None and old_ds_offline_const is not None:
                        ds_config.HF_DATASETS_OFFLINE = old_ds_offline_const
                    if ds_config is not None and old_ds_hub_offline_const is not None:
                        ds_config.HF_HUB_OFFLINE = old_ds_hub_offline_const
                except (AttributeError, NameError):
                    pass

            # 计算样本数
            if fallback_df is not None:
                sample_count = len(fallback_df)
            elif split:
                sample_count = len(ds)
            elif hasattr(ds, "values"):
                # DatasetDict: 取第一个 split 的长度
                sample_count = sum(len(v) for v in ds.values())
            else:
                sample_count = len(ds)

            # 将数据集保存为本地文件以便后续使用
            safe_name = repo.replace("/", "__")
            local_path = target_dir / f"{safe_name}"
            if subset:
                local_path = local_path / subset
            local_path.mkdir(parents=True, exist_ok=True)

            # 保存为 JSONL 格式
            output_file = local_path / "data.jsonl"
            if fallback_df is not None:
                # 回退路径：直接用 pandas 保存，避免 Dataset.to_json 在超宽数据集上失败
                fallback_df.to_json(str(output_file), orient="records", lines=True, force_ascii=False)
            elif hasattr(ds, "items") and not hasattr(ds, "to_json"):
                # DatasetDict: 取第一个 split
                first_split = next(iter(ds.values()))
                first_split.to_json(str(output_file))
            elif hasattr(ds, "to_json"):
                ds.to_json(str(output_file))

            logger.info("HuggingFace 数据集下载完成: %s -> %s (样本数: %d)", repo, output_file, sample_count)

            # 修正宽表格式（如 T-Eval），确保 PyArrow 能加载
            self._sanitize_jsonl_for_pyarrow(output_file)

            return self._HFPullResult(
                ok=True,
                local_path=str(output_file),
                sample_count=sample_count,
            )

        except ImportError as exc:
            return self._HFPullResult(
                ok=False,
                error=AdapterError(
                    code="DATASETS_NOT_INSTALLED",
                    message="未安装 datasets 库",
                    detail=f"请运行 pip install datasets: {exc}",
                ),
            )
        except Exception as exc:
            return self._HFPullResult(
                ok=False,
                error=AdapterError(
                    code="HF_PULL_FAILED",
                    message=f"下载 HuggingFace 数据集失败: {type(exc).__name__}",
                    detail=str(exc),
                ),
            )

    # ------------------------------------------------------------------
    # JSONL 格式修正（解决 PyArrow 混合类型问题）
    # ------------------------------------------------------------------

    @staticmethod
    def _sanitize_jsonl_for_pyarrow(jsonl_path: Path) -> None:
        """修正 JSONL 文件中的混合类型问题，使 PyArrow 能正常加载。

        处理两类问题：
        1. 宽表格式（如 T-Eval）：少量行 × 大量列 → 转置为每列一行
        2. 同列混合 list / non-list → 统一序列化为 JSON 字符串
        """
        try:
            with open(jsonl_path, "r", encoding="utf-8") as f:
                records = [json.loads(line) for line in f if line.strip()]
        except Exception:
            return

        if not records:
            return

        # --- 检测宽表格式 ---
        num_rows = len(records)
        num_cols = len(records[0]) if records else 0

        if num_rows <= 10 and num_cols > 20:
            transposed = OpenCompassAdapter._transpose_wide_jsonl(records)
            if transposed is not None:
                with open(jsonl_path, "w", encoding="utf-8") as f:
                    for sample in transposed:
                        f.write(json.dumps(sample, ensure_ascii=False) + "\n")
                logger.info(
                    "宽表 JSONL 已转置: %d 行 × %d 列 → %d 样本",
                    num_rows, num_cols, len(transposed),
                )
                return

        # --- 通用修正：序列化混合类型列 ---
        sanitized = OpenCompassAdapter._fix_mixed_type_columns(records)
        if sanitized is not records:
            with open(jsonl_path, "w", encoding="utf-8") as f:
                for record in sanitized:
                    f.write(json.dumps(record, ensure_ascii=False) + "\n")
            logger.info("JSONL 混合类型列已序列化: %d 行", len(sanitized))

    @staticmethod
    def _transpose_wide_jsonl(records: list[dict]) -> list[dict] | None:
        """将宽表格式（少量行 × 大量列）转置为长表（每列一行）。

        典型场景：T-Eval 等评测数据集，4 行 × N 列，
        每行分别存储 None / response_format / expected_tool_call / conversation。
        转置后字段名映射到 question / reference_answer 以兼容 CustomDataset。
        """
        if not records:
            return None

        keys = list(records[0].keys())
        num_rows = len(records)

        # 推断每行的语义角色（按类型启发式）
        # list 值行 → conversation（question），
        # 复杂 dict 值行（多个 key）→ tool call（reference_answer），
        # 简单 dict 值行（单个 key）→ 元数据
        conversation_row: int | None = None
        tool_call_row: int | None = None
        for row_idx in range(num_rows):
            sample_vals = [records[row_idx].get(k) for k in keys[:20]]
            non_null = [v for v in sample_vals if v is not None]
            if not non_null:
                continue
            if isinstance(non_null[0], list) and conversation_row is None:
                conversation_row = row_idx
            elif isinstance(non_null[0], dict) and tool_call_row is None:
                # 复杂 dict（如 tool call 含 thought/name/args）优先于简单元数据 dict
                if len(non_null[0]) > 1:
                    tool_call_row = row_idx
        # 回退：如果还没找到 tool_call_row，取第一个 dict 行
        if tool_call_row is None:
            for row_idx in range(num_rows):
                sample_vals = [records[row_idx].get(k) for k in keys[:20]]
                non_null = [v for v in sample_vals if v is not None]
                if non_null and isinstance(non_null[0], dict):
                    tool_call_row = row_idx
                    break

        samples: list[dict] = []
        for key in keys:
            sample: dict = {"query_id": key}
            has_data = False
            for row_idx in range(num_rows):
                val = records[row_idx].get(key)
                if val is None:
                    continue
                has_data = True
                # 映射到标准字段名
                if row_idx == conversation_row:
                    field = "question"
                elif row_idx == tool_call_row:
                    field = "reference_answer"
                else:
                    field = f"row_{row_idx}"

                if isinstance(val, (dict, list)):
                    sample[field] = json.dumps(val, ensure_ascii=False)
                else:
                    sample[field] = str(val)
            if has_data:
                samples.append(sample)

        return samples if samples else None

    @staticmethod
    def _fix_mixed_type_columns(records: list[dict]) -> list[dict]:
        """检测并修复同列混合 list / non-list 类型，统一序列化为 JSON 字符串。"""
        if not records:
            return records

        keys = set()
        for r in records:
            keys.update(r.keys())

        needs_fix = False
        for key in keys:
            has_list = False
            has_non_list_non_null = False
            for r in records:
                v = r.get(key)
                if v is None:
                    continue
                if isinstance(v, list):
                    has_list = True
                else:
                    has_non_list_non_null = True
            if has_list and has_non_list_non_null:
                needs_fix = True
                break

        if not needs_fix:
            return records

        # 有混合列，对所有复杂类型做序列化
        fixed = []
        for r in records:
            new_r = {}
            for k, v in r.items():
                if isinstance(v, (dict, list)):
                    new_r[k] = json.dumps(v, ensure_ascii=False)
                elif v is None:
                    new_r[k] = ""
                else:
                    new_r[k] = v
            fixed.append(new_r)
        return fixed

    # ------------------------------------------------------------------
    # 自定义数据集准备
    # ------------------------------------------------------------------

    @dataclass(frozen=True)
    class _CustomDatasetResult:
        ok: bool
        config_path: str = ""
        sample_count: int = 0
        error: "AdapterError | None" = None

    def prepare_custom_dataset(self, local_path: str, task_type: str) -> _CustomDatasetResult:
        """验证自定义数据集文件格式并生成 OpenCompass 可用的配置。"""
        if not local_path:
            return self._CustomDatasetResult(
                ok=False,
                error=AdapterError(code="INVALID_PATH", message="数据集文件路径不能为空"),
            )

        file_path = Path(local_path).expanduser().resolve()
        if not file_path.exists():
            return self._CustomDatasetResult(
                ok=False,
                error=AdapterError(code="FILE_NOT_FOUND", message=f"文件不存在: {file_path}"),
            )

        suffix = file_path.suffix.lower()
        if suffix not in (".csv", ".jsonl", ".json"):
            return self._CustomDatasetResult(
                ok=False,
                error=AdapterError(
                    code="UNSUPPORTED_FORMAT",
                    message=f"不支持的文件格式: {suffix}",
                    detail="支持 .csv, .jsonl, .json",
                ),
            )

        # 读取并验证文件
        try:
            # 先修正格式问题（宽表转置 / 混合类型序列化）
            if suffix == ".jsonl":
                self._sanitize_jsonl_for_pyarrow(file_path)
            sample_count = self._count_samples(file_path, suffix)
        except Exception as exc:
            return self._CustomDatasetResult(
                ok=False,
                error=AdapterError(
                    code="READ_FAILED",
                    message=f"读取文件失败: {type(exc).__name__}",
                    detail=str(exc),
                ),
            )

        # 生成 OpenCompass 配置
        task_type = task_type or "qa"
        config_dir = self._settings.opencompass.dataset_cache_dir / "configs"
        config_dir.mkdir(parents=True, exist_ok=True)

        config_path = config_dir / f"custom_{file_path.stem}_config.py"
        config_text = self._render_custom_dataset_config(file_path, task_type)
        config_path.write_text(config_text, encoding="utf-8")

        logger.info("自定义数据集配置已生成: %s (样本数: %d)", config_path, sample_count)

        return self._CustomDatasetResult(
            ok=True,
            config_path=str(config_path),
            sample_count=sample_count,
        )

    @staticmethod
    def _count_samples(file_path: Path, suffix: str) -> int:
        """统计文件中的样本数。"""
        if suffix == ".csv":
            with file_path.open("r", encoding="utf-8") as f:
                reader = csv.reader(f)
                rows = list(reader)
                return max(0, len(rows) - 1)  # 减去表头
        elif suffix == ".jsonl":
            with file_path.open("r", encoding="utf-8") as f:
                return sum(1 for line in f if line.strip())
        elif suffix == ".json":
            with file_path.open("r", encoding="utf-8") as f:
                data = json.load(f)
                if isinstance(data, list):
                    return len(data)
                return 1
        return 0

    @staticmethod
    def _render_custom_dataset_config(file_path: Path, task_type: str) -> str:
        """生成 OpenCompass 自定义数据集的 mmengine 配置文件。"""
        if task_type == "choice":
            return OpenCompassAdapter._render_choice_config(file_path)
        elif task_type == "classification":
            return OpenCompassAdapter._render_classification_config(file_path)
        else:  # qa (default)
            return OpenCompassAdapter._render_qa_config(file_path)

    @staticmethod
    def _render_choice_config(file_path: Path) -> str:
        return (
            "from opencompass.datasets import CustomDataset\n"
            "from opencompass.openicl import ZeroRetriever, GenInferencer\n"
            "from opencompass.openicl.icl_prompt_template import PromptTemplate\n"
            "from opencompass.openicl.icl_evaluator import AccEvaluator\n"
            "\n"
            f"_dataset_path = {repr(str(file_path))}\n"
            "\n"
            "datasets = [\n"
            "    dict(\n"
            "        type=CustomDataset,\n"
            f"        path=_dataset_path,\n"
            "        reader_cfg=dict(\n"
            "            input_columns=['question', 'A', 'B', 'C', 'D'],\n"
            "            output_column='answer',\n"
            "        ),\n"
            "        infer_cfg=dict(\n"
            "            prompt_template=dict(\n"
            "                type=PromptTemplate,\n"
            "                template=dict(\n"
            "                    round=[\n"
            "                        dict(role='HUMAN', prompt='{question}\\nA. {A}\\nB. {B}\\nC. {C}\\nD. {D}\\nAnswer:'),\n"
            "                    ],\n"
            "                ),\n"
            "            ),\n"
            "            retriever=dict(type=ZeroRetriever),\n"
            "            inferencer=dict(type=GenInferencer, max_out_len=512, max_seq_len=16384),\n"
            "        ),\n"
            "        eval_cfg=dict(\n"
            "            evaluator=dict(type=AccEvaluator),\n"
            "        ),\n"
            "    )\n"
            "]\n"
        )

    # ------------------------------------------------------------------
    # Evaluator 模板（内置方案 B：固定选项）
    # ------------------------------------------------------------------

    # 每种 evaluator 对应的：import 代码、类定义（如需自定义）、evaluator 引用、默认 output_column
    _EVALUATOR_TEMPLATES: dict[str, dict] = {
        "rouge": {
            "import_line": "from opencompass.openicl.icl_evaluator import RougeEvaluator",
            "class_code": "",  # 内置，无需自定义类
            "evaluator_ref": "RougeEvaluator",
            "default_output_column": "reference_answer",
            "display_name": "ROUGE 文本相似度",
        },
        "accuracy": {
            "import_line": "from opencompass.openicl.icl_evaluator import AccEvaluator",
            "class_code": "",
            "evaluator_ref": "AccEvaluator",
            "default_output_column": "label",
            "display_name": "准确率 Accuracy",
        },
        "em": {
            "import_line": "from opencompass.openicl.icl_evaluator import EMEvaluator",
            "class_code": "",
            "evaluator_ref": "EMEvaluator",
            "default_output_column": "answer",
            "display_name": "精确匹配 EM",
        },
        "keyword_match": {
            "import_line": "from opencompass_core.evaluators import KeywordMatchEvaluator",
            "class_code": "",  # 通过 import 引入，mmengine lazy import 机制保证 dump/reload 正确序列化
            "evaluator_ref": "KeywordMatchEvaluator",
            "default_output_column": "expected_keywords",
            "display_name": "关键词命中率",
        },
        "bleu": {
            "import_line": "from opencompass.openicl.icl_evaluator import BleuEvaluator",
            "class_code": "",
            "evaluator_ref": "BleuEvaluator",
            "default_output_column": "reference_answer",
            "display_name": "BLEU 翻译指标",
        },
        "jieba_rouge": {
            "import_line": "from opencompass.openicl.icl_evaluator import JiebaRougeEvaluator",
            "class_code": "",
            "evaluator_ref": "JiebaRougeEvaluator",
            "default_output_column": "reference_answer",
            "display_name": "中文 ROUGE（jieba 分词）",
        },
    }

    @staticmethod
    def _resolve_evaluator(evaluator_type: str) -> dict:
        """根据 evaluator_type 返回 evaluator 模板，未知类型回退到 rouge。"""
        return OpenCompassAdapter._EVALUATOR_TEMPLATES.get(
            evaluator_type,
            OpenCompassAdapter._EVALUATOR_TEMPLATES["rouge"],
        )

    @staticmethod
    def _render_qa_config(file_path: Path, evaluator_type: str = "rouge") -> str:
        ev = OpenCompassAdapter._resolve_evaluator(evaluator_type)
        output_col = ev["default_output_column"]

        # 拼接 import 行（内置 evaluator 才有）
        import_line = ev["import_line"] + "\n" if ev["import_line"] else ""
        # 自定义类代码（如 KeywordMatchEvaluator）
        class_code = ev["class_code"] + "\n" if ev["class_code"] else ""

        return (
            "from opencompass.datasets import CustomDataset\n"
            "from opencompass.openicl import ZeroRetriever, GenInferencer\n"
            "from opencompass.openicl.icl_prompt_template import PromptTemplate\n"
            f"{import_line}"
            f"{class_code}"
            "\n"
            f"_dataset_path = {repr(str(file_path))}\n"
            "\n"
            "datasets = [\n"
            "    dict(\n"
            "        type=CustomDataset,\n"
            f"        path=_dataset_path,\n"
            "        reader_cfg=dict(\n"
            "            input_columns=['question'],\n"
            f"            output_column={repr(output_col)},\n"
            "        ),\n"
            "        infer_cfg=dict(\n"
            "            prompt_template=dict(\n"
            "                type=PromptTemplate,\n"
            "                template='{question}',\n"
            "            ),\n"
            "            retriever=dict(type=ZeroRetriever),\n"
            "            inferencer=dict(type=GenInferencer, max_out_len=512, max_seq_len=16384),\n"
            "        ),\n"
            "        eval_cfg=dict(\n"
            f"            evaluator=dict(type={ev['evaluator_ref']}),\n"
            "        ),\n"
            "    )\n"
            "]\n"
        )

    @staticmethod
    def _render_classification_config(file_path: Path) -> str:
        return (
            "from opencompass.datasets import CustomDataset\n"
            "from opencompass.openicl import ZeroRetriever, GenInferencer\n"
            "from opencompass.openicl.icl_prompt_template import PromptTemplate\n"
            "from opencompass.openicl.icl_evaluator import AccEvaluator\n"
            "\n"
            f"_dataset_path = {repr(str(file_path))}\n"
            "\n"
            "datasets = [\n"
            "    dict(\n"
            "        type=CustomDataset,\n"
            f"        path=_dataset_path,\n"
            "        reader_cfg=dict(\n"
            "            input_columns=['text'],\n"
            "            output_column='label',\n"
            "        ),\n"
            "        infer_cfg=dict(\n"
            "            prompt_template=dict(\n"
            "                type=PromptTemplate,\n"
            "                template='{text}',\n"
            "            ),\n"
            "            retriever=dict(type=ZeroRetriever),\n"
            "            inferencer=dict(type=GenInferencer, max_out_len=512, max_seq_len=16384),\n"
            "        ),\n"
            "        eval_cfg=dict(\n"
            "            evaluator=dict(type=AccEvaluator),\n"
            "        ),\n"
            "    )\n"
            "]\n"
        )

    @staticmethod
    def _render_auto_custom_config(
        data_file_path: str, model_block: str, evaluator_type: str = "rouge"
    ) -> str:
        """防御性回退：当 dataset_path 指向数据文件（.jsonl/.csv/.json）而非 .py 配置时，
        自动生成一个 CustomDataset mmengine 配置。支持通过 evaluator_type 选择评分方式。
        """
        # 必须解析为绝对路径，否则 OpenCompass get_data_path 会把相对路径
        # 当作 dataset id 去 DATASETS_MAPPING 查表，从而抛 KeyError。
        file_path = Path(data_file_path).expanduser().resolve()

        ev = OpenCompassAdapter._resolve_evaluator(evaluator_type)
        output_col = ev["default_output_column"]
        import_line = ev["import_line"] + "\n" if ev["import_line"] else ""
        class_code = ev["class_code"] + "\n" if ev["class_code"] else ""

        return (
            "from opencompass.datasets import CustomDataset\n"
            "from opencompass.openicl import ZeroRetriever, GenInferencer\n"
            "from opencompass.openicl.icl_prompt_template import PromptTemplate\n"
            f"{import_line}"
            f"{class_code}"
            "from opencompass.models import OpenAISDK\n"
            "\n"
            f"_dataset_path = {repr(str(file_path))}\n"
            "\n"
            "datasets = [\n"
            "    dict(\n"
            "        type=CustomDataset,\n"
            "        path=_dataset_path,\n"
            "        reader_cfg=dict(\n"
            "            input_columns=['question'],\n"
            f"            output_column={repr(output_col)},\n"
            "        ),\n"
            "        infer_cfg=dict(\n"
            "            prompt_template=dict(\n"
            "                type=PromptTemplate,\n"
            "                template='{question}',\n"
            "            ),\n"
            "            retriever=dict(type=ZeroRetriever),\n"
            "            inferencer=dict(type=GenInferencer, max_out_len=512, max_seq_len=16384),\n"
            "        ),\n"
            "        eval_cfg=dict(\n"
            f"            evaluator=dict(type={ev['evaluator_ref']}),\n"
            "        ),\n"
            "    )\n"
            "]\n"
            "\n"
            f"models = [\n{model_block}]\n"
        )

    # ------------------------------------------------------------------
    # Demo 数据集
    # ------------------------------------------------------------------

    @staticmethod
    def _sanitize_legacy_evaluator_paths(config_path: Path) -> None:
        """Rewrite legacy ``opencompass.metrics.*`` string-based evaluator
        types that were used in pre-0.5.x configs to the current
        ``opencompass.openicl.icl_evaluator`` class references.

        This prevents ``TypeError: None is not a callable object`` when
        OpenCompass 0.5.x tries to look up an evaluator that no longer
        exists under the old module path.
        """
        _LEGACY_EVALUATOR_MAP = {
            "opencompass.metrics.rouge.Rouge": "RougeEvaluator",
            "opencompass.metrics.accuracy.Accuracy": "AccEvaluator",
            "opencompass.metrics.em.Em": "EmEvaluator",
            "opencompass.metrics.f1.F1": "F1Evaluator",
            "opencompass.metrics.jaccard.Jaccard": "JaccardEvaluator",
            "opencompass.metrics.bleu.Bleu": "BleuEvaluator",
        }
        try:
            text = config_path.read_text(encoding="utf-8")
        except OSError:
            return

        original = text
        for old_path, new_class in _LEGACY_EVALUATOR_MAP.items():
            text = text.replace(f"type='{old_path}'", f"type={new_class}")
            text = text.replace(f'type="{old_path}"', f"type={new_class}")

        if text != original:
            # When replacing a string type with a class reference we also
            # need to add the import so the name resolves.
            needed_imports: set[str] = set()
            for old_path, new_class in _LEGACY_EVALUATOR_MAP.items():
                if new_class in text and f"import {new_class}" not in text:
                    needed_imports.add(new_class)
            if needed_imports:
                import_line = (
                    "from opencompass.openicl.icl_evaluator import "
                    + ", ".join(sorted(needed_imports))
                    + "\n"
                )
                # Insert after the last existing import line.
                lines = text.split("\n")
                last_import_idx = -1
                for i, line in enumerate(lines):
                    if line.startswith("from ") or line.startswith("import "):
                        last_import_idx = i
                if last_import_idx >= 0:
                    lines.insert(last_import_idx + 1, import_line.rstrip())
                else:
                    lines.insert(0, import_line.rstrip())
                text = "\n".join(lines)
            config_path.write_text(text, encoding="utf-8")
            logger.info("已修正旧版 evaluator 路径: %s", config_path)


    def list_demo_datasets(self) -> list[dict]:
        """返回内置 demo 数据集列表。"""
        demos_dir = Path(__file__).parent.parent / "demos"
        if not demos_dir.exists():
            return []

        demos = []
        for f in sorted(demos_dir.iterdir()):
            if f.is_file() and f.suffix in (".csv", ".jsonl"):
                task_type = "choice" if "choice" in f.stem else "qa"
                sample_count = 0
                try:
                    sample_count = self._count_samples(f, f.suffix)
                except Exception:
                    pass
                demos.append({
                    "name": f.stem,
                    "path": str(f),
                    "task_type": task_type,
                    "file_format": f.suffix.lstrip("."),
                    "sample_count": sample_count,
                    "description": f"内置 demo 数据集 ({task_type})",
                })
        return demos

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

    def _build_command(
        self,
        opencompass_command: str,
        config: Any,
        config_path: str,
        output_dir: Path,
        reuse_timestamp: str = "",
    ) -> list[str]:
        custom_args = config.extra_params.get("opencompass_args")
        if custom_args:
            return [opencompass_command, *custom_args.split()]

        # 用 build 阶段生成的 mmengine .py 配置驱动 OpenCompass，避免 CLI 拼模型/数据集参数。
        cmd = [
            opencompass_command,
            str(config_path),
            "--work-dir",
            str(output_dir),
            "--dump-eval-details",
        ]
        # reuse 模式：让 OpenCompass 复用 output_dir/<ts>/predictions，仅重新跑评测
        if reuse_timestamp:
            cmd.extend(["-r", reuse_timestamp])
        return cmd

    def _write_mmengine_config(self, config: Any, output_dir: Path) -> Path:
        model_block = self._render_model_block(config)

        # 从 dataset.params 读取用户选择的 evaluator 类型（方案 B：内置固定选项）
        evaluator_type = config.dataset.params.get("evaluator_type", "rouge")
        # 当用户选择了非默认 evaluator 时，Go 端会将原始数据文件路径放入 raw_data_path，
        # 此时应重新生成配置（而非使用上传时生成的 .py 配置，因其 evaluator 是固定的）。
        raw_data_path = config.dataset.params.get("raw_data_path", "")

        # 检查是否为自定义/HF 数据集（直接使用已生成的配置文件）
        # 注意：必须先 expanduser+resolve 成绝对路径，否则当 dataset_path 为
        # 相对路径（例如 DB 中存的 ../runtime/...）时，下面的 .exists() 检查
        # 会受运行时 CWD 影响而出错；同时也避免相对路径被 OpenCompass 当成
        # 内置 dataset id 抛 KeyError。
        dataset_path = config.dataset.path
        if dataset_path:
            dataset_path = str(Path(dataset_path).expanduser().resolve())

        # 如果有 raw_data_path（用户选择了特定 evaluator），优先使用它重新生成配置
        if raw_data_path and Path(raw_data_path).expanduser().resolve().exists():
            resolved_raw = str(Path(raw_data_path).expanduser().resolve())
            config_text = self._render_auto_custom_config(
                resolved_raw, model_block, evaluator_type=evaluator_type
            )
        elif dataset_path and Path(dataset_path).suffix == ".py" and Path(dataset_path).exists():
            # 将数据集配置复制到 output_dir，用文件名做相对导入，避免绝对路径语法错误
            src = Path(dataset_path).resolve()
            dst = output_dir / src.name
            shutil.copy2(src, dst)
            # 修正旧版 opencompass.metrics.* 字符串路径为 0.5.x 的 icl_evaluator 类引用
            self._sanitize_legacy_evaluator_paths(dst)
            module_name = src.stem
            config_text = (
                "from mmengine.config import read_base\n"
                "from opencompass.models import OpenAISDK\n"
                "\n"
                "with read_base():\n"
                f"    from .{module_name} import datasets as _custom_datasets\n"
                "\n"
                "datasets = _custom_datasets\n"
                "\n"
                f"models = [\n{model_block}]\n"
            )
        elif dataset_path and Path(dataset_path).suffix in (".jsonl", ".csv", ".json") and Path(dataset_path).exists():
            # 防御性回退：dataset_path 是数据文件而非 .py 配置，自动生成 CustomDataset 配置
            config_text = self._render_auto_custom_config(dataset_path, model_block, evaluator_type=evaluator_type)
        else:
            # 使用 OpenCompass 内置数据集
            dataset_name = config.dataset.name or self._settings.opencompass.default_dataset
            dataset_module, dataset_var = self._resolve_dataset_module(dataset_name)
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
        """从 logs/ 下的 infer/ 和 eval/ 子目录抽取最后一行错误，给前端一个可读的失败原因。"""
        for sub in ("infer", "eval"):
            log_dir = run_dir / "logs" / sub
            if not log_dir.is_dir():
                continue
            for log in log_dir.rglob("*.out"):
                try:
                    tail = log.read_text(encoding="utf-8", errors="ignore").strip().splitlines()
                except OSError:
                    continue
                for line in reversed(tail[-50:]):
                    if "Error" in line or "ERROR" in line or "Traceback" in line:
                        return line.strip()[:300]
        return "推理/评测日志未发现明确错误，请查看 log_path 详细排查"
