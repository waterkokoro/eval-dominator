#!/usr/bin/env bash
# ============================================================
#  Eval-Dominator 统一启动/停止/重启/状态管理脚本
#  用法:
#    ./scripts/start.sh                        # 启动全部（默认）
#    ./scripts/start.sh start    [all|frontend|backend|core]
#    ./scripts/start.sh stop     [all|frontend|backend|core]
#    ./scripts/start.sh restart  [all|frontend|backend|core]
#    ./scripts/start.sh status
#    ./scripts/start.sh log      [frontend|backend|core]   # 查看日志
# ============================================================
set -uo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SCRIPTS_DIR="${ROOT_DIR}/scripts"
PID_DIR="${ROOT_DIR}/runtime/pids"
LOG_DIR="${ROOT_DIR}/runtime/logs"

# ── 颜色 ──────────────────────────────────────────────────────
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# ── 工具函数 ──────────────────────────────────────────────────

ensure_dirs() {
  mkdir -p "${PID_DIR}" "${LOG_DIR}/frontend" "${LOG_DIR}/backend" "${LOG_DIR}/core"
}

pid_file() {
  echo "${PID_DIR}/${1}.pid"
}

is_running() {
  local pf
  pf="$(pid_file "$1")"
  if [ -f "${pf}" ]; then
    local pid
    pid="$(cat "${pf}")"
    if kill -0 "${pid}" 2>/dev/null; then
      return 0
    fi
    # 进程已不存在，清理 pid 文件
    rm -f "${pf}"
  fi
  return 1
}

get_pid() {
  local pf
  pf="$(pid_file "$1")"
  if [ -f "${pf}" ]; then
    cat "${pf}"
  else
    echo ""
  fi
}

# 从配置文件读取端口号，读不到就用默认值
get_backend_port() {
  local cfg="${ROOT_DIR}/backend/config/config.yaml"
  if [ -f "${cfg}" ]; then
    local p
    p="$(grep -E '^\s*port:' "${cfg}" | head -1 | awk '{print $2}' | tr -d '"' | tr -d "'")"
    if [ -n "${p}" ]; then echo "${p}"; return; fi
  fi
  echo "8080"
}

get_core_port() {
  local cfg="${ROOT_DIR}/core/config/config.yaml"
  if [ -f "${cfg}" ]; then
    local p
    p="$(grep -E '^\s*port:' "${cfg}" | head -1 | awk '{print $2}' | tr -d '"' | tr -d "'")"
    if [ -n "${p}" ]; then echo "${p}"; return; fi
  fi
  echo "50051"
}

# 从 Vue CLI 日志中抓取实际监听端口（因为它会自动避开被占用的端口）
get_frontend_port() {
  local log="${LOG_DIR}/frontend/stdout.log"
  if [ -f "${log}" ]; then
    local p
    p="$(grep -oE 'http://[^/:]+:([0-9]+)' "${log}" | tail -1 | grep -oE '[0-9]+$')"
    if [ -n "${p}" ]; then echo "${p}"; return; fi
  fi
  echo "8081"
}

# 等服务端口就绪（最多等 N 秒）
wait_for_port() {
  local host="$1" port="$2" timeout="$3" label="$4"
  local waited=0
  while ! (echo > /dev/tcp/${host}/${port}) 2>/dev/null && [ "${waited}" -lt "${timeout}" ]; do
    sleep 1
    waited=$((waited + 1))
  done
  if (echo > /dev/tcp/${host}/${port}) 2>/dev/null; then
    return 0
  fi
  return 1
}

