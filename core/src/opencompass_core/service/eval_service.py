from __future__ import annotations

import logging

from opencompass_core import __version__
from opencompass_core.adapter import OpenCompassAdapter
from opencompass_core.adapter.opencompass_adapter import AdapterError, ArtifactRef, MetricValue
from opencompass_core.grpc.generated import ensure_generated_path

ensure_generated_path()

try:
    import eval_service_pb2
    import eval_service_pb2_grpc
except ImportError as exc:  # pragma: no cover - 生成 proto 前用于给出明确错误
    raise ImportError("请先运行 `python scripts/generate_proto.py` 生成 gRPC 代码") from exc

logger = logging.getLogger(__name__)


class EvalService(eval_service_pb2_grpc.EvalServiceServicer):
    def __init__(self, adapter: OpenCompassAdapter) -> None:
        self._adapter = adapter

    def HealthCheck(self, request, context):
        return eval_service_pb2.HealthCheckResponse(
            ok=True,
            service="OpenCompassCore",
            version=__version__,
            message="Core 服务可用",
        )

    def ValidateEvalConfig(self, request, context):
        issues = self._adapter.validate_eval_config(request.config)
        return eval_service_pb2.ValidateEvalConfigResponse(
            valid=len(issues) == 0,
            errors=[
                eval_service_pb2.ValidationError(field=issue.field, message=issue.message)
                for issue in issues
            ],
        )

    def BuildEvalConfig(self, request, context):
        result = self._adapter.build_eval_config(request.config)
        return eval_service_pb2.BuildEvalConfigResponse(
            ok=result.ok,
            config_path=result.config_path,
            output_dir=result.output_dir,
            artifacts=[_to_artifact_pb(artifact) for artifact in result.artifacts],
            error=_to_error_pb(result.error),
        )

    def ExecuteEval(self, request, context):
        result = self._adapter.execute_eval(
            eval_task_id=request.eval_task_id,
            config=request.config,
            config_path=request.config_path,
            output_dir=request.output_dir,
            reuse_timestamp=request.reuse_timestamp,
        )
        return eval_service_pb2.ExecuteEvalResponse(
            ok=result.ok,
            output_dir=result.output_dir,
            artifacts=[_to_artifact_pb(artifact) for artifact in result.artifacts],
            error=_to_error_pb(result.error),
        )

    def ParseEvalResult(self, request, context):
        result = self._adapter.parse_eval_result(request.output_dir)
        return eval_service_pb2.ParseEvalResultResponse(
            ok=result.ok,
            result=eval_service_pb2.EvalResult(
                metrics=[_to_metric_pb(metric) for metric in result.metrics],
                artifacts=[_to_artifact_pb(artifact) for artifact in result.artifacts],
                raw_result_path=result.raw_result_path,
                report_path=result.report_path,
                log_path=result.log_path,
                metadata=result.metadata or {},
            ),
            error=_to_error_pb(result.error),
        )

    def CancelEval(self, request, context):
        running = self._adapter.cancel_eval(request.eval_task_id)
        return eval_service_pb2.CancelEvalResponse(ok=True, running=running)

    def PullHuggingFaceDataset(self, request, context):
        result = self._adapter.pull_huggingface_dataset(
            repo=request.repo,
            subset=request.subset,
            split=request.split,
            cache_dir=request.cache_dir,
        )
        return eval_service_pb2.PullHuggingFaceDatasetResponse(
            ok=result.ok,
            local_path=result.local_path,
            sample_count=result.sample_count,
            error=_to_error_pb(result.error),
        )

    def PrepareCustomDataset(self, request, context):
        result = self._adapter.prepare_custom_dataset(
            local_path=request.local_path,
            task_type=request.task_type,
        )
        return eval_service_pb2.PrepareCustomDatasetResponse(
            ok=result.ok,
            config_path=result.config_path,
            sample_count=result.sample_count,
            error=_to_error_pb(result.error),
        )


def _to_artifact_pb(artifact: ArtifactRef):
    return eval_service_pb2.Artifact(
        type=_artifact_type_to_pb(artifact.type_name),
        name=artifact.name,
        path=artifact.path,
        description=artifact.description,
    )


def _to_metric_pb(metric: MetricValue):
    return eval_service_pb2.Metric(
        name=metric.name,
        value=metric.value,
        display_name=metric.display_name,
        description=metric.description,
        extra=metric.extra or {},
    )


def _to_error_pb(error: AdapterError | None):
    if error is None:
        return eval_service_pb2.CoreError()
    return eval_service_pb2.CoreError(code=error.code, message=error.message, detail=error.detail)


def _artifact_type_to_pb(type_name: str) -> int:
    mapping = {
        "config": eval_service_pb2.ARTIFACT_TYPE_CONFIG,
        "raw_result": eval_service_pb2.ARTIFACT_TYPE_RAW_RESULT,
        "report": eval_service_pb2.ARTIFACT_TYPE_REPORT,
        "log": eval_service_pb2.ARTIFACT_TYPE_LOG,
        "other": eval_service_pb2.ARTIFACT_TYPE_OTHER,
    }
    return mapping.get(type_name, eval_service_pb2.ARTIFACT_TYPE_UNSPECIFIED)
