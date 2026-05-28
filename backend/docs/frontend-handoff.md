# Frontend Handoff

本文档用于把当前后端能力交给前端实现方，尤其适用于交给大模型生成前端代码时作为约束说明。

目标不是替代 Swagger，而是明确前端实现边界、主流程、字段语义和禁止事项，避免前端自行猜测协议。

## 1. 总原则

1. 用户唯一标识统一使用 `user_id`。
2. 登录凭证统一使用 `username + password`。
3. 前端不得再引入、兼容或推断 `public_id`。
4. 会话消息的唯一业务游标统一使用 `seq`。
5. HTTP 负责资源查询和补拉，WebSocket 负责实时事件和回执。
6. 后端返回字段名必须原样使用，前端不要自行改名或包装成另一套协议。

## 2. 必须遵守的约束

### 2.1 身份与用户字段

- 用户主键是 `user_id`，不是用户名，也不是历史上的 `public_id`。
- 所有用户相关对象都按 `user_id` 建立引用关系。
- `username` 用于登录和展示，不是业务主键。
- `avatar` 直接使用服务端返回路径，不要在前端拼接规则。

### 2.2 消息与游标

- 会话消息顺序只看 `seq`。
- 历史翻页只用 `before_seq`。
- 增量补拉只用 `after_seq`。
- 不能使用 `id`、`msg_id`、`send_time` 作为范围游标。
- `msg_id` 是客户端生成的幂等键，用于本地去重和发送态关联。

### 2.3 WebSocket

- 连接地址：`GET /api/v1/ws/`
- 认证方式：`Authorization: Bearer <token>` 或 `?token=<jwt>`
- 前端必须处理：
  - `message.sent`
  - `message.created`
  - `message.read`
  - `presence.changed`
  - `sync.required`
  - `error`
- 前端必须发送：
  - `message.delivered`
  - `message.read`
- 收到 `sync.required` 后必须尽快调用 `/api/v1/messages/sync` 补洞。

### 2.4 错误处理

- 以前端统一处理 `code` 为准，不要只依赖 HTTP status。
- 鉴权失效时应进入重新登录或刷新 token 流程。
- 冲突类错误需要给出稳定的用户提示，不要静默吞掉。

## 3. 当前接口分组

完整字段定义以 Swagger 为准：

- Swagger: [swagger.yaml](/home/zhuyin/im-system/backend/docs/swagger.yaml:1)
- WS 协议: [ws-protocol.md](/home/zhuyin/im-system/backend/docs/ws-protocol.md:1)

### 3.1 认证

- `POST /api/v1/auth/register`
  - 请求：`{ username, password }`
  - 响应：`{ user_id, username }`
- `POST /api/v1/auth/login`
  - 请求：`{ username, password }`
  - 响应：`{ access_token, refresh_token, expires_in }`
- `POST /api/v1/auth/refresh`
  - 请求：`{ refresh_token }`
  - 响应：`{ access_token, refresh_token, expires_in }`
- `POST /api/v1/auth/logout`
  - 需要 Bearer token

### 3.2 用户

- `GET /api/v1/users/me`
- `GET /api/v1/users/:id`
- `GET /api/v1/user/online`
- `POST /api/v1/users/avatar`
  - `multipart/form-data`
  - 字段：`file`
- `DELETE /api/v1/users/avatar`

### 3.3 好友与好友申请

- `GET /api/v1/friends`
- `DELETE /api/v1/friends/:id`
- `POST /api/v1/friend-requests`
  - 请求：`{ username, message }`
- `GET /api/v1/friend-requests/incoming`
- `GET /api/v1/friend-requests/outgoing`
- `POST /api/v1/friend-requests/:id/accept`
- `POST /api/v1/friend-requests/:id/reject`

这里的 `:id` 在好友申请 accept/reject 场景里都是申请单据 `id`；发送好友申请时目标用户通过 `username` 传递，不存在 `public_id`。

### 3.4 会话与群组

- `GET /api/v1/conversations`
- `GET /api/v1/conversations/groups`
- `POST /api/v1/conversations/:id/open`
- `POST /api/v1/conversations/:id/hide`
- `POST /api/v1/conversations/groups`
- `GET /api/v1/conversations/groups/:id`
- `GET /api/v1/conversations/groups/:id/members`
- `POST /api/v1/conversations/groups/:id/name`
- `POST /api/v1/conversations/groups/:id/invite`
- `POST /api/v1/conversations/groups/:id/members/:user_id/remove`
- `POST /api/v1/conversations/groups/:id/leave`
- `POST /api/v1/conversations/groups/:id/dismiss`

