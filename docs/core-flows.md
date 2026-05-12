# IM System Core Flows

本文档描述当前项目已经实现的核心业务链路，重点回答两个问题：

1. 一条业务请求从哪里进来，经过哪些模块，最后落到哪里。
2. 当前实现的关键语义是什么，哪些点是“刻意这样设计”的，哪些点是后续可以继续增强的。

这份文档面向两个场景：

- 你自己后续继续开发时，用来快速回忆主链路。
- 面试或项目介绍时，用来讲清楚“这个 IM 系统现在已经具备哪些真实能力”。

---

## 1. 总体说明

当前项目里，核心链路主要涉及下面几个模块：

- `handler`
  HTTP / WebSocket 接口入口，负责参数绑定、读取鉴权上下文、统一响应。
- `middleware`
  负责 JWT 鉴权和错误日志补充。
- `service`
  负责认证、好友、好友申请、会话、消息等核心业务逻辑。
- `repository`
  负责 MySQL / Redis 数据访问。
- `ws`
  负责连接管理、在线状态、实时转发、离线补推。

当前系统中的几个关键事实：

- “登录成功”只代表拿到了 JWT，不代表用户已经在线。
- 用户是否在线，由 WebSocket 建连成功后 `Hub` 写入 `PresenceRepo` 决定。
- 好友关系建立成功时，会默认创建或恢复单聊会话。
- 消息会先完成数据库持久化和会话游标推进，再进入实时转发链路。
- 离线消息不是靠 `Hub` 在内存里长期缓存，而是靠数据库中的消息记录和会话游标在用户上线时补推。

---

## 2. 登录

### 2.1 入口

- HTTP 接口：`POST /api/v1/auth/login`
- 代码入口：
  - [auth_handler.go](/home/zhuyin/im-system/internal/handler/auth_handler.go)
  - [auth_service.go](/home/zhuyin/im-system/internal/service/auth_service.go)

### 2.2 链路步骤

1. 客户端提交 `username`、`password`。
2. `AuthHandler.Login` 负责解析 JSON 请求体。
3. `AuthService.Login` 执行登录逻辑：
   - 校验用户名和密码不能为空。
   - 通过 `UserRepo.GetByUsername` 查询用户。
   - 使用 `utils.VerifyPassword` 校验密码哈希。
   - 调用 `jwt.GenerateJWT` 生成 token。
4. 后端返回 JWT 给客户端。

### 2.3 数据变化

- 登录接口本身不会修改用户表。
- 登录接口本身也不会把用户标记为在线。
- 如果后续调用了登出接口：
  - `POST /api/v1/auth/logout`
  - 会把当前 token 的 `jti` 写入黑名单。

### 2.4 关键语义

- “HTTP 登录成功”与“IM 在线”是两件事。
- 当前项目中，在线状态是在 WebSocket 建连时由 `Hub` 写入的，而不是在登录接口里写入。

### 2.5 为什么这样设计

这是比较合理的做法，因为 IM 的“在线”本质上是“是否持有一个可用的实时连接”。

如果只在 HTTP 登录时把用户标成在线，会有两个问题：

- 用户拿到 token 但没有建立 WebSocket，状态会被错误地视为在线。
- token 可能长期有效，但实时连接可能早就断了，在线状态会失真。

---

## 3. 加好友

### 3.1 入口

- HTTP 接口：`POST /api/v1/friend-requests/{id}`
- 代码入口：
  - [friend_request_handler.go](/home/zhuyin/im-system/internal/handler/friend_request_handler.go)
  - [friend_request_service.go](/home/zhuyin/im-system/internal/service/friend_request_service.go)

### 3.2 链路步骤

1. 当前用户向目标用户发起好友申请。
2. `FriendRequestHandler.Send` 读取路径参数中的目标用户 ID，并解析可选附言。
3. `FriendRequestService.Send` 执行业务判断：
   - 校验 `requester_id`、`receiver_id` 不能为空。
   - 不能给自己发好友申请。
   - 目标用户必须存在。
   - 如果双方已经是好友，直接返回 `already_friends`。
   - 如果发现“对方之前已经给我发过一个待处理申请”，则直接走自动同意逻辑，返回 `auto_accepted`。
   - 如果我已经发过待处理申请，则返回 `pending`。
   - 否则，新建一条 `friend_requests` 记录，状态为 `pending`。

