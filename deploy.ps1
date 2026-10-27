<#
.SYNOPSIS
    server-medical-system 一键部署脚本 (Windows PowerShell)

.DESCRIPTION
    一条命令完成「前端构建 → 后端编译 → 启动服务」，产出单二进制 + web/dist。

.PARAMETER Port
    服务监听端口，默认 6970。

.PARAMETER BuildOnly
    仅构建，不启动服务。

.PARAMETER SkipWeb
    跳过前端构建（web/dist 已存在时使用）。

.EXAMPLE
    .\deploy.ps1
    构建并以默认端口 6970 启动。

.EXAMPLE
    .\deploy.ps1 -Port 7001 -BuildOnly
    仅构建，端口信息仅用于提示。

.NOTES
    环境变量：PORT DB_DRIVER DB_DSN JWT_SECRET UPLOAD_DIR SEED_DEMO ALLOWED_ORIGINS
    详见 README「环境变量」章节。
#>
[CmdletBinding()]
param(
    [int]$Port = 6970,
    [switch]$BuildOnly,
    [switch]$SkipWeb
)

$ErrorActionPreference = 'Stop'

$AppName   = 'medical-system.exe'
$RootDir   = Split-Path -Parent $MyInvocation.MyCommand.Path

function Write-Prefix($Text, $Color) {
    Write-Host "[deploy] " -ForegroundColor $Color -NoNewline
    Write-Host $Text
}
function Step($Text)  { Write-Prefix $Text Cyan }
function Good($Text)  { Write-Host "[  OK  ] $Text" -ForegroundColor Green }
function Note($Text)  { Write-Host "[ info ] $Text" -ForegroundColor Yellow }
function Fail($Text)  { Write-Host "[ fail ] $Text" -ForegroundColor Red; exit 1 }

Step "医疗管理系统 · 一键部署 (port=$Port)"

# --------------------------------------------------------------------------- #
#  1. 依赖检查
# --------------------------------------------------------------------------- #
if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    Fail "未检测到 Go，请先安装 Go 1.27+ ： https://go.dev/dl"
}
Good "Go 已就绪: $((go version) -replace 'go version ','')"

if (-not $SkipWeb) {
    if (-not (Get-Command node -ErrorAction SilentlyContinue)) {
        Fail "未检测到 Node.js，请安装 Node 18+ 或使用 -SkipWeb"
    }
    if (-not (Get-Command npm -ErrorAction SilentlyContinue)) {
        Fail "未检测到 npm，请安装 Node 18+ 或使用 -SkipWeb"
    }
    Good "Node 已就绪: $(node -v)"
}

Set-Location $RootDir

# --------------------------------------------------------------------------- #
#  2. 构建前端 SPA (React 19 + Vite -> web/dist)
# --------------------------------------------------------------------------- #
if ($SkipWeb) {
    Note "跳过前端构建 (-SkipWeb)"
    if (-not (Test-Path 'web\dist')) { Note "注意：web\dist 不存在，部署后页面可能无法显示" }
}
else {
    Step "构建前端..."
    if (-not (Test-Path 'web\node_modules')) {
        Push-Location web
        try { npm install --no-audit --no-fund; if ($LASTEXITCODE -ne 0) { Fail 'npm install 失败' } }
        finally { Pop-Location }
    }
    Push-Location web
    try { npm run build; if ($LASTEXITCODE -ne 0) { Fail '前端构建失败' } }
    finally { Pop-Location }
    Good "前端构建完成 -> web\dist"
}

# --------------------------------------------------------------------------- #
#  3. 构建后端二进制
#     -trimpath 移除本地绝对路径（可复现构建）
#     -ldflags -s -w 精简符号表与 DWARF，减小体积（打包优化）
# --------------------------------------------------------------------------- #
Step "构建后端二进制..."
$env:CGO_ENABLED = '0'
$env:GOOS   = 'windows'
$env:GOARCH = 'amd64'

$buildArgs = @(
    'build', '-trimpath', '-ldflags=-s -w',
    '-o', $AppName, './cmd/server'
)
& go @buildArgs
if ($LASTEXITCODE -ne 0) { Fail '后端编译失败' }
Good "后端构建完成 -> .\$AppName"

# 确保运行时目录存在（影像样片存放于 .\uploads）
New-Item -ItemType Directory -Force -Path 'uploads' | Out-Null

if ($BuildOnly) {
    Good "BuildOnly 模式，跳过启动。可执行： `$env:PORT=$Port; .\$AppName"
    exit 0
}

# --------------------------------------------------------------------------- #
#  4. 启动服务
# --------------------------------------------------------------------------- #
$env:PORT = "$Port"
Step "启动服务..."

$inUse = Get-NetTCPConnection -LocalPort $Port -State Listen -ErrorAction SilentlyContinue
if ($inUse) { Note "端口 $Port 已被占用，新实例可能无法启动。可换端口： .\deploy.ps1 -Port 7001" }

Write-Host ''
Good "服务即将启动： http://localhost:$Port"
Write-Host '---------------------------------------------------------------' -ForegroundColor Yellow
Write-Host '  默认账号： admin / admin123          （首次部署后请立即修改密码）'
Write-Host '  停止服务： Ctrl+C'
Write-Host '---------------------------------------------------------------' -ForegroundColor Yellow
Write-Host ''

& ".\$AppName"
