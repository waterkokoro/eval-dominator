#!/usr/bin/env bash
# ============================================================
#  Eval-Dominator 版本号统一管理脚本
#
#  用法:
#    ./scripts/bump_version.sh <new-version>
#    ./scripts/bump_version.sh 0.3.0
#
#  涉及文件:
#    - README.md                     (version badge + status line)
#    - README.en.md                  (version badge + status line)
#    - frontend/package.json         (version field)
#    - core/pyproject.toml           (version field)
# ============================================================
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# ── 颜色 ──────────────────────────────────────────────────────
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# ── 参数校验 ──────────────────────────────────────────────────
if [ $# -lt 1 ] || [ -z "${1:-}" ]; then
  echo -e "${RED}错误: 请提供新版本号${NC}"
  echo ""
  echo "用法: $(basename "$0") <new-version>"
  echo "示例: $(basename "$0") 0.3.0"
  echo ""
  echo "当前版本:"
  echo "  frontend/package.json : $(grep '"version"' "${ROOT_DIR}/frontend/package.json" | head -1 | sed 's/.*"\([0-9][^"]*\)".*/\1/')"
  echo "  core/pyproject.toml   : $(grep '^version' "${ROOT_DIR}/core/pyproject.toml" | head -1 | sed 's/.*= *"\(.*\)"/\1/')"
  echo "  README.md badge       : $(grep -o 'version-v[^-]*' "${ROOT_DIR}/README.md" | head -1 | sed 's/version-//')"
  exit 1
fi

NEW_VERSION="$1"

# 校验格式: x.y.z 或 x.y.z-suffix
if ! echo "${NEW_VERSION}" | grep -qE '^[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.]+)?$'; then
  echo -e "${RED}错误: 版本号格式不正确，期望 x.y.z 或 x.y.z-suffix${NC}"
  echo "示例: 0.2.0, 0.3.0-beta, 1.0.0-rc1"
  exit 1
fi

echo -e "${BLUE}============================================${NC}"
echo -e "${BLUE}  Eval-Dominator 版本更新 → v${NEW_VERSION}${NC}"
echo -e "${BLUE}============================================${NC}"
echo ""

# ── 读取当前版本 ──────────────────────────────────────────────
OLD_FE_VERSION=$(grep '"version"' "${ROOT_DIR}/frontend/package.json" | head -1 | sed 's/.*"\([0-9][^"]*\)".*/\1/')
OLD_CORE_VERSION=$(grep '^version' "${ROOT_DIR}/core/pyproject.toml" | head -1 | sed 's/.*= *"\(.*\)"/\1/')

echo -e "当前版本: frontend=${OLD_FE_VERSION}, core=${OLD_CORE_VERSION}"
echo -e "目标版本: ${NEW_VERSION}"
echo ""

# ── 更新 frontend/package.json ────────────────────────────────
FE_FILE="${ROOT_DIR}/frontend/package.json"
if [ -f "${FE_FILE}" ]; then
  # macOS sed 需要 -i '' 写法
  if [[ "$OSTYPE" == "darwin"* ]]; then
    sed -i '' "s/\"version\": \"${OLD_FE_VERSION}\"/\"version\": \"${NEW_VERSION}\"/" "${FE_FILE}"
  else
    sed -i "s/\"version\": \"${OLD_FE_VERSION}\"/\"version\": \"${NEW_VERSION}\"/" "${FE_FILE}"
  fi
  echo -e "  ${GREEN}✓${NC} frontend/package.json  ${OLD_FE_VERSION} → ${NEW_VERSION}"
else
  echo -e "  ${YELLOW}⚠${NC} frontend/package.json 未找到，跳过"
fi

# ── 更新 core/pyproject.toml ──────────────────────────────────
CORE_FILE="${ROOT_DIR}/core/pyproject.toml"
if [ -f "${CORE_FILE}" ]; then
  if [[ "$OSTYPE" == "darwin"* ]]; then
    sed -i '' "s/^version = \"${OLD_CORE_VERSION}\"/version = \"${NEW_VERSION}\"/" "${CORE_FILE}"
  else
    sed -i "s/^version = \"${OLD_CORE_VERSION}\"/version = \"${NEW_VERSION}\"/" "${CORE_FILE}"
  fi
  echo -e "  ${GREEN}✓${NC} core/pyproject.toml    ${OLD_CORE_VERSION} → ${NEW_VERSION}"
else
  echo -e "  ${YELLOW}⚠${NC} core/pyproject.toml 未找到，跳过"
fi

# ── 更新 README.md ────────────────────────────────────────────
README_ZH="${ROOT_DIR}/README.md"
if [ -f "${README_ZH}" ]; then
  # 替换 version badge: version-vX.Y.Z--suffix 或 version-vX.Y.Z
  if [[ "$OSTYPE" == "darwin"* ]]; then
    sed -i '' -E "s|version-v[0-9]+\.[0-9]+\.[0-9]+[^\"-]*|version-v${NEW_VERSION}|" "${README_ZH}"
    # 替换 status line 中的版本号
    sed -i '' -E "s|v[0-9]+\.[0-9]+\.[0-9]+[^，)]*|v${NEW_VERSION}|g" "${README_ZH}"
  else
    sed -i -E "s|version-v[0-9]+\.[0-9]+\.[0-9]+[^\"-]*|version-v${NEW_VERSION}|" "${README_ZH}"
    sed -i -E "s|v[0-9]+\.[0-9]+\.[0-9]+[^，)]*|v${NEW_VERSION}|g" "${README_ZH}"
  fi
  echo -e "  ${GREEN}✓${NC} README.md"
else
  echo -e "  ${YELLOW}⚠${NC} README.md 未找到，跳过"
fi

# ── 更新 README.en.md ─────────────────────────────────────────
README_EN="${ROOT_DIR}/README.en.md"
if [ -f "${README_EN}" ]; then
  if [[ "$OSTYPE" == "darwin"* ]]; then
    sed -i '' -E "s|version-v[0-9]+\.[0-9]+\.[0-9]+[^\"-]*|version-v${NEW_VERSION}|" "${README_EN}"
    sed -i '' -E "s|v[0-9]+\.[0-9]+\.[0-9]+[^ ,)]*|v${NEW_VERSION}|g" "${README_EN}"
  else
    sed -i -E "s|version-v[0-9]+\.[0-9]+\.[0-9]+[^\"-]*|version-v${NEW_VERSION}|" "${README_EN}"
    sed -i -E "s|v[0-9]+\.[0-9]+\.[0-9]+[^ ,)]*|v${NEW_VERSION}|g" "${README_EN}"
  fi
  echo -e "  ${GREEN}✓${NC} README.en.md"
else
  echo -e "  ${YELLOW}⚠${NC} README.en.md 未找到，跳过"
fi

echo ""
echo -e "${GREEN}============================================${NC}"
echo -e "${GREEN}  版本已更新为 v${NEW_VERSION}${NC}"
echo -e "${GREEN}============================================${NC}"
echo ""
echo "请检查以下文件的版本号是否正确:"
echo "  - frontend/package.json"
echo "  - core/pyproject.toml"
echo "  - README.md"
echo "  - README.en.md"
echo ""
echo "如状态标签需要单独更新（如 Beta → Stable），请手动修改 README 中的 status badge。"
