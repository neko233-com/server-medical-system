# 部署指南 · DEPLOYMENT

## 一、一键部署（推荐）

默认端口 **6970**（刻意避开常见的 8080 / 3000，减少冲突）。

### Windows（PowerShell）

```powershell
.\deploy.ps1                # 构建并以 6970 启动
.\deploy.ps1 -Port 7001     # 指定端口
.\deploy.ps1 -BuildOnly     # 仅构建不启动
.\deploy.ps1 -SkipWeb       # 跳过前端构建（web/dist 已存在）
```

### Linux / macOS / Git Bash

```bash
./deploy.sh                 # 构建并以 6970 启动
./deploy.sh 8080            # 指定端口
BUILD_ONLY=1 ./deploy.sh    # 仅构建不启动
SKIP_WEB=1   ./deploy.sh    # 跳过前端构建
```

脚本依次完成：**依赖检查 → 构建前端 → 编译后端（静态链接 + 体积裁剪）→ 启动服务**。

产出：
- `./medical-system`（Windows 为 `medical-system.exe`）—— 单二进制，可直接拷贝部署
- `./web/dist` —— 构建好的前端静态资源，由二进制托管

> 仅需把这两个产物与 `uploads/`（运行时自动生成）放在同一目录即可运行，无需安装 Go / Node。

## 二、手动部署

```bash
# 1) 构建前端
cd web && npm install && npm run build && cd ..

# 2) 编译后端（静态链接、去除符号与 DWARF，体积更小）
CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o medical-system ./cmd/server

# 3) 运行
PORT=6970 ./medical-system
```

访问 http://localhost:6970 ，登录 `admin` / `admin123`。

## 三、开发模式（前后端分离）

```bash
# 终端 1：后端（6970）
PORT=6970 go run ./cmd/server

# 终端 2：前端（5173，/api 自动代理到 6970）
cd web && npm run dev
```

代理配置见 `web/vite.config.js`（`BACKEND_PORT` 常量，须与后端端口一致）。

## 四、环境变量

复制 `.env.example` 为 `.env` 后按需修改（`.env` 已被 git 忽略）：

| 变量 | 默认值 | 说明 |
| --- | --- | --- |
| `PORT` | `6970` | HTTP 端口 |
| `JWT_SECRET` | 内置开发值 | **生产必须替换**（`openssl rand -hex 32`） |
| `DB_DRIVER` | `sqlite` | `sqlite` / `postgres` / `mysql` |
| `DB_PATH` | `medical.db` | SQLite 文件路径 |
| `DB_DSN` | — | Postgres / MySQL 连接串 |
| `UPLOAD_DIR` | `./uploads` | 影像样片存储目录 |
| `SEED_DEMO` | `true` | 是否注入演示数据 |
| `ALLOWED_ORIGINS` | `*` | CORS 白名单，生产请填具体域名 |
| `DEFAULT_LANG` | `zh-CN` | 回退语言 |
| `LOG_DIR` | `./logs` | 轮转日志目录 |
| `LOG_MAX_MB` | `64` | 单日志大小上限，超出自动分片 |
| `LOG_KEEP_DAYS` | `31` | 日志保留天数 |
| `CHAT_RETENTION_DAYS` | `31` | 聊天消息保留天数 |
| `CHAT_MAX_PER_CONVERSATION` | `2000` | 单会话消息上限（LRU 淘汰） |

### 反向代理与 WebSocket

聊天依赖 WebSocket，Nginx 需显式放行 Upgrade：

```nginx
location /api/chat/ws {
    proxy_pass http://127.0.0.1:6970;
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection "upgrade";
    proxy_read_timeout 3600s;
}
```

> 多实例部署时 WebSocket 中枢为进程内实现，需开启会话粘滞（sticky session）；聊天历史已落库，不受影响。详见 [CHAT.md](./CHAT.md)。

## 五、作为系统服务运行

### systemd（Linux）

```ini
[Unit]
Description=Medical System
After=network.target

[Service]
WorkingDirectory=/opt/medical
ExecStart=/opt/medical/medical-system
Environment=PORT=6970
Environment=JWT_SECRET=替换为高强度随机串
Environment=DB_DRIVER=sqlite
Environment=DB_PATH=/opt/medical/medical.db
Restart=always

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl enable --now medical
sudo journalctl -u medical -f
```

### Windows 服务

可用 NSSM 将 `medical-system.exe` 注册为服务：

```powershell
nssm install MedicalSystem "C:\opt\medical\medical-system.exe"
nssm set MedicalSystem AppEnvironmentExtra PORT=6970 JWT_SECRET=<随机串>
nssm start MedicalSystem
```

## 六、生产检查清单

- [ ] 修改所有默认账号密码（`admin` / `doctor` …）
- [ ] 设置高强度 `JWT_SECRET`
- [ ] `ALLOWED_ORIGINS` 设为具体域名（勿用 `*`）
- [ ] 生产环境建议 `SEED_DEMO=false`
- [ ] 置于 HTTPS 反向代理（Nginx / Caddy）之后
- [ ] 定期备份数据库文件与 `uploads/`
- [ ] 影像样片目录限制写权限并纳入备份

## 七、升级

```bash
git pull
./deploy.sh          # 重新构建并启动（数据库结构与种子数据会自动幂等升级）
```

> 自动迁移（`AutoMigrate`）只会**新增**字段与表，不会删除数据；种子数据按「已存在则跳过」策略，重复执行安全。

## 八、故障排查

| 现象 | 原因 / 处理 |
| --- | --- |
| 页面空白 | 未构建前端；执行 `cd web && npm run build` |
| 端口占用 | 换端口：`./deploy.sh 7001` 或 `.\deploy.ps1 -Port 7001` |
| 接口 401 / 反复跳登录 | 令牌过期或 `JWT_SECRET` 变更后旧令牌失效，重新登录 |
| 接口 403 | 当前角色无该权限，见 [RBAC.md](./RBAC.md) |
| 影像无法回显 | 检查 `UPLOAD_DIR` 是否存在且进程有读写权限 |
