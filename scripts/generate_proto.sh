#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PROTO_DIR="${ROOT_DIR}/proto"
CORE_OUT="${ROOT_DIR}/core/src/opencompass_core/grpc/gen"
GO_BIN="$(go env GOPATH)/bin"

export PATH="${GO_BIN}:${PATH}"

if ! command -v protoc-gen-go >/dev/null 2>&1; then
  echo "未找到 protoc-gen-go，开始安装..."
  go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
fi

if ! command -v protoc-gen-go-grpc >/dev/null 2>&1; then
  echo "未找到 protoc-gen-go-grpc，开始安装..."
  go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
fi

if ! command -v buf >/dev/null 2>&1; then
  echo "未找到 buf，开始安装..."
  go install github.com/bufbuild/buf/cmd/buf@latest
fi

mkdir -p "${CORE_OUT}"

cd "${ROOT_DIR}"
buf generate

python3 -m pip install -r "${ROOT_DIR}/core/requirements.txt"
python3 -m grpc_tools.protoc \
  -I "${PROTO_DIR}" \
  --python_out="${CORE_OUT}" \
  --grpc_python_out="${CORE_OUT}" \
  "${PROTO_DIR}/eval_service.proto"

echo "proto 代码生成完成"
