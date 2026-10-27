#!/usr/bin/env bash
# =============================================================================
#  server-medical-system — 一键部署脚本 (Linux / macOS / Git Bash)
# =============================================================================
#
#  用途：一条命令完成「前端构建 → 后端编译 → 启动服务」，产出单二进制 + web/dist。
#
#  用法：
#     ./deploy.sh                 # 构建并以默认端口 6970 启动
#     ./deploy.sh 8080            # 指定端口启动
#     BUILD_ONLY=1 ./deploy.sh    # 只构建，不启动
#     SKIP_WEB=1  ./deploy.sh     # 跳过前端构建（web/dist 已存在时）
#
#  环境变量（均有默认值，见 README「环境变量」章节）：
#     PORT  DB_DRIVER  DB_DSN  JWT_SECRET  UPLOAD_DIR  SEED_DEMO  ALLOWED_ORIGINS
#
# =============================================================================

set -euo pipefail

# --------------------------------------------------------------------------- #
#  配置区
# --------------------------------------------------------------------------- #
PORT="${1:-${PORT:-6970}}"          # 默认端口 6970（命令行参数优先，其次环境变量）
BUILD_ONLY="${BUILD_ONLY:-0}"       # 设为 1 则只构建不启动
SKIP_WEB="${SKIP_WEB:-0}"           # 设为 1 则跳过前端构建

APP_NAME="medical-system"           # 产物二进制名称
CGO_ENABLED_VALUE=0                 # 纯 Go 静态编译，保证跨平台免依赖分发

# 仓库根目录（脚本所在目录）
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# 颜色输出（非 TTY 时自动降级为无颜色）
if [ -t 1 ]; then
  RED=$'\033[31m'; GREEN=$'\033[32m'; YELLOW=$'\033[33m'; BLUE=$'\033[34m'; NC=$'\033[0m'
else
  RED=""; GREEN=""; YELLOW=""; BLUE=""; NC=""
fi

log()  { echo "${BLUE}[deploy]${NC} $*"; }
ok()   { echo "${GREEN}[  OK  ]${NC} $*"; }
warn() { echo "${YELLOW}[ warn ]${NC} $*"; }
die()  { echo "${RED}[ fail ]${NC} $*" >&2; exit 1; }

log "医疗管理系统 · 一键部署 (port=${PORT})"

# --------------------------------------------------------------------------- #
#  1. 依赖检查
# --------------------------------------------------------------------------- #
command -v go  >/dev/null 2>&1 || die "未检测到 Go，请先安装 Go 1.27+：https://go.dev/dl"
ok "Go 已就绪: $(go version | awk '{print $3}')"

if [ "$SKIP_WEB" != "1" ]; then
  command -v node >/dev/null 2>&1 || die "未检测到 Node.js，请安装 Node 18+ 或设置 SKIP_WEB=1"
  command -v npm  >/dev/null 2>&1 || die "未检测到 npm，请安装 Node 18+ 或设置 SKIP_WEB=1"
  ok "Node 已就绪: $(node -v) (npm $(npm -v))"
fi

cd "$ROOT_DIR"

# --------------------------------------------------------------------------- #
#  2. 构建前端 SPA (React 19 + Vite → web/dist)
# --------------------------------------------------------------------------- #
if [ "$SKIP_WEB" = "1" ]; then
  warn "跳过前端构建 (SKIP_WEB=1)"
  if [ ! -f "web/dist/index.html" ]; then
    warn "警告：web/dist/index.html 不存在，编译出的二进制将仅提供 API（无前端界面）"
  else
    ok "复用已有的 web/dist"
  fi
else
  log "构建前端..."
  if [ ! -d "web/node_modules" ]; then
    ( cd web && npm install --no-audit --no-fund ) || die "npm install 失败"
  fi
  ( cd web && npm run build ) || die "前端构建失败"
  ok "前端构建完成 → web/dist"
fi

# --------------------------------------------------------------------------- #
#  3. 构建后端二进制（静态链接，前端 SPA 通过 go:embed 编译进二进制）
#     -trimpath 移除本地绝对路径，保证构建可复现；
#     -ldflags 精简符号表与 DWARF，显著减小体积（打包优化）。
# --------------------------------------------------------------------------- #
log "构建后端二进制..."
# web/dist 必须先于 go build 存在，否则嵌入的前端为空。
CGO_ENABLED=$CGO_ENABLED_VALUE go build \
  -trimpath \
  -ldflags="-s -w" \
  -o "$APP_NAME" \
  ./cmd/server || die "后端编译失败"
ok "后端构建完成 → ./$APP_NAME（单文件即含前端，拷贝该文件即可部署）"

# 确保运行时目录存在（影像样片存放于 ./uploads）
mkdir -p uploads

if [ "$BUILD_ONLY" = "1" ]; then
  ok "BUILD_ONLY=1，跳过启动。可执行 PORT=$PORT ./$APP_NAME 启动。"
  exit 0
fi

# --------------------------------------------------------------------------- #
#  4. 启动服务
#     前端由二进制托管：访问 http://localhost:${PORT}
#     API 基址：        http://localhost:${PORT}/api
# --------------------------------------------------------------------------- #
export PORT="$PORT"
log "启动服务..."

# 若端口已被占用，提前给出清晰提示而不是崩溃
if command -v lsof >/dev/null 2>&1 && lsof -i ":${PORT}" >/dev/null 2>&1; then
  warn "端口 ${PORT} 已被占用，新实例可能无法启动。可换端口：./deploy.sh 7001"
fi

echo
ok "服务即将启动：http://localhost:${PORT}"
echo "${YELLOW}---------------------------------------------------------------${NC}"
echo "  默认账号： root / root              （首次部署后请立即修改密码）"
echo "  停止服务： Ctrl+C"
echo "${YELLOW}---------------------------------------------------------------${NC}"
echo

exec "./$APP_NAME"
