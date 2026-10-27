# 院内即时通讯 · CHAT

内置**浮窗聊天系统**，右下角悬浮按钮随时唤起，覆盖「同事私聊 + 全院大群」两种场景，支持**文字与图片**，消息实时投递并按 **31 天窗口 + LRU 容量自动回收**。

---

## 一、能力清单

| 能力 | 说明 |
| --- | --- |
| **私聊** | 任意两位在线同事之间点对点沟通，支持离线留言（上线后可见历史） |
| **全院大群** | 单一内置群（key = `hospital`），全员可见可发言 |
| **文字消息** | 支持换行，`Enter` 发送 |
| **图片消息** | 单张 ≤10MB，先上传再作为图片消息发送；图片经鉴权接口下发 |
| **在线名单** | 实时 presence，头像侧显示绿点 |
| **未读角标** | 窗口关闭时累计未读，展开即清零 |
| **断线重连** | WebSocket 指数退避重连（上限 8s），网络抖动自动恢复 |
| **历史回溯** | REST 拉取最近 50 条（可配至 200），与实时消息按 id 去重 |
| **自动回收** | 31 天时间窗口 + 每会话消息上限，超出即从最旧一条淘汰 |

## 二、为什么 WS 用 `?token=` 而不是 Header

浏览器原生 `WebSocket` 无法在握手时携带自定义 Header（如 `Authorization`），因此令牌通过查询参数传递：

```
ws://<host>/api/chat/ws?token=<JWT>
```

服务端用 `middleware.TokenUserID()` 校验，**与 HTTP 请求走完全相同的 JWT 校验逻辑**，不会形成旁路。
REST 聊天接口（`history` / `images` / `files`）仍使用标准 `Bearer` 头。

## 三、消息寻址模型

| 场景 | `scope` | 关键字段 |
| --- | --- | --- |
| 全院大群 | `group` | `group_key = "hospital"`，`receiver_id = 0` |
| 私聊 | `direct` | `peer_key = "<minID>-<maxID>"`，`receiver_id = 对方 ID` |

`peer_key` 由两端 ID 排序后拼接，**保证 A→B 与 B→A 落在同一会话**（有单元测试覆盖）。
私聊消息除投递给对方外，还会回显到发送者的其它标签页，保证多端一致。

## 四、接口

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/api/chat/ws?token=` | WebSocket 连接（push：message / presence / online / error） |
| GET | `/api/chat/online` | 当前在线名单 |
| GET | `/api/chat/history?group=1` | 大群历史（`&limit=`，默认 50，上限 200） |
| GET | `/api/chat/history?peer=<uid>` | 与某人的私聊历史 |
| POST | `/api/chat/images` | multipart 上传聊天图片 → `{file_name, orig_name, file_size}` |
| GET | `/api/chat/files/:file` | **鉴权**读取聊天图片 |

### 帧格式

上行（客户端 → 服务端）：
```json
{ "type": "message", "scope": "group", "content_type": "text", "content": "hello" }
{ "type": "message", "scope": "direct", "receiver_id": 3, "content_type": "image",
  "file_name": "xxx.png", "orig_name": "a.png", "file_size": 1234 }
```

下行（服务端 → 客户端）：`message`（含完整消息记录）、`presence` / `online`（在线名单）、`error`（非致命提示，如对方离线）。

## 五、消息保留策略（31 天 + LRU）

后台常驻任务（默认每小时一次，启动时也立即执行一次）按两条规则回收：

1. **时间窗口**：删除 `created_at` 早于 31 天的消息；
2. **LRU 容量**：每个会话最多保留最新 **2000** 条，超出则从最旧一条开始淘汰。

被删除消息的**图片附件会从磁盘同步移除**，并额外扫描清理无引用的孤儿文件，避免 `uploads/chat/` 无限膨胀。

可调参数（环境变量）：

| 变量 | 默认 | 说明 |
| --- | --- | --- |
| `CHAT_RETENTION_DAYS` | `31` | 保留天数 |
| `CHAT_MAX_PER_CONVERSATION` | `2000` | 每会话消息上限（LRU） |

> 策略由 `internal/chat/retention.go` 实现，并有单元测试覆盖（过期清理、LRU 上限、私聊会话、附件回收、孤儿清理、`peer_key` 对称性）。

## 六、日志轮转（log rotate）

系统自带**零依赖**轮转写盘日志（`internal/logging`），无需 lumberjack 等外部组件：

- 同时输出到 **stdout** 与文件 `logs/medical-system-YYYY-MM-DD.log`
- 单文件超过阈值自动切分，生成 `.1`、`.2` …递增分片
- 按天滚动；超过保留天数的旧日志自动删除，磁盘占用有界

| 变量 | 默认 | 说明 |
| --- | --- | --- |
| `LOG_DIR` | `./logs` | 日志目录 |
| `LOG_MAX_MB` | `64` | 单文件大小上限（MB），超出即分片 |
| `LOG_KEEP_DAYS` | `31` | 日志保留天数 |

## 七、运维建议

- WebSocket 需要反向代理显式支持 Upgrade（Nginx 需 `proxy_set_header Upgrade $http_upgrade;`）。
- 多实例部署时，当前 WS 中枢为**进程内**实现，需配置会话粘滞（sticky session），否则跨实例用户收不到实时消息；历史记录因已落库不受影响。
- 聊天图片存放在 `UPLOAD_DIR/chat/`，请与数据库一并纳入备份。
