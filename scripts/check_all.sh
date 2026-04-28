#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

echo "检查 Python Core"
cd "${ROOT_DIR}"
python3 -m compileall core/src core/scripts
PYTHONPATH=core/src python3 core/scripts/check_config.py --config core/config/config.example.yaml

echo "检查 Go Backend"
cd "${ROOT_DIR}/backend"
gofmt -w cmd internal
GOTOOLCHAIN=local go test ./...

echo "检查 Frontend"
cd "${ROOT_DIR}/frontend"
npm run build

echo "全部检查完成"
