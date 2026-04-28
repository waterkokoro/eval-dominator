#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT_DIR}/frontend"

if [ ! -f ".env.development" ]; then
  cp ".env.development.example" ".env.development"
  echo "已创建 frontend/.env.development，请按需修改后端 API 地址。"
fi

npm install
npm run serve
