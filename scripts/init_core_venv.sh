#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CORE_DIR="${ROOT_DIR}/core"
VENV_DIR="${CORE_DIR}/.venv"

if command -v python3.10 >/dev/null 2>&1; then
  PYTHON_BIN="python3.10"
elif command -v pyenv >/dev/null 2>&1 && pyenv versions --bare | grep -q "^3.10"; then
  PYTHON_BIN="$(pyenv root)/versions/$(pyenv versions --bare | grep '^3.10' | head -n 1)/bin/python"
else
  echo "未找到 Python 3.10，OpenCompass 不建议使用当前 Python 3.13 安装。"
  echo "建议执行以下命令安装 Python 3.10："
  echo "  pyenv install 3.10.14"
  echo "  pyenv local 3.10.14"
  echo "然后重新执行："
  echo "  ./scripts/init_core_venv.sh"
  exit 1
fi

echo "使用 Python: $(${PYTHON_BIN} --version)"

cd "${CORE_DIR}"
"${PYTHON_BIN}" -m venv "${VENV_DIR}"
"${VENV_DIR}/bin/python" -m pip install --upgrade pip setuptools wheel
"${VENV_DIR}/bin/python" -m pip install -r requirements-opencompass.txt

echo "Core 虚拟环境初始化完成: ${VENV_DIR}"
echo "验证 opencompass:"
echo "  ${VENV_DIR}/bin/opencompass --help"
