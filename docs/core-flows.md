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

- HTTP 接口：`POST /api/v1/conversations/direct/{id}/open`
- 代码入口：
  - [conversation_handler.go](/home/zhuyin/im-system/internal/handler/conversation_handler.go)
  - [conversation_service.go](/home/zhuyin/im-system/internal/service/conversation_service.go)

### 5.2 链路步骤

1. 用户在前端点击某个好友，调用“打开单聊会话”接口。
2. `ConversationService.OpenDirectConversation` 执行业务逻辑：
   - 校验当前用户 ID 和好友 ID 不为空。
   - 通过 `FriendRepo.AreFriends` 校验双方必须已经是好友。
   - 调用 `ConversationRepo.GetOrCreateSingle` 获取或创建单聊会话。
   - 调用 `ConversationRepo.SetVisible(..., true)` 把当前用户对该会话的显示状态恢复为可见。
   - 调用 `buildConversationSummary` 组装会话摘要返回给前端。

### 5.3 数据变化

- 如果此前会话不存在，会新建：
  - `conversations`
  - `conversation_members`
- 如果会话已存在但被当前用户隐藏过：
  - 只会把当前用户对应的 `conversation_members.visible` 改回 `true`

### 5.4 关键语义

- “打开会话”不是“重新创建一个新的会话”。
- 单聊会话在当前设计中是天然幂等的，由 `single_key=min(a,b):max(a,b)` 唯一确定。

### 5.5 当前收益

- 你可以安全地从好友列表发起聊天，而不会重复造出多个单聊会话。
- 即使用户曾经隐藏过会话，也可以通过“打开会话”恢复显示。

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
  "type": "chat.send",
  "version": 1,
  "data": {
    "msg_id": "msg_xxx",
    "to": 10,
    "content": "hello"
  }
}
```

### 6.3 链路步骤

1. 用户先通过 WebSocket 建立连接。
2. `Client.ReadPump` 持续读取客户端发来的消息。
3. 服务端做第一层校验：
   - `type` 必须为 `chat.send`
   - `data.msg_id` 不能为空
   - `data.to` 不能为空
   - 当前用户和目标用户必须已经是好友
4. 服务端通过 `ConversationProvider.EnsureDirectConversationID` 获取双方单聊 `conversation_id`。
5. 服务端补齐：
   - `from`
   - `conversation_id`
   - `send_time`
6. 服务端调用 `MessageService.SaveMessage`，在一个事务里完成：
   - `MessageRepo.Create` 落库
   - `ConversationRepo.EnsureMember` 确保双方都是会话成员
   - `UpdateLastDeliveredMsgSeq` 推进接收方的 `LastDeliveredMsgSeq`
   - `SetVisible(..., true)` 恢复双方会话可见状态
7. 持久化成功后，服务端再调用 `Hub.Forward` 推送 `chat.message` 实时消息。

### 6.4 持久化与会话状态更新

`MessageService.SaveMessage` 执行以下逻辑：

1. 校验 `msg_id`、`conversation_id`、`from`、`to`、`content`
2. 解析 `conversation_id`
3. 若客户端未传 `send_time`，则补当前时间
4. 在单个数据库事务内：
   - 调用 `MessageRepo.Create` 落库
   - 通过 `ConversationRepo.EnsureMember` 确保发送方和接收方都是会话成员
   - 调用 `UpdateLastDeliveredMsgSeq` 推进接收方的 `LastDeliveredMsgSeq`
   - 调用 `SetVisible(..., true)`，确保双方会话重新显示

这一步完成后，接收方看到的实时消息和后续离线补推、已读推进，都会基于同一条已经持久化成功的消息记录。

### 6.5 实时转发链路

`Hub` 收到 `ForwardMessage` 后，会向目标用户推送 `chat.message` envelope：

- 如果目标用户当前在线且已经 ready：
  - 直接把消息写入对方连接的 `Send` 通道
- 如果目标用户已经连接但还没完成 bootstrap：
  - 先放入 `PendingMessages`
  - 等 bootstrap 完成后再补发
- 如果目标用户不在线：
  - `Hub` 不会把这条消息长期缓存在内存中
  - 后续依赖数据库中的已持久化消息 + 离线补推链路给对方补消息

### 6.6 当前实现的关键语义

- 当前发送消息没有“发送成功回执”单独返回给发送方。
- 当前发送方主要依赖：
  - 自己前端本地先渲染
  - 或后续重新拉历史消息
- 接收方收到的实时消息，已经对应一条成功入库并推进过 `LastDeliveredMsgSeq` 的记录。
- 这能避免“消息先显示，但已读推进时数据库里还没有这条消息”的竞态。

### 6.7 这条链路的工程意义

这是你项目里比较像真实 IM 系统的一条链路，因为它已经把：

- 同步持久化
- 实时转发
- 会话索引更新
- 离线补推前置条件维护

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

- `peer_id`
  - 当前要查看历史消息的好友用户 ID
- `limit`
  - 本次最多返回多少条历史消息，默认 20，最大 100
- `before_id`
  - 可选。若传入，则表示“继续查询这条消息 ID 之前的更早消息”

### 7.3 当前分页语义

历史消息现在按消息主键 `id` 做游标分页，而不是一次性把整个会话历史全拉下来：

1. 前端第一次打开会话时，不传 `before_id`
2. 后端按双方消息查询最新一页
3. 若前端继续上翻，则携带 `before_id`
4. 后端查询 `id < before_id` 的更早消息
5. SQL 在库内按 `id DESC` 取一页，再在返回前翻转成 `id ASC`
6. 所以前端渲染时看到的顺序仍然是“从旧到新”

### 7.4 返回结构

历史消息接口现在返回：

- `messages`
  - 当前这一页的消息，顺序为从旧到新
- `has_more`
  - 是否还存在更早的历史消息
- `next_before_id`
  - 如果 `has_more=true`，前端下一次继续上翻时应传入的 `before_id`

### 7.5 前端交互方式

当前前端的目标行为是接近常见聊天软件：

1. 打开会话时先拉最近一页
2. 如果当前会话存在离线补推消息，前端会先把离线消息和这页历史消息合并
3. 用户把聊天框滚到顶部时，再携带 `before_id=next_before_id` 请求更早一页
4. 新加载的一页会插入到消息列表顶部
5. 前端会修正滚动位置，避免页面突然跳动

### 7.6 为什么不用 `send_time` 做分页游标

因为 `send_time` 可能重复：

- 多条消息可能落在同一毫秒
- 如果只用时间戳做 `< before` 条件，可能出现漏消息

所以当前改成用数据库主键 `id` 做历史分页游标，这样更稳定，也和 `LastReadMsgSeq` / `LastDeliveredMsgSeq` 的消息序号语义更统一。

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
   - 找出满足 `LastReadMsgSeq < LastDeliveredMsgSeq` 的会话
   - 对每个会话查询 `(LastReadMsgSeq, LastDeliveredMsgSeq]` 区间内的消息
   - 合并所有会话的结果
   - 按 `send_time ASC`，若同时间再按 `id ASC` 排序
6. `Hub` 收到 `ClientBootstrapped` 事件后：
   - 先把离线消息刷给客户端
   - 再把 bootstrap 期间积压的 pending 消息刷给客户端
   - 最后把该用户标记为 `ready`

### 8.3 数据依赖

离线补推依赖两个核心游标：

- `LastDeliveredMsgSeq`
  表示“系统认为已经应该投递给这个用户的最新消息序号”
- `LastReadMsgSeq`
  表示“用户已经读到的最新消息序号”

离线消息补推区间是：

`LastReadMsgSeq < message.id <= LastDeliveredMsgSeq`

### 8.4 关键语义

- 离线补推不是简单地查“所有未读消息”。
- 它查的是“系统已经投递责任成立，但用户还没读掉的消息”。
- 当前项目里，消息主键 `chat_msgs.id` 就承担了消息序号 `seq` 的角色。

### 8.5 为什么当前设计合理

这样设计的好处是：

- 不需要额外维护一套独立的消息序列表。
- 查询区间简单，SQL 也更直接。
- 同一个会话内可以稳定按主键递增推进已投递和已读游标。

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
  "conversation_id": "1",
  "msg_id": "msg_xxx"
}
```

