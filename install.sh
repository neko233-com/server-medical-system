#!/usr/bin/env bash
# =============================================================================
#  server-medical-system — 一行命令安装（Linux / macOS）
# =============================================================================
#
#  一行安装（无需先克隆仓库）：
#     curl -fsSL https://raw.githubusercontent.com/neko233-com/server-medical-system/main/install.sh | bash
#
#  常用环境变量：
#     PORT=7000            # 服务端口（默认 6970）
#     INSTALL_DIR=/opt/medical   # 安装目录（默认 /opt/medical）
#     VERSION=v1.0.0       # 指定版本（默认 latest）
#     INSTALL_SERVICE=1    # 安装并启动 systemd 服务（Linux, root）
#     FROM_SOURCE=1        # 从源码编译安装（需 Go + Node）
#
#  安装内容：
#     <INSTALL_DIR>/medical-system    二进制（前端 SPA 已通过 go:embed 编译在内，
#                                     单独拷贝这一个文件即可完整运行）
#     <INSTALL_DIR>/.env              默认配置（PORT=6970 等）
#     <INSTALL_DIR>/data              数据目录（SQLite 库与影像上传）
#     /etc/systemd/system/medical.service  （可选）
#
# =============================================================================
set -euo pipefail

PORT="${PORT:-6970}"
INSTALL_DIR="${INSTALL_DIR:-/opt/medical}"
VERSION="${VERSION:-latest}"
WANT_SERVICE="${INSTALL_SERVICE:-0}"
FROM_SOURCE="${FROM_SOURCE:-0}"

REPO="neko233-com/server-medical-system"
APP="medical-system"

if [ -t 1 ]; then C=$'\033[36m'; G=$'\033[32m'; Y=$'\033[33m'; R=$'\033[31m'; N=$'\033[0m'; else C=""; G=""; Y=""; R=""; N=""; fi
step() { echo "${C}[install]${N} $*"; }
ok()   { echo "${G}[  ok  ]${N} $*"; }
warn() { echo "${Y}[ warn ]${N} $*"; }
die()  { echo "${R}[ fail ]${N} $*" >&2; exit 1; }

step "server-medical-system 安装器 (port=${PORT} → ${INSTALL_DIR})"

# --------------------------------------------------------------------------- #
#  1. 探测目标平台
# --------------------------------------------------------------------------- #
OS="$(uname -s)"; ARCH="$(uname -m)"
case "$OS" in Linux) GOOS=linux ;; Darwin) GOOS=darwin ;; *) die "不支持的系统: $OS" ;; esac
case "$ARCH" in
  x86_64|amd64) GOARCH=amd64 ;;
  arm64|aarch64) GOARCH=arm64 ;;
  *) die "不支持的架构: $ARCH" ;;
esac
if command -v sudo >/dev/null 2>&1 && [ "$(id -u)" -ne 0 ]; then SUDO=sudo; else SUDO=""; fi
$SUDO mkdir -p "$INSTALL_DIR" "$INSTALL_DIR/data" || die "无法创建安装目录 $INSTALL_DIR"
ok "平台: ${GOOS}/${GOARCH}"

