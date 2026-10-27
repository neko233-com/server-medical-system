# 端到端测试脚本

这些脚本针对**真实运行的服务**做黑盒验证：启动服务 → 用 HTTP 走完整业务流程 →
校验状态码与响应体。它们与 Go 单元测试互补——单元测试覆盖纯逻辑（如聊天留存），
这些脚本覆盖**接口契约 + RBAC 拦截 + 跨模块联动**。

## 用法

```bash
# 1. 编译并启动服务（建议用干净的数据库，保证种子数据可复现）
go build -o .tmptest/medsrv.exe ./cmd/server
rm -f .tmptest/smoke.db
UPLOAD_DIR="$PWD/.tmptest/uploads" \
DB_PATH="$PWD/.tmptest/smoke.db" \
LOG_DIR="$PWD/.tmptest/logs" \
./.tmptest/medsrv.exe &

# 2. 运行测试
python scripts/e2e/smoke_test.py     # 核心业务 + RBAC + 运营分析 + 聊天
python scripts/e2e/clinic_test.py    # 预约挂号 / 住院床位 / 收费结算

# 3. 停止服务
kill %1
```

> 脚本默认连接 `http://127.0.0.1:6970`。若改了端口，请修改脚本顶部的 `BASE`。

## 覆盖范围

| 脚本 | 覆盖内容 |
| --- | --- |
| `smoke_test.py` | 登录与令牌、RBAC 拦截（越权返回 403）、患者/病例/处方、房间与打印机解析、手术状态流转、影像上传与鉴权回显、打印任务审计、运营分析、WebSocket 群聊与私聊投递、图文消息、非法令牌拒绝 |
| `clinic_test.py` | 预约建档与状态流转、非法状态拒绝、病区/床位 CRUD、入院/出院与重复占床拦截、病区非空删除拦截、结算单开单/明细/重算/收款/退费、已收款单据不可删除、医生越权拦截、运营分析中的床位占用与收入构成 |

两个脚本都以退出码表示结果（`0` = 全部通过），可直接接入 CI。

## 演示账号

脚本使用以下账号（`SEED_DEMO=true` 时自动创建）：

| 用户名 | 密码 | 角色 |
| --- | --- | --- |
| `root` | `root` | 超级管理员 |
| `doctor` | `doctor123` | 医生 |
| `nurse` | `nurse123` | 护士 |
| `reception` | `recept123` | 挂号收费 |
| `radiologist` | `radio123` | 影像科 |
| `surgeon` | `surgeon123` | 手术医生 |
| `pharmacist` | `pharm123` | 药剂师 |
