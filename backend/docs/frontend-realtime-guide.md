# Frontend Realtime Guide

本文档专门说明 IM 前端的实时层实现方式，避免 WebSocket、补洞、送达和已读语义被写错。

完整消息字段以 [ws-protocol.md](/home/zhuyin/im-system/backend/docs/ws-protocol.md:1) 为准。

## 1. 基本原则

1. 服务端是真实状态源，前端是本地投影。
2. 消息顺序统一按 `seq` 处理。
3. 客户端消息幂等键统一使用 `msg_id`。
4. 单次会话补洞统一使用 `/api/v1/messages/sync`。
5. `message.sent`、`message.created`、上行 `message.delivered`、`message.read` 语义不同，不能混用。

## 2. WebSocket 建连

- 地址：`GET /api/v1/ws/`
- 认证：
  - `Authorization: Bearer <token>`
  - 或 `?token=<jwt>`

连接成功后，服务端会先补推离线消息，再进入实时转发阶段。

这意味着：

1. 前端不能把“WS 已连接”理解为“本地消息一定已经最新”。
2. 离线补推期间可能先收到若干 `message.created`。
3. 前端必须支持在连接初期就进行消息去重和排序。

## 3. 客户端发送事件

### 3.1 `message.send`

请求格式：

```json
{
  "type": "message.send",
  "data": {
    "msg_id": "msg_1742970000000_abcd1234",
    "conversation_id": 1,
    "content": "hello"
  }
}
```

规则：

1. `msg_id` 由客户端生成，必须在本地唯一。
2. `conversation_id` 来自后端会话数据。
3. `content` 当前按纯文本处理。
4. `from_user_id`、`seq`、`send_time` 都由服务端生成，前端不要本地伪造。

推荐本地流程：

1. 本地插入一条 `sending` 消息
2. 发送 `message.send`
3. 等待 `message.sent`
4. 用返回数据覆盖本地消息的服务端字段

### 3.2 `message.delivered`

```json
{
  "type": "message.delivered",
  "data": {
    "conversation_id": 1,
    "delivered_seq": 123
  }
}
```

表示当前客户端已经收到该会话的某个 `seq`。

建议：

1. 收到新消息并成功写入本地后再发送 delivered。
2. 可以做节流或批量推进，但必须保证单调递增。

### 3.3 `message.read`

```json
{
  "type": "message.read",
  "data": {
    "conversation_id": 1,
    "read_seq": 123
  }
}
```

表示用户已经真正读到该消息，而不是仅仅收到。

建议：

1. 只有消息可见且用户确实进入阅读场景时才推进。
2. 不要在消息一到就直接发 `read`。

## 4. 服务端推送事件

### 4.1 `message.sent`

这是一条发送方专属回执，表示消息已成功提交到数据库。

前端应做：

1. 把本地 `sending` 消息转为 `sent`
2. 同步服务端返回的 `id`、`seq`、`send_time`
3. 用 `msg_id` 做本地消息关联

### 4.2 `message.created`

这是一条接收侧消息事件，同时也用于离线补推。

关键字段：

- `msg_id`
- `conversation_id`
- `seq`
- `from_user_id`
- `content`
- `extra`

前端应做：

1. 先按 `conversation_id` 分发到对应会话
2. 再按 `msg_id` 去重
3. 最终按 `seq` 排序
4. 更新该会话最新消息与未读状态

### 4.3 `message.read`

```json
{
  "type": "message.read",
  "data": {
    "conversation_id": 1,
    "user_id": 2,
    "read_seq": 123
  }
}
```

表示某个用户已经读到 `read_seq`。

前端应做：

1. 更新阅读进度 UI
2. 单聊和群聊都按相同语义处理

### 4.4 `presence.changed`

```json
{
  "type": "presence.changed",
  "data": {
    "user_id": 2,
    "online": true
  }
}
```

只表示好友在线状态变化。

前端应做：

1. 更新好友列表和会话列表里的在线状态
2. 不要把它理解成用户资料变更事件

### 4.5 `sync.required`

```json
{
  "type": "sync.required",
  "data": {
    "conversation_id": 1,
    "reason": "pending_queue_full"
  }
}
```

这是最容易被忽略、但必须支持的事件。

收到后应做：

1. 如果有 `conversation_id`，优先补该会话。
2. 如果 `conversation_id` 为空，做更宽泛的会话补齐检查。
3. 调用 `/api/v1/messages/sync?conversation_id=...&after_seq=...`
4. 用返回消息按 `msg_id` 去重、按 `seq` 合并。

不要做：

1. 不要忽略该事件。
2. 不要在本地假设缺失消息一定不存在。

## 5. 推荐本地状态设计

每个会话建议至少维护：

- `conversationId`
- `messages`
- `maxSeq`
- `lastReadSeqByUser`
- `hasMoreHistory`

每条消息建议至少维护：

- `msgId`
- `seq`
- `conversationId`
- `fromUserId`
- `content`
- `sendTime`
- `status`

其中 `status` 为前端本地状态，建议：

- `sending`
- `sent`
- `failed`

## 6. 推荐排序与去重规则

### 6.1 去重

优先级建议：

1. 优先按 `msg_id` 去重
2. 同步或回放场景下，如果同一条消息出现多次，以服务端最新版本覆盖本地临时版本

### 6.2 排序

会话内消息排序：

1. 优先按 `seq ASC`
2. 不依赖 `send_time`

会话列表排序：

1. 按最后一条消息的时间或 `seq` 派生排序
2. 具体 UI 规则由前端决定

## 7. 页面行为建议

### 7.1 打开会话

1. 调用 `/api/v1/conversations/:id/open`
2. 获取 `latest_read_state`
3. 再拉历史消息
4. 用返回的 `read_by_user_ids` 和 `latest_sent_seq` 初始化阅读态 UI

### 7.2 进入后台或重连

1. 重连后接受离线补推
2. 收到 `sync.required` 再补拉
3. 不要因为本地已有旧缓存就跳过补齐

### 7.3 发送失败

若 WS 断开、超时或服务端错误：

1. 本地消息改为 `failed`
2. 提供重试能力
3. 重试时沿用原 `content`，重新生成新的 `msg_id`

## 8. 大模型实现时的额外提示

如果把这部分交给大模型，请额外强调：

1. 不要自己脑补 ack 机制。
2. 不要在客户端维护另一套“消息主键”协议。
3. 不要把 delivered/read 写成接口轮询。
4. 不要忽略 `sync.required`。
5. 不要引入 `public_id`。

## 9. 最低验收标准

实时层至少应满足：

1. 能成功连接 WS
2. 能发送消息并收到 `message.sent`
3. 能接收他人 `message.created`
4. 能发送 `message.delivered`
5. 能发送 `message.read`
6. 能处理 `presence.changed`
7. 能处理 `sync.required`
8. 断线重连后不会明显丢消息
