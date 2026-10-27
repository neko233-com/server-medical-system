# 数据库 · DATABASE

默认**内置 SQLite（纯 Go，无 CGO 依赖）**，开箱即用；通过环境变量即可切换为外置 **PostgreSQL / MySQL**。三者表结构完全一致（同一套 GORM AutoMigrate）。

---

## 一、SQLite（默认）

```bash
PORT=6970 ./medical-system
# 等价于：DB_DRIVER=sqlite DB_PATH=medical.db
```

- 实现：`glebarez/sqlite`（纯 Go），**无需 CGO、无需安装任何数据库**。
- 产物：单个 `medical.db` 文件，随二进制一起备份即可。
- 适用：单机部署、演示、边缘站点、中小规模门诊。

## 二、PostgreSQL

```bash
DB_DRIVER=postgres \
DB_DSN="host=127.0.0.1 port=5432 user=medical password=secret dbname=medical sslmode=disable TimeZone=Asia/Shanghai" \
./medical-system
```

依赖驱动：`gorm.io/driver/postgres`。建议 `sslmode=require` 用于生产。

## 三、MySQL

```bash
DB_DRIVER=mysql \
DB_DSN="medical:secret@tcp(127.0.0.1:3306)/medical?charset=utf8mb4&parseTime=True&loc=Local" \
./medical-system
```

依赖驱动：`gorm.io/driver/mysql`。请使用 `utf8mb4` 字符集以支持完整中文与表情。

---

## 四、数据模型总览

| 模型 | 说明 | 关键字段 |
| --- | --- | --- |
| `User` | 账号 | role, department, title, status |
| `Role` / `Permission` / `RolePermission` / `UserPermission` | RBAC | 角色、权限、授予关系 |
| `Patient` | 患者 | blood_type, allergies |
| `MedicalCase` | 病例 | room_id/room_no, visit_date, diagnosis |
| `Prescription` / `PrescriptionItem` | 处方与明细 | total_amount, status, room_id |
| `Medicine` | 药品 | price, is_active, **stock_qty, reorder_level** |
| `Room` | 门诊房间 | room_no, type, doctor_id, printer_id |
| `Printer` | 打印机 | doctor_id, room_id, is_default, connection |
| `PrintJob` | 打印审计 | printer/doctor/room/patient, doc_type, ref_id |
| `Surgery` | 手术 | surgery_no, status, started_at/ended_at |
| `ImagingStudy` / `ImagingImage` | 影像检查与样片 | modality, findings, stored_name |

## 五、迁移策略

启动即执行 `AutoMigrate`：**只增不改不删**，字段新增安全；如需删除字段请手动处理。

- **幂等**：重复启动不会重复写入种子数据（按「表已有记录则跳过」策略）。
- **升级流程**：停止服务 → 备份数据库 → 替换二进制 → 启动（自动迁移）→ 验证。

## 六、备份与恢复

### SQLite

```bash
# 备份（热备，建议在低峰执行）
cp medical.db medical.db.bak
tar czf uploads-$(date +%F).tgz uploads/

# 恢复
cp medical.db.bak medical.db
```

### PostgreSQL / MySQL

```bash
pg_dump -Fc medical > backup.dump     # 备份
pg_restore -d medical backup.dump     # 恢复

mysqldump -u medical -p medical > backup.sql
mysql -u medical -p medical < backup.sql
```

> `uploads/` 存放影像样片，**必须与数据库一同备份**，否则影像记录会变成死链。

## 七、跨库兼容性说明

为避免不同数据库的方言差异导致统计口径不一致，系统**刻意不使用**任何方言日期函数（如 SQLite 的 `strftime`、PG 的 `date_trunc`）。
日期窗口与按天分组全部在 Go 侧完成（`internal/handlers/operations.go`），因此 **SQLite / Postgres / MySQL 的运营分析结果完全一致**。
