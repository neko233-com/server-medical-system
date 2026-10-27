# 权限体系 · RBAC

## 一、模型

```
用户 (User) ── 归属 ──► 角色 (Role) ── 授予 ──► 权限 (Permission)
   │
   └──────────── 用户级授权/撤销 (UserPermission) ────────────┐
                                                              ▼
                        有效权限 = 角色权限 ∪ 用户授予 − 用户撤销
```

- **角色（Role）**：7 种内置角色，`IsSystem=true` 的系统角色不可删除（保证升级安全）。
- **权限（Permission）**：26 项细粒度权限码，形如 `资源:动作`。
- **用户级覆盖**：在角色之外单独授予某项权限，或撤销角色自带的权限（实现「这个护士临时可以看影像」这类例外）。

> 计算逻辑见 `internal/rbac/rbac.go` 的 `UserPermissions()`；**这是权限的唯一真相来源**，中间件与前端共用。

## 二、26 项权限清单

| 权限码 | 中文 | 说明 |
| --- | --- | --- |
| `dashboard:view` | 查看工作台 | 访问系统工作台 |
| `patient:read` | 查看患者 | 浏览患者档案 |
| `patient:write` | 管理患者 | 新建/编辑患者 |
| `case:read` | 查看病例 | 浏览诊疗记录 |
| `case:write` | 管理病例 | 新建/编辑病例 |
| `prescription:read` | 查看处方 | 浏览处方 |
| `prescription:create` | 开具处方 | 开方 |
| `prescription:dispense` | 发药 | 处方发药 |
| `medicine:read` | 查看药品 | 浏览药品库 |
| `medicine:manage` | 管理药品 | 增删改药品与定价 |
| `surgery:read` | 查看手术 | 浏览手术记录 |
| `surgery:manage` | 管理手术 | 排期/记录手术 |
| `imaging:read` | 查看影像 | 浏览影像检查 |
| `imaging:upload` | 管理影像 | 开单/上传影像 |
| `imaging:report` | 出具报告 | 书写影像报告 |
| `room:read` | 查看房间 | 浏览门诊房间 |
| `room:manage` | 管理房间 | 管理门诊房间 |
| `printer:read` | 查看打印机 | 浏览打印机 |
| `printer:manage` | 管理打印机 | 管理打印机 |
| `inventory:read` | 查看库存 | 浏览药品库存 |
| `inventory:manage` | 管理库存 | 入库/出库与盘点 |
| `report:read` | **运营报表** | **查看全院运营分析与后台运营** |
| `user:read` | 查看账号 | 浏览用户账号（管理属性） |
| `user:manage` | 管理账号 | 增删改账号、分配角色 |
| `role:read` | 查看角色 | 浏览角色与权限矩阵 |
| `role:manage` | 管理角色 | 增删改角色与权限 |

## 三、角色权限矩阵（✅ = 默认拥有）

| 权限 | 管理员 | 医生 | 护士 | 挂号收费 | 影像科 | 手术医生 | 药房 |
| --- | :---: | :---: | :---: | :---: | :---: | :---: | :---: |
| `dashboard:view` | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| `patient:read` | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| `patient:write` | ✅ | ✅ | ✅ | ✅ | | | |
| `case:read` | ✅ | ✅ | ✅ | | | ✅ | |
| `case:write` | ✅ | ✅ | | | | | |
| `prescription:read` | ✅ | ✅ | ✅ | ✅ | | | ✅ |
| `prescription:create` | ✅ | ✅ | | | | | |
| `prescription:dispense` | ✅ | | | | | | ✅ |
| `medicine:read` | ✅ | ✅ | ✅ | ✅ | | | ✅ |
| `medicine:manage` | ✅ | | | | | | |
| `surgery:read` | ✅ | ✅ | | | | ✅ | |
| `surgery:manage` | ✅ | | | | | ✅ | |
| `imaging:read` | ✅ | ✅ | ✅ | | ✅ | ✅ | |
| `imaging:upload` | ✅ | ✅ | | | ✅ | | |
| `imaging:report` | ✅ | | | | ✅ | | |
| `room:read` | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| `room:manage` | ✅ | | | | | | |
| `printer:read` | ✅ | ✅ | | ✅ | | | ✅ |
| `printer:manage` | ✅ | | | | | | |
| `inventory:read` | ✅ | | | | | | ✅ |
| `inventory:manage` | ✅ | | | | | | ✅ |
| `report:read` | ✅ | | | | | | |
| `user:read` / `user:manage` | ✅ | | | | | | |
| `role:read` / `role:manage` | ✅ | | | | | | |

> **安全设计**：`user:read` 刻意不授予医生——列出全院账号属管理操作，医护人员如需选择同事，应使用面向所有登录用户的精简端点 `GET /api/rbac/directory`（仅返回姓名/科室等必要字段）。

## 四、强制校验位置

权限在**服务端中间件层**强制执行（`internal/middleware/auth.go`）：

```go
r.GET("/surgeries", middleware.RequireAny(db, "surgery:read"), h.List)
r.POST("/surgeries", middleware.RequireAny(db, "surgery:manage"), h.Create)
```

- `RequirePermission(db, code)`：要求具备指定权限，否则 **403**。
- `RequireAny(db, codes...)`：具备其中任一项即放行，否则 **403**。

前端 `AuthContext.can(code)` 仅控制菜单与按钮**显隐**，与后端共用同一套权限码，形成「看不到、也调不到」的双重防护。

## 五、演示账号

| 角色 | 用户名 | 密码 |
| --- | --- | --- |
| 管理员 | `admin` | `admin123` |
| 医生 | `doctor` | `doctor123` |
| 护士 | `nurse` | `nurse123` |
| 挂号收费 | `reception` | `recept123` |
| 影像科 | `radiologist` | `radio123` |
| 手术医生 | `surgeon` | `surgeon123` |
| 药房 | `pharmacist` | `pharm123` |

登录响应包含 `{ token, user, permissions }`，其中 `permissions` 为形如 `{"surgery:manage": true}` 的对象。

## 六、生产建议

1. 首次启动后**立即修改所有默认账号密码**。
2. 遵循最小权限：用内置角色满足常规需求，确需例外时优先使用**用户级授予**而非扩权角色。
3. 自定义角色请勿随意勾选 `*_:manage` / `user:*` / `role:*` / `report:read`。
4. `JWT_SECRET` 必须设置为高强度随机值（`openssl rand -hex 32`）。