### 3.3 数据变化

常规发起申请时：

- 写入 `friend_requests` 表一条待处理申请。

反向自动同意时：

- 不再新增申请。
- 会直接进入“同意申请”的后半段逻辑：
  - 建立好友关系
  - 创建/恢复会话
  - 把双方相关 pending 申请统一改成 `accepted`

### 3.4 关键语义

- 当前“加好友”不是直接写 `friends` 表，而是先进入好友申请流。
- 这是标准 IM 处理方式，比“点一下直接加成功”更符合真实产品逻辑。

### 3.5 当前实现的优点

- 支持待处理申请去重。
- 支持“双方同时互加”时自动收敛成一条已接受结果，避免形成两条独立申请。
- 好友关系和会话建立是统一闭环，不会出现“成为好友但没有会话”的问题。

---

## 4. 同意申请

### 4.1 入口

- HTTP 接口：`POST /api/v1/friend-requests/{id}/accept`
- 代码入口：
  - [friend_request_handler.go](/home/zhuyin/im-system/internal/handler/friend_request_handler.go)
  - [friend_request_service.go](/home/zhuyin/im-system/internal/service/friend_request_service.go)
  - [friend_service.go](/home/zhuyin/im-system/internal/service/friend_service.go)

### 4.2 链路步骤

1. 被申请人调用“同意申请”接口。
2. `FriendRequestService.Accept` 先读取申请记录。
3. 做三类关键校验：
   - 申请必须存在。
   - 当前用户必须是该申请的接收方。
   - 当前申请必须仍然是 `pending` 状态。
4. 调用 `FriendService.AddFriend`：
   - 通过 `FriendRepo.AddPair` 建立双向好友关系。
   - 通过 `ConversationRepo.GetOrCreateSingle` 创建或获取单聊会话。
   - 调用 `ConversationRepo.SetVisible(..., true)`，保证双方会话都显示在会话列表里。
5. 调用 `FriendRequestRepo.ResolvePendingBetween`：
   - 把双方之间所有 pending 申请统一改为 `accepted`。

### 4.3 数据变化

- `friends` 表中新增两条记录：
  - `A -> B`
  - `B -> A`
- `conversations` 表中创建或复用单聊会话。
- `conversation_members` 表中确保双方成员都存在且 `visible=true`。
- `friend_requests` 表中双方之间所有 pending 申请统一更新为 `accepted`，并写入 `handled_at`。

### 4.4 关键语义

- 同意申请不仅仅是“改申请状态”。
- 真正的业务闭环是：
  - 建好友关系
  - 建/恢复会话
  - 清理 pending 申请

### 4.5 为什么这样设计

如果只改申请状态，不同时建立好友和会话，就会出现下面这些产品问题：

- 前端显示“已同意”，但双方实际不能聊天。
- 会话列表为空，用户还得自己额外触发一次“创建会话”。
- 双向 pending 申请残留，后面查询申请列表会混乱。

---

## 5. 打开会话

### 5.1 入口

- HTTP 接口：`POST /api/v1/conversations/{id}/open`
- 代码入口：
  - [conversation_handler.go](/home/zhuyin/im-system/internal/handler/conversation_handler.go)
  - [conversation_service.go](/home/zhuyin/im-system/internal/service/conversation_service.go)

### 5.2 链路步骤

1. 用户在前端点击某个会话入口。
2. 这个入口可以来自：
   - 消息栏中的已有会话
   - 好友列表中对应单聊的 `conversation_id`
   - 群聊列表中对应群会话的 `conversation_id`
3. `ConversationService.OpenConversation` 执行业务逻辑：
   - 校验当前用户是否是该会话的有效成员。
   - 如果当前用户此前把该会话隐藏过，则调用 `ConversationRepo.SetVisible(..., true)` 恢复显示。
   - 调用 `buildConversationSummary` 组装会话摘要返回给前端。

