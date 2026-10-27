# 医疗管理系统 · Medical System

> 面向医院的全球化医疗管理系统：覆盖医生诊疗、病历、处方、药品、门诊房间、手术、影像检查、打印与精细化权限（RBAC）的全流程，**全部集成在 Web 端**。默认内置 SQLite 数据库，可一键切换为外置 Postgres / MySQL。

技术栈：**Go 1.27 (Gin + GORM + 纯 Go SQLite)** · **React 19 + Vite** · **react-i18next** · **reicon** 图标 · **ReactBits 风格**动画组件。

单二进制即可同时托管后端 API 与前端 SPA。

---

## ✨ 功能特性

### 诊疗闭环（核心）
- **患者管理**：建档（含血型、过敏史）、检索、详情页时间线。
- **病历 / 病例**：主诉、现病史、既往史、诊断、医嘱，按患者维度聚合全部就诊记录。
- **处方开方**：从药品库选药、自动计算金额，关联门诊房间号与患者。
- **药品库（常规药全量 + 后台可扩展）**：内置 60+ 种常用药（中英双语），管理员可新增/编辑/停用并**配置价格**。

### 医院运营（本次新增）
- **门诊房间管理**：房间号、名称（中英）、类型（诊室/手术室/放射科/急诊）、科室、楼层、绑定医生与默认打印机、启停状态。
- **手术管理**：手术排期（计划 → 进行中 → 完成），主刀/助手/麻醉师、手术室、麻醉方式、术前/术后诊断、术中发现、并发症、时长与时间戳。
- **影像中心**：CT / X 光 / MRI / 超声等检查登记、图文报告（所见 / 印象）、多张样片上传与**带鉴权的图像回显**。
- **打印机与打印**：打印机登记（类型/连接方式/IP/端口）、按「绑定医生 → 房间默认 → 全局默认」三级优先自动解析目标打印机；每次打印生成**审计记录（PrintJob）**。

### 💬 院内即时通讯（浮窗聊天，本次新增）
右下角浮窗随时唤起，**全员可用**：
- **私聊**：同事之间点对点沟通（离线留言，上线可见）；
- **全院大群**：单一内置群，全员可见可发言；
- **图文消息**：文字 + 图片（图片经鉴权接口下发）；
- **实时投递**：WebSocket 实时推送，含在线名单、未读角标、断线指数退避重连；
- **自动回收**：消息按 **31 天窗口 + 每会话 2000 条 LRU** 自动清理，图片附件与孤儿文件同步回收；
- **日志轮转**：内置零依赖轮转日志（stdout + `logs/*.log`，按大小分片、按天保留）。

详见 [docs/CHAT.md](./docs/CHAT.md)。

### 🧮 医院运营总览（管理员后台运营）
面向管理层的**全院量化核算**（受 `report:read` 权限保护，默认仅管理员可见）：
- **核心 KPI**：累计患者、在岗医生、今日门诊量、今日处方数；
- **收入核算**：今日收入、本月收入（以处方金额为准，两位小数精确）；
- **待办积压**：待排手术数、待写影像报告数；
- **库存预警**：低于补货警戒线的药品品种数；
- **趋势图**：近 7/14/30 天逐日门诊量与收入（缺失日期自动补零，图表连续）；
- **科室效能**：各科室医生数、门诊量、收入；
- **医生效能**：各医生门诊量、处方数、收入排行；
- **资源状态**：房间实时占用（今日已接诊/空闲）、打印机清单、最近打印任务审计。

> 实现上刻意不使用任何数据库方言日期函数，时间窗口与按天分组在 Go 侧完成，确保 SQLite / Postgres / MySQL 结果一致。