# --------------------------------------------------------------------------- #
#  2. 获取二进制：优先 release，其次源码编译
# --------------------------------------------------------------------------- #
install_binary() {
  if [ "$FROM_SOURCE" = "1" ]; then
    step "从源码编译安装..."
    command -v go >/dev/null 2>&1 || die "FROM_SOURCE=1 需要 Go，见 https://go.dev/dl"
    TMP="$(mktemp -d)"
    # 注意构建顺序：必须先构建前端（web/dist），Go 编译器才会把它嵌入二进制。
    ( cd "$TMP" && curl -fsSL "https://github.com/${REPO}/archive/refs/heads/main.tar.gz" \
        | tar xz && cd server-medical-system-main \
        && ( cd web && (command -v npm >/dev/null 2>&1 && npm ci --no-audit --no-fund && npm run build \
             || echo "warn: 前端构建跳过，二进制将仅提供 API" >&2) ) \
        && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o "$APP" ./cmd/server ) || die "源码编译失败"
    $SUDO install -m 0755 "$TMP/server-medical-system-main/$APP" "$INSTALL_DIR/$APP"
    rm -rf "$TMP"
    ok "二进制安装完成（源码编译，含前端）"
    return
  fi

  step "下载发布版 $VERSION ..."
  if [ "$VERSION" = "latest" ]; then
    url="https://github.com/${REPO}/releases/latest/download/${APP}-${GOOS}-${GOARCH}"
  else
    url="https://github.com/${REPO}/releases/download/${VERSION}/${APP}-${GOOS}-${GOARCH}"
  fi
  TMPBIN="$(mktemp)"
  if curl -fsSL "$url" -o "$TMPBIN" 2>/dev/null && [ -s "$TMPBIN" ]; then
    $SUDO install -m 0755 "$TMPBIN" "$INSTALL_DIR/$APP"
    rm -f "$TMPBIN"
    ok "二进制安装完成（release）"
  else
    rm -f "$TMPBIN"
    warn "未找到对应平台的预编译包，回退到源码编译"
    FROM_SOURCE=1
    install_binary
  fi
}
install_binary

# --------------------------------------------------------------------------- #
#  3. 生成默认 .env（已存在则保留，绝不覆盖用户配置）
# --------------------------------------------------------------------------- #
ENV_FILE_TARGET="$INSTALL_DIR/.env"
if [ -f "$ENV_FILE_TARGET" ]; then
  warn "已存在 $ENV_FILE_TARGET，保留原配置"
else
  $SUDO tee "$ENV_FILE_TARGET" >/dev/null <<EOF
# server-medical-system 运行环境配置
# 修改后重启服务生效： sudo systemctl restart medical

PORT=${PORT}
# 生产务必替换： openssl rand -hex 32
JWT_SECRET=$(openssl rand -hex 32 2>/dev/null || head -c 32 /dev/urandom | od -An -tx1 | tr -d ' \n')
DB_DRIVER=sqlite
DB_PATH=${INSTALL_DIR}/data/medical.db
UPLOAD_DIR=${INSTALL_DIR}/data/uploads
LOG_DIR=${INSTALL_DIR}/data/logs
LOG_KEEP_DAYS=31
CHAT_RETENTION_DAYS=31
ALLOWED_ORIGINS=*
DEFAULT_LANG=zh-CN
EOF
  $SUDO chmod 600 "$ENV_FILE_TARGET"
  ok "已生成配置 $ENV_FILE_TARGET"
fi

# 数据目录归属（非 root 安装时保证可写）
$SUDO chmod 755 "$INSTALL_DIR/data" 2>/dev/null || true

# --------------------------------------------------------------------------- #
#  4. 可选：注册 systemd 服务
# --------------------------------------------------------------------------- #
if [ "$WANT_SERVICE" = "1" ]; then
  if [ "$OS" != "Linux" ]; then
    warn "systemd 仅 Linux 支持，跳过服务安装"
  else
    $SUDO tee /etc/systemd/system/medical.service >/dev/null <<EOF
[Unit]
Description=server-medical-system
After=network.target

[Service]
Type=simple
WorkingDirectory=${INSTALL_DIR}
EnvironmentFile=${ENV_FILE_TARGET}
ExecStart=${INSTALL_DIR}/${APP}
Restart=always
RestartSec=3
NoNewPrivileges=true

[Install]
WantedBy=multi-user.target
EOF
    $SUDO systemctl daemon-reload
    $SUDO systemctl enable --now medical
    ok "systemd 服务已启用并启动"
    step "查看日志： sudo journalctl -u medical -f"
    exit 0
  fi
fi

echo
ok "安装完成！"
echo "${Y}---------------------------------------------------------------${N}"
echo "  启动： cd ${INSTALL_DIR} && ./${APP}"
echo "  或  ： PORT=${PORT} ${INSTALL_DIR}/${APP}"
echo "  访问： http://localhost:${PORT}      默认账号 root / root"
echo "  配置： ${ENV_FILE_TARGET}"
echo "${Y}---------------------------------------------------------------${N}"