### 5.3 数据变化

- 打开会话本身不会新建单聊或群聊。
- 如果会话已存在但被当前用户隐藏过：
  - 只会把当前用户对应的 `conversation_members.visible` 改回 `true`

### 5.4 关键语义

- “打开会话”只负责恢复和进入，不负责创建。
- 单聊会话在成为好友时就应已自动创建并稳定绑定到 `conversation_id`。

### 5.5 当前收益

- 前端所有聊天入口都可以统一到 `conversation_id`。
- 即使用户曾经隐藏过会话，也可以通过一次标准 `OpenConversation` 恢复显示。

---

## 6. 发消息

### 6.1 入口

- WebSocket：`GET /api/v1/ws/` 建连后，客户端发送 JSON 消息
- 代码入口：
  - [ws_handler.go](/home/zhuyin/im-system/internal/handler/ws_handler.go)
  - [client.go](/home/zhuyin/im-system/internal/ws/client.go)
  - [message_service.go](/home/zhuyin/im-system/internal/service/message_service.go)

### 6.2 客户端发送格式

当前客户端发送的最小消息体是：

```json
{
  "type": "message.send",
  "version": 1,
  "data": {
    "msg_id": "msg_xxx",
    "conversation_id": 12,
    "content": "hello"
  }
}
```

### 6.3 链路步骤

1. 用户先通过 WebSocket 建立连接。
2. `Client.ReadPump` 持续读取客户端发来的消息。
3. 服务端做第一层校验：
   - `type` 必须为 `message.send`
   - `data.msg_id` 不能为空
   - `data.conversation_id` 不能为空
   - 当前用户必须是该会话的活跃成员
4. 服务端补齐：
	   - `from`
	   - `send_time`（统一由服务端生成）
	5. 服务端调用 `MessageSendService.SendTextMessage`，在一个事务里完成：
	   - `MessageRepo.Create` 落库
	   - `ListActiveMembers` 找到该会话当前应接收消息的成员
	   - 推进发送方自己的 `last_acked_msg_seq` 和 `last_read_msg_seq`
	   - `SetVisible(..., true)` 恢复活跃成员会话可见状态
6. 持久化成功后，服务端向发送方推送 `message.sent`，向其他在线活跃成员推送 `message.created`。

### 6.4 持久化与会话状态更新

`MessageSendService.SendTextMessage` 执行以下逻辑：

1. 校验 `msg_id`、`conversation_id`、`from`、`content`
2. 校验发送方必须是该会话的活跃成员
3. 统一由服务端生成 `send_time`
	4. 在单个数据库事务内：
	   - 调用 `MessageRepo.Create` 落库
	   - 查询会话全部活跃成员
	   - 推进发送方自己的送达和已读游标
	   - 调用 `SetVisible(..., true)`，确保当前活跃成员会话重新显示

这一步完成后，群聊和单聊都以同一条已提交消息作为事实来源；接收方送达和已读由后续 `message.delivered` / `message.read` 推进。

### 6.5 实时转发链路

`Hub` 收到 `ForwardMessage` 后，会向目标用户推送对应 envelope：

- 如果目标用户当前在线且已经 ready：
  - 直接把消息写入对方连接的 `Send` 通道
- 如果目标用户已经连接但还没完成 bootstrap：
  - 先放入 `PendingMessages`
  - 等 bootstrap 完成后再补发
- 如果目标用户不在线：
  - `Hub` 不会把这条消息长期缓存在内存中
  - 后续依赖数据库中的已持久化消息 + 离线补推链路给对方补消息

### 6.6 当前实现的关键语义

- 发送方收到 `message.sent`，表示消息已经完成 DB commit。
- 接收方收到 `message.created`，表示消息已经进入 MySQL 真源。
- 接收方收到后发送 `message.delivered`，服务端单调推进 `last_acked_msg_seq`。
- 用户读到消息后发送 `message.read` 或调用 HTTP 已读接口，服务端单调推进 `last_read_msg_seq`。