### 权限体系（RBAC，本次新增）
- **角色**：管理员 / 医生 / 护士 / 前台 / 放射科 / 手术医生 / 药剂师 七种内置角色。
- **权限目录**：23 项细粒度权限（`patient:read`、`case:write`、`prescription:create`、`surgery:manage`、`imaging:upload`、`room:manage`、`printer:manage`、`user:manage`、`role:manage` …）。
- **组合授权**：用户权限 = 所属角色权限 ∪ 个人授予 − 个人撤销，在 Gin 中间件层**接口级强制校验**（无权限返回 403）。
- **管理后台**：图形化维护用户、角色、角色-权限矩阵、用户级权限覆盖。

### 全球化 / i18n
- 界面默认中文，一键切换英文；药品名称内建多语言字段，切换即时生效。新增语言只需加一个语言包文件，零后端改动。
- 服务端错误/提示消息同样支持 `Accept-Language` 多语言。

### 数据库（可插拔）
- **默认内置 SQLite**（纯 Go，`glebarez/sqlite`，无 CGO 依赖），开箱即用。
- **可选外置**：通过环境变量 `DB_DRIVER=postgres|mysql` + `DB_DSN=...` 切换为 Postgres / MySQL，自动迁移结构完全一致。

### 单二进制部署
编译后的二进制可直接托管已构建的前端（SPA，从 `./web/dist` 提供），并同时暴露 `/api`。

---

## 🚀 快速开始

> 🚀 **最快的方式**：`./deploy.sh`（Windows 用 `.\deploy.ps1`）一键构建并启动，默认端口 **6970**，浏览器打开 http://localhost:6970

### 方式零：一键部署（推荐）
```bash
# Windows (PowerShell)
.\deploy.ps1

# Linux / macOS / Git Bash
./deploy.sh

# 常用变体
.\deploy.ps1 -Port 7001      # 指定端口
./deploy.sh 7001             # 指定端口
BUILD_ONLY=1 ./deploy.sh     # 只构建不启动
.\deploy.ps1 -SkipWeb        # 跳过前端构建（web/dist 已存在）
```
脚本依次完成：依赖检查 → 构建前端 → 编译后端（静态链接 + 体积裁剪）→ 启动服务。
产物为单二进制 `medical-system` + `web/dist`，二者拷到目标机器即可运行，无需安装 Go/Node。

### 方式一：前后端分离开发（推荐开发期）
```bash
# 终端 1 —— 后端 (默认 :6970)
go run ./cmd/server

# 终端 2 —— 前端 (默认 :5173，/api 自动代理到 6970)
cd web && npm install && npm run dev
```
打开 http://localhost:5173

### 方式二：手动构建单二进制
```bash
cd web && npm install && npm run build   # 生成 web/dist
CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o medical-system ./cmd/server
PORT=6970 ./medical-system               # 访问 http://localhost:6970
```

### 环境变量
| 变量 | 默认值 | 说明 |
| --- | --- | --- |
| `PORT` | `6970` | HTTP 端口（刻意避开 8080/3000 常见冲突口） |
| `DB_DRIVER` | `sqlite` | `sqlite` / `postgres` / `mysql` |
| `DB_DSN` | `medical.db` | sqlite 文件路径，或 Postgres/MySQL 连接串 |
| `JWT_SECRET` | 内置默认值 | JWT 签名密钥 |
| `UPLOAD_DIR` | `./uploads` | 影像样片存储目录 |
| `SEED_DEMO` | `true` | 是否注入演示账号与各模块种子数据 |
| `ALLOWED_ORIGINS` | `*` | CORS 允许来源 |
| `LOG_DIR` | `./logs` | 轮转日志目录 |
| `LOG_MAX_MB` | `64` | 单日志文件大小上限，超出自动分片 |
| `LOG_KEEP_DAYS` | `31` | 日志保留天数 |
| `CHAT_RETENTION_DAYS` | `31` | 聊天消息保留天数 |
| `CHAT_MAX_PER_CONVERSATION` | `2000` | 单会话消息上限（LRU 淘汰） |

