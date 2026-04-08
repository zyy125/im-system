# WebSocket Protocol

本文档描述 `/api/v1/ws/` WebSocket 连接的消息格式、事件类型和字段约束。

当前协议统一使用 envelope 结构：

```json
{
  "type": "chat.message",
  "version": 1,
  "data": {}
}
```

字段说明：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `type` | string | 消息类型 |
| `version` | int | 协议版本，当前固定为 `1` |
| `data` | object | 对应类型的业务负载 |

## 1. 连接方式

- 路径：`GET /api/v1/ws/`
- 认证方式：
  - `Authorization: Bearer <token>`
  - 或查询参数：`?token=<jwt>`
- 连接成功后，服务端会维护该用户的在线状态，并在初始化完成后推送：
  - 离线消息
  - 好友在线状态变化事件

## 2. 客户端发送消息

客户端发送的是一个 envelope，当前支持“单聊消息发送”。

### 2.1 请求格式

```json
{
  "type": "chat.send",
  "version": 1,
  "data": {
    "msg_id": "msg_1742970000000_abcd1234",
    "to": 10,
    "content": "hello",
    "send_time": 1742970000000
  }
}
```

### 2.2 字段说明

`data` 字段说明：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `msg_id` | string | 是 | 客户端生成的消息唯一标识。全局要求唯一，用于幂等去重。 |
| `to` | uint64 | 是 | 接收方用户 ID。必须为当前用户的好友。 |
| `content` | string | 是 | 消息内容。当前为普通文本消息。 |
| `send_time` | int64 | 否 | 客户端发送时间戳，毫秒。若不传，服务端会补充当前时间。 |

### 2.3 服务端处理规则

- 服务端会强制覆盖/补充以下字段：
  - `from`：取当前已认证用户 ID
  - `conversation_id`：由服务端根据双方用户 ID 获取或创建单聊会话
  - `send_time`：若客户端未传，则由服务端补齐
- 当前不信任客户端传入的 `from`、`conversation_id`
- 若好友关系校验失败，服务端不会转发，也不会入库

## 3. 服务端推送消息

服务端当前会推送三类内容：

1. 聊天消息
2. 错误事件
3. 在线状态事件

### 3.1 聊天消息

聊天消息使用 `chat.message` envelope。

```json
{
  "type": "chat.message",
  "version": 1,
  "data": {
    "id": 123,
    "msg_id": "msg_1742970000000_abcd1234",
    "conversation_id": "1",
    "from": 9,
    "to": 10,
    "send_time": 1742970000000,
    "content": "hello"
  }
}
```

`data` 字段说明：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `id` | uint64 | 数据库主键，自增消息序号，也是会话内游标推进依据。 |
| `msg_id` | string | 客户端生成的业务消息 ID。 |
| `conversation_id` | string | 会话 ID。当前单聊会话由服务端创建并返回。 |
| `from` | uint64 | 发送方用户 ID。 |
| `to` | uint64 | 接收方用户 ID。 |
| `send_time` | int64 | 发送时间戳，毫秒。 |
| `content` | string | 消息内容。 |

### 3.2 错误事件

当客户端发送非法消息、非好友发消息等情况时，服务端会推送错误事件。

```json
{
  "type": "error",
  "version": 1,
  "data": {
    "code": "message.invalid_payload",
    "message": "invalid message"
  }
}
```

`data` 字段说明：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `code` | string | 业务错误码 |
| `message` | string | 面向前端展示的错误信息 |

### 3.3 在线状态事件

当好友上线/下线时，服务端会向在线好友推送 presence 事件。

```json
{
  "type": "presence.changed",
  "version": 1,
  "data": {
    "user_id": 10,
    "online": true
  }
}
```

`data` 字段说明：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `user_id` | uint64 | 状态发生变化的用户 ID |
| `online` | bool | `true` 表示上线，`false` 表示下线 |

## 4. 离线消息补推

用户连接建立后，Hub 初始化流程会先补推离线消息，再开始实时转发。

当前离线补推范围：

- 按会话维度查询当前用户 `LastReadMsgSeq < 消息主键 ID <= LastDeliveredMsgSeq` 的消息
- 多个会话的离线消息会按：
  - `send_time` 升序
  - 若时间相同则 `id` 升序
  进行合并排序后推送

## 5. 前端建议

- 先按 envelope 的 `type` 分发，再解析 `data`
- 聊天消息使用 `data.msg_id` 做业务去重
- 错误事件根据 `data.code` 分支处理，不要依赖中文 `data.message`
- `presence.changed` 事件只更新好友在线状态，不应作为聊天消息渲染
- 客户端重连后要能正确处理离线补推与实时消息混合到达
