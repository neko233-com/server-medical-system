<#
.SYNOPSIS
    server-medical-system 一行命令安装（Windows）

.DESCRIPTION
    一行安装（无需先克隆仓库），在 PowerShell 中执行：

        powershell -ExecutionPolicy Bypass -c "irm https://raw.githubusercontent.com/neko233-com/server-medical-system/main/install.ps1 | iex"

.PARAMETER Port
    服务端口，默认 6970。

.PARAMETER InstallDir
    安装目录，默认 C:\Program Files\MedicalSystem。

.PARAMETER Version
    指定发布版本，默认 latest。

.PARAMETER FromSource
    从源码编译安装（需要 Go 与 Node）。

.NOTES
    安装内容：二进制 medical-system.exe、默认 .env、data 数据目录。
    Windows 服务 (-InstallService) 使用内置 sc.exe 注册。
#>
[CmdletBinding()]
param(
    [int]$Port = 6970,
    [string]$InstallDir = "C:\Program Files\MedicalSystem",
    [string]$Version = "latest",
    [switch]$FromSource,
    [switch]$InstallService
)

$ErrorActionPreference = 'Stop'
$Repo = 'neko233-com/server-medical-system'
$App  = 'medical-system.exe'

function Say($t)  { Write-Host "[install] $t" -ForegroundColor Cyan }
function Good($t) { Write-Host "[  ok  ] $t" -ForegroundColor Green }
function Note($t) { Write-Host "[ warn ] $t" -ForegroundColor Yellow }
function Bad($t)  { Write-Host "[ fail ] $t" -ForegroundColor Red; exit 1 }

Say "server-medical-system 安装器 (port=$Port → $InstallDir)"

# --------------------------------------------------------------------------- #
#  1. 准备目录
# --------------------------------------------------------------------------- #
$DataDir = Join-Path $InstallDir 'data'
foreach ($d in @($InstallDir, $DataDir)) {
    if (-not (Test-Path $d)) { New-Item -ItemType Directory -Path $d -Force | Out-Null }
}
Good "目录就绪: $InstallDir"

# --------------------------------------------------------------------------- #
#  2. 获取二进制：优先 release，其次源码编译
# --------------------------------------------------------------------------- #
$dest = Join-Path $InstallDir $App

if ($FromSource) {
    Say '从源码编译安装...'
    if (-not (Get-Command go -ErrorAction SilentlyContinue)) { Bad 'FROM_SOURCE 需要 Go，见 https://go.dev/dl' }
    $tmp = Join-Path $env:TEMP ("med-build-" + [guid]::NewGuid().ToString('N'))
    New-Item -ItemType Directory -Path $tmp -Force | Out-Null
    Push-Location $tmp
    try {
        Invoke-WebRequest "https://github.com/$Repo/archive/refs/heads/main.tar.gz" -OutFile src.tar.gz -UseBasicParsing
        tar xzf src.tar.gz
        Set-Location 'server-medical-system-main'
        # 构建顺序很重要：必须先构建前端（web/dist），Go 编译器才会把它嵌入二进制。
        if (Get-Command npm -ErrorAction SilentlyContinue) {
            Push-Location web
            try { npm ci --no-audit --no-fund; npm run build }
            catch { Note '前端构建跳过，二进制将仅提供 API' }
            finally { Pop-Location }
        }
        else {
            Note '未检测到 npm，跳过前端构建，二进制将仅提供 API'
        }
        $env:CGO_ENABLED = '0'; $env:GOOS = 'windows'; $env:GOARCH = 'amd64'
        & go build -trimpath '-ldflags=-s -w' -o $App ./cmd/server
        if ($LASTEXITCODE -ne 0) { Bad '编译失败' }
    }
    finally { Pop-Location }
    Copy-Item (Join-Path (Join-Path $tmp 'server-medical-system-main') $App) $dest -Force
    Remove-Item $tmp -Recurse -Force -ErrorAction SilentlyContinue
    Good '二进制安装完成（源码编译，含前端）'
}
else {
    Say "下载发布版 $Version ..."
    if ($Version -eq 'latest') {
        $url = "https://github.com/$Repo/releases/latest/download/$App"
    }
    else {
        $url = "https://github.com/$Repo/releases/download/$Version/$App"
    }
    try {
        Invoke-WebRequest $url -OutFile $dest -UseBasicParsing
        Good '二进制安装完成（release）'
    }
    catch {
        Note "未找到预编译包：$url"
        Note "请改用一行命令的源码编译方式："
        Note '  irm https://raw.githubusercontent.com/neko233-com/server-medical-system/main/install.ps1 | iex -ArgumentList ...'
        Bad  '或克隆仓库后执行 .\deploy.ps1'
    }
}

# --------------------------------------------------------------------------- #
#  3. 生成默认 .env（已存在则保留）
# --------------------------------------------------------------------------- #
$envFile = Join-Path $InstallDir '.env'
if (Test-Path $envFile) {
    Note "已存在 $envFile，保留原配置"
}
else {
    $secret = -join ((1..32) | ForEach-Object { '{0:x2}' -f (Get-Random -Maximum 256) })
    $dbPath = Join-Path $DataDir 'medical.db'
    $upDir  = Join-Path $DataDir 'uploads'
    $logDir = Join-Path $DataDir 'logs'
    @"
# server-medical-system 运行环境配置
PORT=$Port
# 生产务必替换（已随机生成）
JWT_SECRET=$secret
DB_DRIVER=sqlite
DB_PATH=$dbPath
UPLOAD_DIR=$upDir
LOG_DIR=$logDir
LOG_KEEP_DAYS=31
CHAT_RETENTION_DAYS=31
ALLOWED_ORIGINS=*
DEFAULT_LANG=zh-CN
"@ | Set-Content -Path $envFile -Encoding UTF8
    Good "已生成配置 $envFile"
}

# --------------------------------------------------------------------------- #
#  4. 可选：注册 Windows 服务
# --------------------------------------------------------------------------- #
if ($InstallService) {
    Say '注册 Windows 服务...'
    & sc.exe create MedicalSystem binPath= "`"$dest`"" start= auto DisplayName= "server-medical-system" | Out-Null
    & sc.exe start MedicalSystem | Out-Null
    Good '服务 MedicalSystem 已注册并启动'
    Write-Host '  查看： Get-EventLog -LogName Application -Source MedicalSystem' -ForegroundColor Yellow
    exit 0
}

Write-Host ''
Good '安装完成！'
Write-Host '---------------------------------------------------------------' -ForegroundColor Yellow
Write-Host "  启动： cd `"$InstallDir`"; .\$App"
Write-Host "  访问： http://localhost:$Port      默认账号 root / root"
Write-Host "  配置： $envFile"
Write-Host '---------------------------------------------------------------' -ForegroundColor Yellow