> 示例（切换为外置 PostgreSQL）：
> ```bash
> DB_DRIVER=postgres DB_DSN="host=127.0.0.1 port=5432 user=med password=secret dbname=medical sslmode=disable" go run ./cmd/server
> ```

### 默认账号
| 角色 | 用户名 | 密码 | 备注 |
| --- | --- | --- | --- |
| **超级管理员** | `root` | `root` | **默认账号**，全权限，含 RBAC 与运营后台 |
| 医生 | `doctor` | `doctor123` | 看病、开方、房间/患者/病历 |
| 手术医生 | `surgeon` | `surgeon123` | 手术排期与记录 |
| 放射科 | `radiologist` | `radio123` | 影像登记、上传与报告 |
| 护士 | `nurse` | `nurse123` | 患者、病历、床位护理 |
| 挂号收费 | `reception` | `recept123` | 挂号/收费账单 |
| 药剂师 | `pharmacist` | `pharm123` | 发药、药品库存 |

> 生产部署请务必在首次启动后修改以上账号密码，并通过 JWT_SECRET 强化令牌签名。

（`SEED_DEMO=true` 时自动创建以上演示账号与房间、打印机、药品种子数据。）

---

## 📁 项目结构
```
cmd/server          后端入口（路由、CORS、i18n、SPA 托管、DB 初始化+种子）
internal/
  config            配置（含多驱动 DB、上传目录、种子开关）
  models            User / Role / Permission / Patient / MedicalCase /
                    Prescription / Medicine / Room / Printer / PrintJob /
                    Surgery / ImagingStudy / ImagingImage
  db                SQLite / Postgres / MySQL 初始化 + 自动迁移
  rbac             用户有效权限计算（角色∪授予−撤销）
  i18n             服务端多语言消息（zh-CN / en-US）
  middleware       JWT 鉴权 + 角色校验 + 接口级权限(RequirePermission/RequireAny) + CORS
  handlers          auth / patients / cases / medicines / prescriptions /
                    rbac / rooms / printers / surgeries / imaging
  seed              默认账号 + 角色权限 + 房间/打印机/药品种子数据
web/
  src/i18n          语言包（按需扩展）与 LANGUAGES 注册表
  src/components    reicon 图标封装 / ReactBits 风格动画 / UI 基础组件 / 布局
  src/pages        Login / Dashboard / Patients / PatientDetail / Consultation /
                   Medicines / Rooms / Printers / Surgeries / Imaging / RBAC / Settings
  src/pages/print  处方打印视图（记录 PrintJob + window.print）
  src/api          Axios 客户端（自动注入 token 与 Accept-Language）
  src/utils         按语言取药品名、金额格式化、AuthImage 鉴权图像
```

---

## 🔐 RBAC 速览
- 权限代码形如 `资源:动作`，例如 `surgery:manage`、`imaging:upload`、`printer:read`。
- 中间件 `RequirePermission(db, code)` 与 `RequireAny(db, codes...)` 在路由注册时挂载。
- 前端 `AuthContext.can(code)` 控制侧边栏与按钮显隐，实现「看不到、也调不到」的双重防护。
- 角色为系统保留项（不可删除），用户级授权可在角色之外做精细加减。

## 🖨️ 打印机解析规则（优先级）
1. **绑定医生**：若当前医生绑定了专属打印机，优先使用。
2. **房间默认**：否则取所在门诊房间的默认打印机。
3. **全局默认**：再否则取标记为默认的全局打印机。

开处方并打印时，系统按上述规则自动选定打印机并写入 `PrintJob` 审计记录（含患者、医生、房间、文档类型与关联 ID）。

---

---

## 📖 文档

完整文档见 [`docs/`](./docs)：

