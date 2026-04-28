from __future__ import annotations

import logging
from concurrent import futures

import grpc

from opencompass_core.adapter import OpenCompassAdapter
from opencompass_core.config import CoreSettings
from opencompass_core.grpc.generated import ensure_generated_path
from opencompass_core.service import EvalService

ensure_generated_path()

try:
    import eval_service_pb2_grpc
except ImportError as exc:  # pragma: no cover - 生成 proto 前用于给出明确错误
    raise ImportError("请先运行 `python scripts/generate_proto.py` 生成 gRPC 代码") from exc

logger = logging.getLogger(__name__)


def serve(settings: CoreSettings) -> None:
    adapter = OpenCompassAdapter(settings)
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=settings.server.max_workers))
    eval_service_pb2_grpc.add_EvalServiceServicer_to_server(EvalService(adapter), server)
    server.add_insecure_port(settings.grpc_address)
    server.start()
    logger.info("OpenCompass Core gRPC 服务已启动，address=%s", settings.grpc_address)
    server.wait_for_termination()
