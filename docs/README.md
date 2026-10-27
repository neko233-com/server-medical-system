# server-medical-system 文档

> 面向医院的全球化医疗管理系统 · 全部功能集成于 Web · 单二进制部署

## 📚 文档索引

| 文档 | 内容 |
| --- | --- |
| [FEATURES.md](./FEATURES.md) | **全部功能清单**：诊疗、药品、预约、床位、收费、房间、打印、手术、影像、库存、运营分析、即时通讯 |
| [RBAC.md](./RBAC.md) | **权限体系**：32 项权限、7 种角色、权限矩阵与用户级授权 |
| [CHAT.md](./CHAT.md) | **院内即时通讯**：浮窗私聊 + 全院大群、图文、websocket、31 天 LRU、日志轮转 |
| [DEPLOYMENT.md](./DEPLOYMENT.md) | **一键部署与一行安装**：install.sh / install.ps1、deploy.sh / deploy.ps1、环境变量、端口、升级与运维 |
| [DATABASE.md](./DATABASE.md) | **数据库**：内置 SQLite 与外置 Postgres / MySQL 切换 |
| [API.md](./API.md) | **API 参考**：全部 REST 接口与权限要求 |

## ⚡ 30 秒上手

```bash
# 一行安装（无需克隆仓库）
curl -fsSL https://raw.githubusercontent.com/neko233-com/server-medical-system/main/install.sh | bash
powershell -ExecutionPolicy Bypass -c "irm https://raw.githubusercontent.com/neko233-com/server-medical-system/main/install.ps1 | iex"

# 或从源码一键构建并启动
# Windows (PowerShell)
.\deploy.ps1

# Linux / macOS / Git Bash
./deploy.sh
```

默认端口 **6970**，浏览器打开 http://localhost:6970 ，使用 `root` / `root` 登录。
发布物是**单个二进制**（前端 SPA 已通过 go:embed 编译在内），目标机器无需 Go / Node。

## 🏗 架构一览

```
浏览器 (React 19 + Vite SPA)
        │  /api/*
        ▼
Go 1.27 HTTP 服务 (Gin)
   ├── middleware : JWT 鉴权 → 接口级权限校验 (RequirePermission/RequireAny) → i18n
   ├── handlers   : 业务模块（患者/病例/处方/药品/预约/床位/收费/房间/打印机/手术/影像/运营/通讯/RBAC）
   ├── rbac       : 有效权限计算（角色权限 ∪ 用户授予 − 用户撤销）
   ├── models     : GORM 数据模型（21 + 张表）
   ├── chat       : WebSocket 中枢 + 31 天 LRU 留存引擎
   ├── logging    : 零依赖轮转日志
   └── db         : SQLite(默认) / Postgres / MySQL 自动迁移
```

两种运行形态完全一致：
- **生产**：Go 二进制同时托管 `/api` 与前端页面（优先使用 `go:embed` 内嵌的 SPA，其次回退到磁盘上的 `./web/dist`）。
- **开发**：`npm run dev` 跑 Vite（5173），`/api` 代理到后端 6970。

## 🔑 核心设计原则

1. **权限单一真相**：权限判定集中在后端中间件，前端 `can()` 仅负责 UI 显隐，二者共用同一套权限码，做到「看不到、也调不到」。
2. **跨数据库一致**：刻意不使用任何数据库方言 SQL（如 `strftime` / `date_trunc`），时间窗口在 Go 侧计算，确保 SQLite / Postgres / MySQL 行为一致。
3. **可审计**：打印打印任务（`PrintJob`）、影像上传均留痕，支持按医生/房间/患者回溯。
4. **零配置可用、全配置可换**：不设任何环境变量即可运行；重要部署项（库、密钥、端口、存储）全部支持环境变量覆盖。
5. **配置优先于环境猜测**：`.env` 中真实环境变量优先，因此容器 / systemd 注入的配置始终压过文件配置；文件查找顺序为 `ENV_FILE` → `./.env` → 可执行文件同目录的 `.env`。
6. **账目不可篡改**：已收款的结算单只能通过「退费」状态流转回退，不能编辑或删除。