show_access_urls() {
  local fe_port be_port core_port
  fe_port="$(get_frontend_port)"
  be_port="$(get_backend_port)"
  core_port="$(get_core_port)"

  echo ""
  echo -e "${GREEN}============================================${NC}"
  echo -e "${GREEN}       服务启动完成 - 访问链接${NC}"
  echo -e "${GREEN}============================================${NC}"

  if is_running "frontend"; then
    echo -e "  ${GREEN}●${NC} Frontend (Vue.js)    ${BLUE}http://127.0.0.1:${fe_port}${NC}"
  else
    echo -e "  ${RED}●${NC} Frontend (Vue.js)    未运行"
  fi

  if is_running "backend"; then
    echo -e "  ${GREEN}●${NC} Backend  (HTTP API)  ${BLUE}http://127.0.0.1:${be_port}/api${NC}"
  else
    echo -e "  ${RED}●${NC} Backend  (HTTP API)  未运行"
  fi

  if is_running "core"; then
    echo -e "  ${GREEN}●${NC} Core     (gRPC)      ${BLUE}127.0.0.1:${core_port}${NC}"
  else
    echo -e "  ${RED}●${NC} Core     (gRPC)      未运行"
  fi

  echo -e "${GREEN}============================================${NC}"
  echo ""
}

# ── 启动函数 ──────────────────────────────────────────────────

start_frontend() {
  if is_running "frontend"; then
    echo -e "${YELLOW}[frontend]${NC} 已在运行 (PID $(get_pid frontend))"
    return 0
  fi

  echo -e "${BLUE}[frontend]${NC} 启动中..."
  cd "${ROOT_DIR}/frontend"

  if [ ! -f ".env.development" ]; then
    cp ".env.development.example" ".env.development"
    echo -e "${YELLOW}[frontend]${NC} 已创建 .env.development，请按需修改后端 API 地址。"
  fi

  npm install --silent 2>&1 | tail -1

  nohup npm run serve \
    > "${LOG_DIR}/frontend/stdout.log" 2>&1 &
  echo $! > "$(pid_file frontend)"
  echo -e "${GREEN}[frontend]${NC} 已启动 (PID $(get_pid frontend))，日志: ${LOG_DIR}/frontend/stdout.log"
}

start_backend() {
  if is_running "backend"; then
    echo -e "${YELLOW}[backend]${NC} 已在运行 (PID $(get_pid backend))"
    return 0
  fi

  echo -e "${BLUE}[backend]${NC} 启动中..."
  cd "${ROOT_DIR}/backend"

  if [ ! -f "config/config.yaml" ]; then
    cp "config/config.example.yaml" "config/config.yaml"
    echo -e "${YELLOW}[backend]${NC} 已创建 config/config.yaml，请按需修改。"
  fi

  GOTOOLCHAIN=local go mod tidy 2>&1 | tail -1

  # 先编译再运行，确保 PID 对应真实 server 进程（go run 会产生子进程导致 kill 不干净）
  local bin="${ROOT_DIR}/runtime/backend-server"
  GOTOOLCHAIN=local go build -o "${bin}" ./cmd/server 2>&1 | tail -3

  nohup "${bin}" \
    --config config/config.yaml \
    --migration migrations/001_init.sql \
    > "${LOG_DIR}/backend/stdout.log" 2>&1 &
  echo $! > "$(pid_file backend)"
  echo -e "${GREEN}[backend]${NC} 已启动 (PID $(get_pid backend))，日志: ${LOG_DIR}/backend/stdout.log"
}

start_core() {
  if is_running "core"; then
    echo -e "${YELLOW}[core]${NC} 已在运行 (PID $(get_pid core))"
    return 0
  fi

  echo -e "${BLUE}[core]${NC} 启动中..."
  cd "${ROOT_DIR}/core"

  local VENV_PYTHON="${ROOT_DIR}/core/.venv/bin/python"

  if [ ! -f "config/config.yaml" ]; then
    cp "config/config.example.yaml" "config/config.yaml"
    echo -e "${YELLOW}[core]${NC} 已创建 config/config.yaml，请按需修改。"
  fi

  if [ ! -x "${VENV_PYTHON}" ]; then
    echo -e "${RED}[core]${NC} 未找到 core/.venv，请先执行: ./scripts/init_core_venv.sh"
    return 1
  fi

  "${VENV_PYTHON}" scripts/generate_proto.py

  # 注入环境变量（与 start_core.sh 保持一致）
  export PATH="$(dirname "${VENV_PYTHON}"):${PATH}"
  export GRPC_ENABLE_FORK_SUPPORT="${GRPC_ENABLE_FORK_SUPPORT:-0}"
  export GRPC_VERBOSITY="${GRPC_VERBOSITY:-error}"
  export HF_HUB_OFFLINE="${HF_HUB_OFFLINE:-1}"
  export TRANSFORMERS_OFFLINE="${TRANSFORMERS_OFFLINE:-1}"

  nohup env PYTHONPATH=src "${VENV_PYTHON}" -m opencompass_core \
    --config config/config.yaml \
    > "${LOG_DIR}/core/stdout.log" 2>&1 &
  echo $! > "$(pid_file core)"
  echo -e "${GREEN}[core]${NC} 已启动 (PID $(get_pid core))，日志: ${LOG_DIR}/core/stdout.log"
}