### 9.3 链路步骤

1. 客户端在某个会话中，把自己“已经读到的最后一条消息”告诉后端。
2. `MessageHandler.MarkRead` 校验：
   - `conversation_id` 不能为空
   - `msg_id` 不能为空
3. `ConversationService.MarkRead` 执行：
   - 解析 `conversation_id`
   - 用 `conversation_id + msg_id` 查询消息记录
   - 取出该消息的数据库主键 `id`
   - 调用 `ConversationRepo.UpdateLastReadMsgSeq`
4. `UpdateLastReadMsgSeq` 只会在“新消息序号比当前已读序号更大”时推进，保证单调递增。

### 9.4 数据变化

- 更新 `conversation_members.last_read_msg_seq`

### 9.5 为什么不用 `msg_id` 直接做已读游标

因为 `msg_id` 是客户端生成的业务唯一标识，它适合做幂等去重，但不适合作为范围游标。

当前实现使用数据库主键 `id` 作为已读/已投递游标更合适，原因是：

- 主键天然递增
- 易于做区间查询
- 可以直接表达“读到第几条”

### 9.6 当前收益

- 已读推进和离线补推共享同一套序号体系。
- 会话未读数、离线区间查询、最近消息索引可以共用这套基础模型。

---

## 10. 一张总的主链路脑图

如果你要用一句话概括当前系统的核心主链路，可以这样说：

> 用户先通过 HTTP 登录拿到 JWT，再通过 WebSocket 建立实时连接；好友关系通过“申请 -> 同意”形成，并同步创建单聊会话；发消息时先同步入库并推进接收方投递游标，再把已持久化的消息做实时转发；用户下次上线时，系统会根据 `LastDeliveredMsgSeq` 和 `LastReadMsgSeq` 之间的消息区间进行离线补推；已读推进则通过消息主键单调更新会话成员游标。

这句话基本就把你现在这个项目最有含金量的部分讲全了。

---

## 11. 当前实现的边界与后续可增强点

当前链路已经是完整可用的，但还有几个明确的增强方向：

- 增加发送成功 / 投递成功 / 已读回执三类前端可感知事件
- 给消息增加明确的消息类型字段，而不只是一种文本消息
- 为群聊补独立会话链路
- 给离线补推增加批量分页与断点续传语义
- 把会话列表读模型继续做强，例如置顶、免打扰、草稿、会话更新时间

如果后面你愿意，我也可以继续把这份文档再拆成两份：

- `docs/http-flows.md`
- `docs/realtime-flows.md`

这样你简历里讲项目时会更利于分层表达。
