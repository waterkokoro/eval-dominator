#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT_DIR}/backend"

if [ ! -f "config/config.yaml" ]; then
  cp "config/config.example.yaml" "config/config.yaml"
  echo "已创建 backend/config/config.yaml，请按需修改 JWT secret、Core 地址和数据库路径。"
fi

GOTOOLCHAIN=local go mod tidy
GOTOOLCHAIN=local go run ./cmd/server --config config/config.yaml --migration migrations/001_init.sql