### 6.7 这条链路的工程意义

这是你项目里比较像真实 IM 系统的一条链路，因为它已经把：

	- 同步持久化
	- 实时转发
	- 会话索引更新
	- ACK/已读游标推进

	串成了一条完整闭环。

---

## 7. 历史消息加载

### 7.1 入口

- HTTP 接口：`GET /api/v1/messages/history`
- 代码入口：
  - [message_handler.go](/home/zhuyin/im-system/internal/handler/message_handler.go)
  - [message_service.go](/home/zhuyin/im-system/internal/service/message_service.go)
  - [message_repo.go](/home/zhuyin/im-system/internal/repository/message_repo.go)

### 7.2 查询参数

- `conversation_id`
  - 当前要查看历史消息的会话 ID
- `limit`
  - 本次最多返回多少条历史消息，默认 20，最大 100
- `before_seq`
  - 可选。若传入，则表示“继续查询这条消息 seq 之前的更早消息”

### 7.3 当前分页语义

历史消息按会话内业务序号 `seq` 做游标分页，而不是一次性把整个会话历史全拉下来：

1. 前端第一次打开会话时，不传 `before_seq`
2. 后端按会话消息查询最新一页
3. 若前端继续上翻，则携带 `before_seq`
4. 后端查询 `seq < before_seq` 的更早消息
5. SQL 在库内按 `seq DESC` 取一页，再在返回前翻转成 `seq ASC`
6. 所以前端渲染时看到的顺序仍然是“从旧到新”

### 7.4 返回结构

历史消息接口现在返回：

- `messages`
  - 当前这一页的消息，顺序为从旧到新
- `has_more`
  - 是否还存在更早的历史消息
- `next_before_seq`
  - 如果 `has_more=true`，前端下一次继续上翻时应传入的 `before_seq`

### 7.5 前端交互方式

当前前端的目标行为是接近常见聊天软件：

1. 打开会话时先拉最近一页
2. 如果当前会话存在离线补推消息，前端会先把离线消息和这页历史消息合并
3. 用户把聊天框滚到顶部时，再携带 `before_seq=next_before_seq` 请求更早一页
4. 新加载的一页会插入到消息列表顶部
5. 前端会修正滚动位置，避免页面突然跳动

### 7.6 为什么只用 `seq` 做分页游标

因为 `send_time` 可能重复：

- 多条消息可能落在同一毫秒
- 如果只用时间戳做 `< before` 条件，可能出现漏消息

`seq` 是会话内单调递增的业务序号，和 sync、delivered、read 共用同一套语义；数据库主键 `id` 只用于定位，不承担业务顺序。

---

## 8. 离线补推

### 8.1 触发时机

- 用户建立 WebSocket 连接并成功注册到 `Hub` 时触发
- 代码入口：
  - [hub.go](/home/zhuyin/im-system/internal/ws/hub.go)
  - [conversation_service.go](/home/zhuyin/im-system/internal/service/conversation_service.go)

### 8.2 链路步骤

1. `Hub.Register` 收到新客户端。
2. `Hub` 会先把该用户放进 `Clients`，并把 `ReadyClients[userID]` 设为 `false`。
3. 异步执行 `initClient`：
   - `PresenceRepo.SetOnline`
   - 广播该用户上线状态
   - 调用 `OfflineLoader.ListOfflineMessages`
4. 当前 `OfflineLoader` 实现是 `ConversationService.ListOfflineMessages`。
5. `ListOfflineMessages` 会：
   - 查询当前用户参与的所有 `conversation_members`
   - 以 `max(joined_msg_seq, last_acked_msg_seq)` 为起点
   - 对每个会话查询 `(after_seq, 当前最大 seq]` 区间内的 MySQL 已提交消息
   - 合并所有会话的结果
   - 按 `send_time ASC`，若同时间再按 `conversation_id ASC, seq ASC` 排序
