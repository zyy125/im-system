# WebSocket Protocol

## 1. 文档目的

本文档定义 `/api/v1/ws/` WebSocket 连接的消息 envelope、客户端事件、服务端事件和离线补推规则。

## 2. 适用范围

本文档覆盖以下内容：

- 连接入口与认证方式
- 客户端上行事件
- 服务端下行事件
- 离线补推与 `sync.required`

本文档不覆盖：

- HTTP 接口字段定义
- 错误码全集
- 会话与群聊的业务规则细节

## 3. 消息格式

所有消息统一使用以下 envelope：

```json
{
  "type": "message.created",
  "data": {}
}
```

字段说明：

- `type`
  事件类型。
- `data`
  事件载荷。

## 4. 连接

- 路径：`GET /api/v1/ws/`
- 认证方式：`Authorization: Bearer <token>` 或 `?token=<jwt>`
- 连接完成后，服务端先补推离线消息，再进入实时转发状态

## 5. 客户端事件

### 5.1 `message.send`

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

约束如下：

- `msg_id` 为客户端生成的全局幂等键
- `conversation_id` 为目标会话 ID
- `content` 为文本内容
- 服务端仅信任当前连接身份，`from`、`send_time`、`seq` 均由服务端生成
- 数据库事务提交成功后，发送方收到 `message.sent`，其他在线成员收到 `message.created`

### 5.2 `message.delivered`

```json
{
  "type": "message.delivered",
  "data": {
    "conversation_id": 1,
    "delivered_seq": 123
  }
}
```

约束如下：

- `delivered_seq` 表示客户端已经确认收到的会话内最大 `seq`
- 服务端单调推进 `conversation_members.last_acked_msg_seq`
- `delivered_seq` 不得超过当前会话 MySQL 已提交的最大 `seq`
- 服务端不下发 `message.delivered` 事件

### 5.3 `message.read`

```json
{
  "type": "message.read",
  "data": {
    "conversation_id": 1,
    "read_seq": 123
  }
}
```

约束如下：

- `read_seq` 表示当前用户已经读到的会话内最大 `seq`
- `read_seq` 必须大于入会时的 `joined_msg_seq`
- `read_seq` 不得超过该用户的 `last_acked_msg_seq`
- 服务端单调推进 `conversation_members.last_read_msg_seq`

## 6. 服务端事件

### 6.1 `message.sent`

发送方专属回执，表示消息已经完成数据库提交。

### 6.2 `message.created`

接收方消息事件。实时推送和离线补推均使用以下 payload：

```json
{
  "type": "message.created",
  "data": {
    "id": 123,
    "msg_id": "msg_1742970000000_abcd1234",
    "conversation_id": 1,
    "seq": 456,
    "type": 1,
    "event": "",
    "from_user_id": 100000009,
    "send_time": 1742970000000,
    "content": "hello",
    "extra": null
  }
}
```

`seq` 是统一业务游标，用于：

- history
- sync
- 离线补推
- delivered
- read

### 6.3 `message.read`

```json
{
  "type": "message.read",
  "data": {
    "conversation_id": 1,
    "user_id": 100000010,
    "read_seq": 123
  }
}
```

服务端在已读推进成功后向会话在线成员广播该事件。

### 6.4 `error`

`error` 事件载荷格式为：

- `{code, message}`

### 6.5 `presence.changed`

`presence.changed` 事件载荷格式为：

- `{user_id, online}`

该事件仅表示好友在线状态变化。

### 6.6 `sync.required`

```json
{
  "type": "sync.required",
  "data": {
    "conversation_id": 1,
    "reason": "pending_queue_full"
  }
}
```

约束如下：

- 当实时链路出现写队列溢出、bootstrap pending 溢出或内部转发拥塞时，服务端下发该事件
- `conversation_id` 可为空，表示当前连接需要执行更宽范围的补齐检查
- 客户端收到后应尽快调用 `/api/v1/messages/sync`

## 7. 离线补推

离线补推规则如下：

- 只读取 MySQL 已提交消息
- 每个会话以 `max(joined_msg_seq, last_acked_msg_seq)` 为起点
- 不按发送人过滤
- 多会话结果按 `send_time ASC`，同时间按 `conversation_id ASC, seq ASC` 合并

## 8. 相关文档

- [../../docs/websocket-flow.md](../../docs/websocket-flow.md)
- [error-codes.md](./error-codes.md)
