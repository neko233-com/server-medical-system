# API 参考 · API

所有接口前缀 `/api`，除登录外均需 `Authorization: Bearer <token>`。
服务端消息支持 `Accept-Language`（`zh-CN` / `en-US`）多语言。

统一分页：`?page=1&page_size=20`，响应形如 `{ items: [...], pagination: { page, page_size, total } }`。
统一成功体：`{ item, message }`；错误体：`{ error: "本地化消息" }`，HTTP 状态码见下表。

---

## 认证

| 方法 | 路径 | 说明 | 权限 |
| --- | --- | --- | --- |
| POST | `/api/auth/login` | 登录，body `{username,password}`，返回 `{token,user,permissions}` | 公开 |
| GET | `/api/auth/me` | 当前用户 `{user,permissions}` | 登录 |

## 健康

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/health` | `{status,service,db_driver}` |

## 患者

| 方法 | 路径 | 说明 | 权限 |
| --- | --- | --- | --- |
| GET | `/api/patients` | 列表（支持 `q` 搜索、分页） | `patient:read` |
| POST | `/api/patients` | 建档 | `patient:write` |
| GET | `/api/patients/:id` | 详情 | `patient:read` |
| PUT / DELETE | `/api/patients/:id` | 编辑 / 删除 | `patient:write` |

## 病例

| 方法 | 路径 | 说明 | 权限 |
| --- | --- | --- | --- |
| GET | `/api/cases` | 列表 | `case:read` |
| POST | `/api/cases` | 新建病例（关联房间号） | `case:write` |
| GET / PUT | `/api/cases/:id` | 查看 / 编辑 | `case:read` / `case:write` |

## 处方

| 方法 | 路径 | 说明 | 权限 |
| --- | --- | --- | --- |
| GET | `/api/prescriptions` | 列表 | `prescription:read` |
| POST | `/api/prescriptions` | 开方（自动算价，关联房间） | `prescription:create` |
| GET | `/api/prescriptions/:id` | 详情（含明细） | `prescription:read` |

## 药品与库存

| 方法 | 路径 | 说明 | 权限 |
| --- | --- | --- | --- |
| GET | `/api/medicines` | 列表（支持 `q` / `category`） | `medicine:read` |
| POST | `/api/medicines` | 新增（可带 `stock_qty` / `reorder_level`） | `medicine:manage` |
| PUT | `/api/medicines/:id` | 编辑（含库存字段） | `medicine:manage` |
| PUT | `/api/medicines/:id/price` | 单独改价 | `medicine:manage` |
| DELETE | `/api/medicines/:id` | 删除 | `medicine:manage` |

## 预约 / 床位 / 收费

| 方法 | 路径 | 说明 | 权限 |
| --- | --- | --- | --- |
| GET / POST | `/api/appointments` | 预约列表 / 新建预约 | `appointment:read` / `appointment:manage` |
| GET / PUT / DELETE | `/api/appointments/:id` | 预约详情 / 修改 / 删除 | `appointment:read` / `appointment:manage` |
| PUT | `/api/appointments/:id/status` | 状态流转（booked→arrived→completed / cancelled / no_show） | `appointment:manage` |
| GET / POST | `/api/wards` | 病区列表 / 新建病区 | `bed:read` / `bed:manage` |
| DELETE | `/api/wards/:id` | 删除病区（仍有床位时返回 409） | `bed:manage` |
| GET / POST | `/api/beds` | 床位列表（可按 ward_id / status 过滤）/ 新建床位 | `bed:read` / `bed:manage` |
| GET / PUT / DELETE | `/api/beds/:id` | 床位详情 / 修改 / 删除 | `bed:read` / `bed:manage` |
| PUT | `/api/beds/:id/admit` | 办理入院（占床并记录时间戳；重复占床返回 409） | `bed:manage` |
| PUT | `/api/beds/:id/discharge` | 办理出院（释放床位） | `bed:manage` |
| GET / POST | `/api/invoices` | 结算单列表 / 开单 | `billing:read` / `billing:manage` |
| GET / PUT / DELETE | `/api/invoices/:id` | 结算单详情 / 修改 / 删除（已收款返回 409） | `billing:read` / `billing:manage` |
| POST / DELETE | `/api/invoices/:id/items` `/api/invoices/items/:itemId` | 增删收费明细并自动重算 | `billing:manage` |
| POST | `/api/invoices/:id/pay` | 收款（payment_method: cash/card/wechat/alipay/insurance） | `billing:manage` |
| POST | `/api/invoices/:id/refund` | 退费（状态流转，不删除记录） | `billing:manage` |

## 门诊房间

| 方法 | 路径 | 说明 | 权限 |
| --- | --- | --- | --- |
| GET | `/api/rooms` | 列表（支持 `q` / `type`） | `room:read` |
| POST / PUT / DELETE | `/api/rooms[/:id]` | 管理房间 | `room:manage` |
| PUT | `/api/rooms/:id/doctor` | 绑定医生（`doctor_id=0` 解绑） | `room:manage` |

房间类型 `type`：`consultation` | `surgery` | `radiology` | `emergency`

## 打印机与打印

| 方法 | 路径 | 说明 | 权限 |
| --- | --- | --- | --- |
| GET | `/api/printers` | 列表 | `printer:read` |
| POST / PUT / DELETE | `/api/printers[/:id]` | 管理打印机 | `printer:manage` |
| PUT | `/api/printers/:id/default` | 设为全局默认 | `printer:manage` |
| GET | `/api/printers/resolve` | 按医生/房间解析目标打印机，**直接返回 printer 对象** | `printer:read` |
| POST | `/api/print-jobs` | 创建打印任务（自动解析打印机并留痕） | 登录 |
| GET | `/api/print-jobs` | 打印审计列表 | `printer:read` |

## 手术

| 方法 | 路径 | 说明 | 权限 |
| --- | --- | --- | --- |
| GET | `/api/surgeries` | 列表（支持 `patient_id` / `status`） | `surgery:read` |
| GET | `/api/surgeries/:id` | 详情 | `surgery:read` |
| POST / PUT / DELETE | `/api/surgeries[/:id]` | 管理手术 | `surgery:manage` |
| PUT | `/api/surgeries/:id/status` | 状态流转，自动写时间戳 | `surgery:manage` |

状态：`planned → in_progress（写 started_at）→ done（写 ended_at）`，或 `cancelled`。

## 影像

| 方法 | 路径 | 说明 | 权限 |
| --- | --- | --- | --- |
| GET | `/api/imaging` | 检查列表 | `imaging:read` |
| GET | `/api/imaging/:id` | 检查详情（含样片） | `imaging:read` |
| POST / PUT / DELETE | `/api/imaging[/:id]` | 登记与编辑检查 | `imaging:upload` |
| PUT | `/api/imaging/:id/status` | 状态流转 | `imaging:report` |
| POST | `/api/imaging/:id/images` | **multipart** 上传样片，字段名 `file`，可选 `caption` | `imaging:upload` |
| GET | `/api/imaging/:id/images` | 样片列表 | `imaging:read` |
| DELETE | `/api/imaging/images/:imageId` | 删除样片 | `imaging:upload` |
| GET | `/api/imaging/images/:id/file` | **鉴权获取样片字节**（前端 AuthImage 使用） | `imaging:read` |

> `file` 字段需以 `multipart/form-data` 上传（同浏览器 `FormData`）；限制 25MB/张，类型须为 `image/*` 或 `application/dicom`。

## 🧮 运营分析（管理员）

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/api/operations/overview` | 核心 KPI：患者数、医生数、今日门诊量/处方数、今日/本月收入、待排手术、待写报告、库存预警数 |
| GET | `/api/operations/trend?days=30` | 逐日 `{date,visits,revenue}` 时间序列（默认 30 天，上限 365） |
| GET | `/api/operations/departments` | 科室效能 `{department,doctors,visits,revenue}` |
| GET | `/api/operations/doctors` | 医生效能 `{doctor_id,name,department,title,visits,prescriptions,revenue}` |
| GET | `/api/operations/status` | 资源实时状态 `{rooms,printers,recent_jobs}` |

全部要求 `report:read`（默认仅管理员），否则 **403**。

## RBAC

| 方法 | 路径 | 说明 | 权限 |
| --- | --- | --- | --- |
| GET | `/api/rbac/permissions` | 权限目录 | `role:read` |
| GET | `/api/rbac/roles` | 角色列表（含权限） | `role:read` |
| POST / PUT / DELETE | `/api/rbac/roles[/:code]` | 管理角色（系统角色禁删） | `role:manage` |
| GET / PUT | `/api/rbac/roles/:code/permissions` | 读写角色权限 | `role:read` / `role:manage` |
| GET | `/api/rbac/users` | 账号列表（不含密码） | `user:read` |
| POST / PUT / DELETE | `/api/rbac/users[/:id]` | 管理账号 | `user:manage` |
| PUT | `/api/rbac/users/:id/role` | 分配角色 | `user:manage` |
| GET / PUT | `/api/rbac/users/:id/permissions` | 读写用户级权限覆盖 | `user:read` / `user:manage` |
| GET | `/api/rbac/directory` | 精简人员字典（下拉选择器用） | 任意登录用户 |

---

## 状态码约定

| 码 | 含义 |
| --- | --- |
| 200 / 201 | 成功 / 已创建 |
| 400 | 请求参数错误（含业务校验失败） |
| 401 | 未认证或令牌失效（前端会自动跳转登录） |
| 403 | **已认证但无权限** |
| 404 | 资源不存在（`/api/*` 未匹配路由同样返回 404） |
| 409 | 资源冲突（如药品编码重复） |
| 500 | 服务端异常 |