6. `Hub` 收到 `ClientBootstrapped` 事件后：
   - 先把离线消息刷给客户端
   - 再把 bootstrap 期间积压的 pending 消息刷给客户端
   - 最后把该用户标记为 `ready`

### 8.3 数据依赖

离线补推依赖两个核心值：

- `last_acked_msg_seq`：客户端已连续收到的最大 `seq`
- `joined_msg_seq`：成员可见历史的起点

离线消息补推区间是：`max(joined_msg_seq, last_acked_msg_seq) < seq <= current_max_seq`。

### 8.4 关键语义

- 离线补推不是查未读消息，而是查客户端尚未确认收到的消息。
- 是否已读只影响未读数和 read 回执，不影响离线补推事实范围。
- MySQL `messages` 是唯一真源；Redis/recent cache 不定义消息事实。

### 8.5 为什么当前设计合理

这样设计的好处是：

- history、sync、实时流、离线补推都围绕同一个 `seq`。
- 断线重连只需要从 `last_acked_msg_seq` 后继续补拉。
- 已读游标不会导致未读但已送达的消息被重复补推。

---

## 9. 已读推进

### 9.1 入口

- HTTP 接口：`POST /api/v1/messages/read`
- 代码入口：
  - [message_handler.go](/home/zhuyin/im-system/internal/handler/message_handler.go)
  - [conversation_service.go](/home/zhuyin/im-system/internal/service/conversation_service.go)

### 9.2 请求格式

```json
{
  "conversation_id": 1,
  "read_seq": 123
}
```

### 9.3 链路步骤

1. 客户端在某个会话中，把自己“已经读到的最后一条消息”告诉后端。
2. `MessageHandler.MarkRead` 校验：
   - `conversation_id` 不能为空
   - `read_seq` 不能为空
3. `ConversationService.MarkRead` 执行：
   - 校验当前用户是会话活跃成员
   - 校验 `read_seq > joined_msg_seq`
   - 校验 `read_seq <= last_acked_msg_seq`
   - 调用 `ConversationRepo.UpdateLastReadMsgSeq`
4. `UpdateLastReadMsgSeq` 只会在“新消息序号比当前已读序号更大”时推进，保证单调递增。

### 9.4 数据变化

- 更新 `conversation_members.last_read_msg_seq`

### 9.5 为什么用 `read_seq` 做已读游标

因为 `msg_id` 是客户端生成的业务唯一标识，它适合做幂等去重，但不适合作为范围游标。

当前实现使用会话内 `seq` 作为已读/送达游标，原因是：

- `seq` 在会话内严格递增
- 易于做区间查询
- 可以和 history、sync、离线补推共用同一套语义

### 9.6 当前收益

- 已读推进和离线补推共享同一套序号体系。
- 会话未读数、离线区间查询、实时回执可以共用这套基础模型。

---

## 10. 一张总的主链路脑图

如果你要用一句话概括当前系统的核心主链路，可以这样说：

> 用户先通过 HTTP 登录拿到 JWT，再通过 WebSocket 建立实时连接；好友关系通过“申请 -> 同意”形成，并同步创建单聊会话；发消息时先同步入库，提交成功后向发送方返回 `message.sent`、向接收方转发 `message.created`；接收方用 `message.delivered` 推进送达游标，用 `message.read` 或 HTTP read 推进已读游标；用户重连时系统根据 `seq` 从 MySQL 真源补推尚未确认收到的消息。

这句话基本就把你现在这个项目最有含金量的部分讲全了。

---

## 11. 当前实现的边界与后续可增强点

当前链路已经是完整可用的，但还有几个明确的增强方向：

- 完善多端同步下的发送成功 / 投递成功 / 已读回执展示
- 给消息增加明确的消息类型字段，而不只是一种文本消息
- 为群聊补独立会话链路
- 给离线补推增加批量分页与断点续传语义
- 把会话列表读模型继续做强，例如置顶、免打扰、草稿、会话更新时间

如果后面你愿意，我也可以继续把这份文档再拆成两份：

- `docs/http-flows.md`
- `docs/realtime-flows.md`

这样你简历里讲项目时会更利于分层表达。