# ── 停止函数 ──────────────────────────────────────────────────

stop_service() {
  local name="$1"
  if ! is_running "${name}"; then
    # PID 文件不存在，但端口可能仍被占用（孤儿进程），按端口兜底清理
    _kill_orphan_by_port "${name}"
    echo -e "${YELLOW}[${name}]${NC} 未在运行"
    return 0
  fi

  local pid
  pid="$(get_pid "${name}")"
  echo -e "${BLUE}[${name}]${NC} 停止中 (PID ${pid})..."

  # 先 SIGTERM，等待最多 10 秒
  kill "${pid}" 2>/dev/null || true
  local waited=0
  while kill -0 "${pid}" 2>/dev/null && [ "${waited}" -lt 10 ]; do
    sleep 1
    waited=$((waited + 1))
  done

  # 还没退出就 SIGKILL
  if kill -0 "${pid}" 2>/dev/null; then
    echo -e "${YELLOW}[${name}]${NC} 强制终止..."
    kill -9 "${pid}" 2>/dev/null || true
    sleep 1
  fi

  rm -f "$(pid_file "${name}")"

  # 兜底：按端口清理孤儿进程
  _kill_orphan_by_port "${name}"

  echo -e "${GREEN}[${name}]${NC} 已停止"
}

# 按端口号兜底杀掉孤儿进程（解决 go run 等场景下子进程残留）
_kill_orphan_by_port() {
  local name="$1"
  local port=""
  case "${name}" in
    backend)  port="$(get_backend_port)" ;;
    core)     port="$(get_core_port)" ;;
    *)        return ;;
  esac

  local pids
  pids="$(lsof -ti :"${port}" 2>/dev/null || true)"
  if [ -n "${pids}" ]; then
    echo -e "${YELLOW}[${name}]${NC} 发现端口 ${port} 孤儿进程，清理中..."
    echo "${pids}" | xargs kill -9 2>/dev/null || true
    sleep 1
  fi
}

# ── 状态函数 ──────────────────────────────────────────────────

show_status() {
  echo ""
  echo -e "${BLUE}========== Eval-Dominator 服务状态 ==========${NC}"
  for svc in frontend backend core; do
    if is_running "${svc}"; then
      echo -e "  ${GREEN}●${NC} ${svc}  运行中  (PID $(get_pid "${svc}"))"
    else
      echo -e "  ${RED}●${NC} ${svc}  未运行"
    fi
  done
  echo ""
}

# ── 日志函数 ──────────────────────────────────────────────────

show_log() {
  local name="$1"
  local log_file="${LOG_DIR}/${name}/stdout.log"
  if [ ! -f "${log_file}" ]; then
    echo -e "${RED}[${name}]${NC} 日志文件不存在: ${log_file}"
    return 1
  fi
  echo -e "${BLUE}[${name}]${NC} 显示最近 50 行日志 (${log_file})："
  echo "---"
  tail -n 50 "${log_file}"
}

# ── 组合操作 ──────────────────────────────────────────────────

