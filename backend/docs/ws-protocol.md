# WebSocket Protocol

本文档描述 `/api/v1/ws/` WebSocket 连接的消息格式、事件类型和字段约束。

所有消息统一使用 envelope：

```json
{
  "type": "message.created",
  "data": {}
}
```

## 1. 连接

- 路径：`GET /api/v1/ws/`
- 认证方式：`Authorization: Bearer <token>` 或 `?token=<jwt>`
- 连接完成后，服务端先补推离线消息，再进入实时转发状态。

## 2. 客户端事件

### message.send

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

- `msg_id` 是客户端生成的全局幂等键。
- `conversation_id` 是目标会话 ID。
- `content` 是文本内容。
- 服务端只信任当前连接身份，`from`、`send_time`、`seq` 均由服务端生成。
- 数据库事务提交成功后，发送方收到 `message.sent`，其他在线成员收到 `message.created`。

### message.delivered

```json
{
  "type": "message.delivered",
  "data": {
    "conversation_id": 1,
    "delivered_seq": 123
  }
}
```

- 表示客户端已经确认收到该会话内的 `delivered_seq`。
- 服务端单调推进 `conversation_members.last_acked_msg_seq`。
- `delivered_seq` 不能超过当前会话 MySQL 已提交的最大 `seq`。

### message.read

```json
{
  "type": "message.read",
  "data": {
    "conversation_id": 1,
    "read_seq": 123
  }
}
```

- 表示用户已经读到该会话内的 `read_seq`。
- `read_seq` 必须大于入会时的 `joined_msg_seq`，且不能超过该用户的 `last_acked_msg_seq`。
- 服务端单调推进 `conversation_members.last_read_msg_seq`。

## 3. 服务端事件

### message.sent

发送方专属回执，表示消息已经完成 DB commit。

### message.created

接收方消息事件，实时推送和离线补推都使用同一 payload：

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

`seq` 是唯一业务游标，history、sync、离线补推、delivered、read 都基于它。

### message.delivered

```json
{
  "type": "message.delivered",
  "data": {
    "conversation_id": 1,
    "user_id": 100000010,
    "delivered_seq": 123
  }
}
```

服务端在接收确认成功后向会话在线成员广播该回执。

### message.read

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

服务端在已读推进成功后向会话在线成员广播该回执。

### error / presence.changed

- `error` 携带 `{code, message}`。
- `presence.changed` 携带 `{user_id, online}`，只表示好友在线状态变化。

### sync.required

```json
{
  "type": "sync.required",
  "data": {
    "conversation_id": 1,
    "reason": "pending_queue_full"
  }
}
```

- 当实时链路出现写队列溢出、bootstrap pending 溢出或内部转发拥塞时，服务端会下发该事件。
- `conversation_id` 可能为空，表示当前连接需要做一次更宽泛的补齐检查。
- 客户端收到后应尽快调用 `/api/v1/messages/sync` 补洞，然后继续接收后续实时消息。

## 4. 离线补推

- 离线补推只读 MySQL 已提交消息。
- 每个会话以 `max(joined_msg_seq, last_acked_msg_seq)` 为起点，读取该会话当前最大 `seq` 之前的消息。
- 不按发送人过滤；是否需要展示或去重由客户端按 `msg_id` 和本地状态处理。
- 多会话结果按 `send_time ASC`，同时间按 `conversation_id ASC, seq ASC` 合并。

## 5. 前端建议

- 本地发送后先标记 `sending`，收到 `message.sent` 后标记 `sent`。
- 收到 `message.created` 后按会话内单调递增的 `seq` 维护本地游标，并发送 `message.delivered`。
- 只有用户实际读到消息时才发送 `message.read`。
- 缺口补拉使用 `/api/v1/messages/sync?conversation_id=...&after_seq=...`。
- 不使用 `id`、`send_time`、`msg_id` 作为范围游标。
