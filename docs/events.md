# RabbitMQ Events

本文档描述当前 RabbitMQ 中的消息结构、投递路径和消费语义。

## 1. 队列概览

当前与聊天相关的队列有两个：

| 队列名 | 作用 |
| --- | --- |
| `chat_msg_queue` | 正常聊天消息入队队列 |
| `chat_msg_dead_letter_queue` | 无法被正常消费的死信队列 |

当前生产者使用默认交换机（空字符串），通过 routing key 直接投递到目标队列。

## 2. 正常消息体结构

`chat_msg_queue` 中的消息体就是 `ChatMessage` 的 JSON。

```json
{
  "msg_id": "msg_1742970000000_abcd1234",
  "conversation_id": "1",
  "from": 9,
  "to": 10,
  "send_time": 1742970000000,
  "content": "hello"
}
```

字段含义：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `msg_id` | string | 业务消息唯一 ID，用于幂等写库 |
| `conversation_id` | string | 会话 ID |
| `from` | uint64 | 发送方用户 ID |
| `to` | uint64 | 接收方用户 ID |
| `send_time` | int64 | 发送时间戳，毫秒 |
| `content` | string | 消息内容 |

说明：

- `id` 数据库主键不会出现在生产者初始投递的消息体中
- `conversation_id` 在 WebSocket 收消息入口由服务端补全
- 当前消息类型只有“聊天消息”这一类

## 3. 生产语义

生产入口：`PublishChatMsg`

发布参数：

- exchange：`""`（默认交换机）
- routing key：`chat_msg_queue`
- content_type：`application/json`
- content_encoding：`utf-8`

当前生产端语义：

- 至少尝试投递一次
- 不在发布时做业务落库
- 业务落库由消费者完成

## 4. 消费语义

消费入口：`ConsumeChatMsg`

当前配置：

- 手动 ACK
- `workerNum = 1`
- `Qos(prefetch=1)`

消费流程：

1. 从 `chat_msg_queue` 拉取消息
2. 反序列化为 `ChatMessage`
3. 调用 `MessageService.SaveMessage`
4. 成功则 `Ack`
5. 失败则按错误类型分流：
   - 永久错误：投递到死信队列，然后 `Ack`
   - 临时错误：`Nack(requeue=true)` 重新入队

## 5. 永久错误与临时错误

### 5.1 永久错误

以下错误会被视为永久错误，不会无限重试：

- `common.invalid_argument`
- `message.msg_id_required`
- `message.conversation_required`
- `conversation.member_not_found`
- `conversation.invalid_single_key`
- `friend.cannot_add_self`
- `strconv.NumError`

这类错误通常代表：

- 消息结构本身非法
- 关键字段缺失
- 会话数据不一致
- 参数格式无法恢复

### 5.2 临时错误

除永久错误外，其余错误默认视为临时错误，消息会重新入队。

典型场景：

- 数据库瞬时异常
- 网络抖动
- 中间件短暂不可用

## 6. 死信消息

死信消息会保留原始消息体，附加 headers：

| Header | 说明 |
| --- | --- |
| `reason` | 死信原因分类，例如 `invalid_json`、`permanent_save_error` |
| `error` | 原始错误信息 |
| `original_routing_key` | 原始 routing key |

示例：

```text
queue: chat_msg_dead_letter_queue
headers:
  reason: permanent_save_error
  error: conversation member not found
  original_routing_key: chat_msg_queue
body:
  {原始 ChatMessage JSON}
```

## 7. 当前消费幂等策略

消息入库使用 `msg_id` 做幂等约束：

- 同一个 `msg_id` 重复消费时，不会插入重复记录
- 仓储层会读取已存在记录并回填到当前消息对象

这意味着：

- MQ 重投不会导致重复消息落库
- 上层可以依赖 `msg_id` 做业务幂等

## 8. 后续建议

- 为不同业务事件拆分独立队列，而不是把所有事件都塞进同一个聊天队列
- 为 dead letter 增加独立观测与人工重放工具
- 给消息体增加版本字段，例如 `event_version`
- 若后续做分布式扩展，建议引入明确的事件 envelope，例如：
  - `event_id`
  - `event_type`
  - `occurred_at`
  - `producer`
  - `payload`