| 文档 | 内容 |
| --- | --- |
| [docs/README.md](./docs/README.md) | 文档索引与架构概览 |
| [docs/FEATURES.md](./docs/FEATURES.md) | 全部功能清单与业务走查 |
| [docs/RBAC.md](./docs/RBAC.md) | 权限体系：26 项权限、7 种角色、权限矩阵 |
| [docs/CHAT.md](./docs/CHAT.md) | 院内即时通讯：私聊 + 大群、图文、31 天 LRU、日志轮转 |
| [docs/DEPLOYMENT.md](./docs/DEPLOYMENT.md) | 一键部署、systemd/Windows 服务、生产清单 |
| [docs/DATABASE.md](./docs/DATABASE.md) | 内置 SQLite 与外置 Postgres / MySQL、备份恢复 |
| [docs/API.md](./docs/API.md) | 全部 REST 接口与所需权限 |

## 🌐 如何扩展语言
1. 新建 `web/src/i18n/locales/<code>.js`，导出与现有包**同 key** 的翻译对象。
2. 在 `web/src/i18n/languages.js` 的 `LANGUAGES` 数组追加 `{ code, label, short }`。
3. 在 `web/src/i18n/index.js` 的 `resources` 注册该语言。
> 药品名称已内建 `name_zh` / `name_en` 双语字段，新增语言时只需在 Medicine 模型补字段并在前端 `utils/format.js` 增加取值即可。

## 💊 如何扩展药品
- 后台「药品库」页（管理员）可图形化新增/编辑药品并配置价格；
- 或通过 API：`POST /api/medicines`（需 `medicine:manage`），字段含 `code, name_zh, name_en, category, category_en, spec, manufacturer, unit, price, currency, is_active`。

---

## 🔌 主要 API
| 方法 | 路径 | 说明 | 权限 |
| --- | --- | --- | --- |
| POST | `/api/auth/login` | 登录获取 JWT（返回 token + 用户 + 权限） | 公开 |
| GET | `/api/rbac/permissions` `/api/rbac/roles` `/api/rbac/users` | 权限目录 / 角色列表 / 用户列表 | 对应 `*:read/manage` |
| GET/POST/PUT/DELETE | `/api/rooms` | 门诊房间管理 | `room:read` / `room:manage` |
| GET/POST/PUT/DELETE | `/api/printers` | 打印机管理 | `printer:read` / `printer:manage` |
| GET | `/api/printers/resolve` | 按医生/房间解析目标打印机 | `printer:read` |
| POST | `/api/print-jobs` | 创建打印任务（自动解析+审计） | 登录 |
| GET/POST/PUT/DELETE | `/api/surgeries` | 手术管理 | `surgery:read` / `surgery:manage` |
| GET/POST/PUT/DELETE | `/api/imaging` | 影像检查登记与报告 | `imaging:read` / `imaging:upload` / `imaging:report` |
| POST | `/api/imaging/:id/images` | 上传 CT / X 光样片 | `imaging:upload` |
| GET | `/api/imaging/images/:id/file` | 鉴权获取样片（前端 AuthImage 使用） | `imaging:read` |
| GET | `/api/operations/overview` `/trend` `/departments` `/doctors` `/status` | 运营分析（KPI/趋势/科室/医生/资源） | `report:read` |
| GET | `/api/patients` `/api/cases` `/api/medicines` `/api/prescriptions` | 核心诊疗数据 | 对应权限 |

---

## 📝 说明
- 数据库默认**纯 Go SQLite**（无 CGO 依赖），经 `DB_DRIVER` + `DB_DSN` 可切换为 Postgres / MySQL，迁移结构一致。
- 影像样片默认存于 `UPLOAD_DIR`（`./uploads`），通过带 Bearer 鉴权的接口回显，前端用 `AuthImage` 组件以 blob 安全加载。
- `reactbits` 非 npm 包，此处以 `src/components/anim/*` 本地实现其动画组件（FadeIn / BlurText / SpotlightCard / AnimatedBackground）。
- 图标使用 `reicon` 函数式 SVG 库，封装于 `src/components/Reicon.jsx`。
