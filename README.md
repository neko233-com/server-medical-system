# 医疗管理系统 · Medical System

> 面向医生的全球化医疗管理系统：病例管理 + 开方 + 全量常规药品库 + 后台可扩展药品与价格 + 多语言（默认中/英，扩展语言极简）。

技术栈：**Go 1.27 (Gin + GORM + 纯 Go SQLite)** · **React 19 + Vite** · **react-i18next** · **reicon** 图标 · **ReactBits 风格**动画组件。

---

## ✨ 功能特性

- **医生诊疗闭环**：选择/新建患者 → 填写主诉/现病史/既往史/诊断/医嘱 → 开具处方，自动计算金额。
- **病例管理**：每次就诊生成病例记录，按患者维度时间线查看全部病史与处方。
- **药品库（常规药全量 + 后台可扩展）**：内置 60+ 种常用药（抗生素、解热镇痛、心血管、消化、呼吸、内分泌、神经、抗过敏、维生素、外用、中成药、五官、其他），全部**中英双语**名称；管理员可在后台新增/编辑/停用药品并**配置价格**。
- **角色权限**：`doctor` 仅能看病开方；`admin` 可维护药品库与价格（接口级 403 防护）。
- **全球化 / i18n**：界面默认中文，一键切换英文；药品名称内建多语言字段，切换即时生效。新增语言只需加一个语言包文件，零后端改动。
- **单二进制部署**：`go run` / 编译后的二进制可直接托管已构建的前端（SPA），也支持 Vite 开发服务器热代理。

## 🚀 快速开始

### 方式一：前后端分离开发（推荐开发期）
```bash
# 终端 1 —— 后端 (默认 :8080)
go run ./cmd/server

# 终端 2 —— 前端 (默认 :5173，/api 自动代理到 8080)
cd web && npm install && npm run dev
```
打开 http://localhost:5173

### 方式二：单一二进制（生产/演示）
```bash
cd web && npm install && npm run build   # 生成 web/dist
go run ./cmd/server                       # 同时托管 API 与前端，访问 http://localhost:8080
```

环境变量：`PORT`(8080) · `DB_PATH`(medical.db) · `JWT_SECRET` · `ALLOWED_ORIGINS`(默认 *)

### 默认账号
| 角色 | 用户名 | 密码 |
| --- | --- | --- |
| 管理员 | `admin` | `admin123` |
| 医生 | `doctor` | `doctor123` |

## 📁 项目结构
```
cmd/server          后端入口（路由、CORS、i18n、SPA 托管）
internal/
  config            配置
  models            User / Patient / MedicalCase / Medicine / Prescription
  db                SQLite 初始化 + 自动迁移
  i18n              服务端多语言消息（zh-CN / en-US）
  middleware        JWT 鉴权 + 管理员校验 + CORS
  handlers          auth / patients / cases / medicines / prescriptions
  seed              默认账号 + 常用药种子数据
web/
  src/i18n          语言包（按需扩展）与 LANGUAGES 注册表
  src/components    reicon 图标封装 / ReactBits 风格动画 / UI 基础组件 / 布局
  src/pages        Login / Dashboard / Patients / PatientDetail / Consultation / Medicines / Settings
  src/api           Axios 客户端（自动注入 token 与 Accept-Language）
  src/utils         按语言取药品名、金额格式化等
```

## 🌐 如何扩展语言
1. 新建 `web/src/i18n/locales/<code>.js`，导出与现有包**同 key** 的翻译对象。
2. 在 `web/src/i18n/languages.js` 的 `LANGUAGES` 数组追加 `{ code, label, short }`。
3. 在 `web/src/i18n/index.js` 的 `resources` 注册该语言。
> 药品名称已内建 `name_zh` / `name_en` 双语字段，新增语言时只需在 Medicine 模型补字段并在前端 `utils/format.js` 增加取值即可。

## 💊 如何扩展药品
- 后台「药品库」页（管理员）可图形化新增/编辑药品并配置价格；
- 或通过 API：`POST /api/medicines`（需 admin token），字段含 `code, name_zh, name_en, category, category_en, spec, manufacturer, unit, price, currency, is_active`。

## 🔌 主要 API
| 方法 | 路径 | 说明 | 权限 |
| --- | --- | --- | --- |
| POST | `/api/auth/login` | 登录获取 JWT | 公开 |
| GET | `/api/patients` | 患者列表（搜索/分页） | 登录 |
| POST | `/api/patients` | 新建患者 | 登录 |
| POST | `/api/cases` | 新建病例 | 登录 |
| GET | `/api/medicines` | 药品库列表 | 登录 |
| POST/PUT/DELETE | `/api/medicines` | 维护药品与价格 | 管理员 |
| POST | `/api/prescriptions` | 开具处方（自动算价） | 登录 |

## 📝 说明
- 数据库默认使用**纯 Go SQLite**（无 CGO 依赖），亦可切换为 Postgres/MySQL（改 `internal/db` 的 driver 即可）。
- `reactbits` 非 npm 包，此处以 `src/components/anim/*` 本地实现其动画组件（FadeIn / BlurText / SpotlightCard / AnimatedBackground）。
- 图标使用 `reicon` 函数式 SVG 库，封装于 `src/components/Reicon.jsx`。