### 3.5 消息

- `GET /api/v1/messages/history`
  - 关键参数：`conversation_id`, `limit`, `before_seq`
- `GET /api/v1/messages/sync`
  - 关键参数：`conversation_id`, `limit`, `after_seq`

## 4. 建议的前端数据模型

以下是建议，不要求和后端 DTO 一模一样，但语义必须保持一致。

### 4.1 认证域

- `AuthSession`
  - `accessToken`
  - `refreshToken`
  - `expiresIn`
- `CurrentUser`
  - `userId`
  - `username`
  - `avatar`
  - `online`

### 4.2 会话域

- `Conversation`
  - `id`
  - `type`
  - `name`
  - `unreadCount`
  - `peer`
  - `lastMessage`

### 4.3 消息域

- `Message`
  - `id`
  - `msgId`
  - `conversationId`
  - `seq`
  - `type`
  - `event`
  - `fromUserId`
  - `sendTime`
  - `content`
  - `extra`
  - `localStatus`

其中 `localStatus` 是前端本地字段，建议取值：

- `sending`
- `sent`
- `failed`

## 5. 推荐实现流程

### 5.1 登录态

1. 登录成功后保存 `access_token`、`refresh_token`。
2. 进入应用后先请求 `/api/v1/users/me`。
3. token 过期优先走 `refresh`。
4. `refresh` 失败则清空本地登录态并回到登录页。

### 5.2 会话首页

1. 先拉 `/api/v1/conversations`
2. 并行拉 `/api/v1/friends`
3. 并行拉好友申请列表
4. 建立 WS 连接
5. 用 WS 事件增量更新本地状态

### 5.3 进入会话

1. 先调用 `/api/v1/conversations/:id/open`
2. 再调用 `/api/v1/messages/history`
3. 建立该会话的本地 `maxSeq`
4. 收到新消息后按 `seq` 追加或纠正顺序

### 5.4 发送消息

1. 客户端先生成 `msg_id`
2. 本地先插入 `sending` 状态消息
3. 通过 WS 发送 `message.send`
4. 收到 `message.sent` 后把本地消息改为 `sent`
5. 若超时或连接错误，可进入失败态并允许重试

## 6. 大模型生成前端时的禁止事项

1. 禁止使用 `public_id` 作为字段、变量名、接口参数或页面文案。
2. 禁止自己发明未定义接口。
3. 禁止把 `id`、`send_time` 当作消息游标。
4. 禁止忽略 `sync.required`。
5. 禁止在前端拼接头像物理路径，只使用服务端返回的 `avatar` URL。
6. 禁止假设好友、会话、群成员一定完整缓存，必须接受后端返回的真实数据覆盖本地状态。
7. 禁止把已送达和已读混为一个状态。

## 7. 推荐验收清单

前端至少应满足以下能力：

1. 用户可以注册、登录、刷新 token、登出。
2. 可以查看当前用户信息和指定用户资料。
3. 可以上传和清空头像。
4. 可以发送好友申请、接受/拒绝申请、查看好友列表。
5. 可以查看会话列表、打开会话、隐藏会话。
6. 可以创建群、查看群详情、查看群成员、邀请成员、移除成员、退群、解散群。
7. 可以收发单聊和群聊消息。
8. 可以正确发送 `message.delivered`，并正确处理 `message.read` 和 `presence.changed`。
9. 可以在断线或队列补洞场景下处理 `sync.required`。

## 8. 交付给大模型时建议附带的输入

除了本文档，建议一并提供：

1. [swagger.yaml](/home/zhuyin/im-system/backend/docs/swagger.yaml:1)
2. [ws-protocol.md](/home/zhuyin/im-system/backend/docs/ws-protocol.md:1)
3. 你希望采用的前端技术栈
4. 期望目录结构
5. UI 范围和优先级

如果只给大模型一句“根据后端写前端”，它很容易在协议边界上自由发挥。把以上材料一起提供，生成质量会稳定很多。