do_start() {
  local target="${1:-all}"
  ensure_dirs
  case "${target}" in
    all)
      echo -e "${BLUE}启动全部服务...${NC}"
      start_frontend
      # 先启动 core，并等其 gRPC 端口就绪再启动 backend，避免 backend dial 超时退出
      start_core
      local core_port be_port
      core_port="$(get_core_port)"
      if ! wait_for_port "127.0.0.1" "${core_port}" 30 "core"; then
        echo -e "${YELLOW}[core]${NC} 端口 ${core_port} 30 秒内未就绪，仍继续启动 backend（可能失败）"
      fi
      # gRPC 服务监听后还需要一小段时间完成内部初始化，给 backend 留出余量
      sleep 1
      start_backend
      echo ""
      echo -e "${BLUE}等待服务就绪...${NC}"
      be_port="$(get_backend_port)"
      wait_for_port "127.0.0.1" "${be_port}" 30 "backend" || true
      sleep 3  # 额外等前端 Vue CLI 编译完成
      show_access_urls
      ;;
    frontend)
      "start_${target}"
      local fe_port
      sleep 2
      fe_port="$(get_frontend_port)"
      echo ""
      echo -e "  ${GREEN}●${NC} Frontend  ${BLUE}http://127.0.0.1:${fe_port}${NC}"
      ;;
    backend)
      "start_${target}"
      local be_port
      be_port="$(get_backend_port)"
      wait_for_port "127.0.0.1" "${be_port}" 30 "backend" || true
      echo ""
      echo -e "  ${GREEN}●${NC} Backend  ${BLUE}http://127.0.0.1:${be_port}/api${NC}"
      ;;
    core)
      "start_${target}"
      local core_port
      core_port="$(get_core_port)"
      wait_for_port "127.0.0.1" "${core_port}" 20 "core" || true
      echo ""
      echo -e "  ${GREEN}●${NC} Core  ${BLUE}127.0.0.1:${core_port}${NC}"
      ;;
    *)
      echo -e "${RED}未知服务: ${target}${NC}"
      echo "可选: all | frontend | backend | core"
      exit 1
      ;;
  esac
}

do_stop() {
  local target="${1:-all}"
  case "${target}" in
    all)
      echo -e "${BLUE}停止全部服务...${NC}"
      stop_service "frontend"
      stop_service "backend"
      stop_service "core"
      echo ""
      show_status
      ;;
    frontend|backend|core)
      stop_service "${target}"
      ;;
    *)
      echo -e "${RED}未知服务: ${target}${NC}"
      echo "可选: all | frontend | backend | core"
      exit 1
      ;;
  esac
}

do_restart() {
  local target="${1:-all}"
  do_stop "${target}"
  echo ""
  do_start "${target}"
}

# ── 主入口 ────────────────────────────────────────────────────

usage() {
  cat <<EOF
Eval-Dominator 服务管理脚本

用法:
  $(basename "$0") <命令> [服务名]

命令:
  start     启动服务（默认 all）
  stop      停止服务（默认 all）
  restart   重启服务（默认 all）
  status    查看所有服务状态
  log       查看服务日志（需指定服务名）

服务名:
  all       全部三个服务（仅 start/stop/restart 可用）
  frontend  前端 (Vue.js)
  backend   后端 (Go)
  core      评测核心 (Python gRPC)

示例:
  $(basename "$0")                       # 启动全部服务
  $(basename "$0") start                 # 启动全部服务
  $(basename "$0") start backend         # 仅启动后端
  $(basename "$0") stop core             # 停止评测核心
  $(basename "$0") restart all           # 重启全部
  $(basename "$0") status                # 查看状态
  $(basename "$0") log frontend          # 查看前端日志
EOF
}

main() {
  local cmd="${1:-start}"
  local target="${2:-all}"

  case "${cmd}" in
    start)
      do_start "${target}"
      ;;
    stop)
      do_stop "${target}"
      ;;
    restart)
      do_restart "${target}"
      ;;
    status)
      ensure_dirs
      show_status
      ;;
    log)
      if [ "${target}" = "all" ]; then
        echo -e "${RED}log 命令需要指定服务名: frontend | backend | core${NC}"
        exit 1
      fi
      ensure_dirs
      show_log "${target}"
      ;;
    help|-h|--help)
      usage
      ;;
    *)
      echo -e "${RED}未知命令: ${cmd}${NC}"
      usage
      exit 1
      ;;
  esac
}

main "$@"
