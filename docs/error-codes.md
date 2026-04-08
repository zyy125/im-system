# Error Codes

项目统一使用“HTTP 状态码 + 业务错误码”的双层错误表达：

- HTTP 状态码：
  表示请求在传输层 / 协议层 / 访问控制层的大类结果，例如 `400`、`401`、`403`、`404`、`409`、`500`。
- 业务错误码：
  表示更具体的业务语义，例如 `auth.token_invalid`、`friend.not_friends`、`conversation.member_not_found`。

## 响应格式

```json
{
  "code": "friend.not_friends",
  "message": "not friends",
  "data": null
}
```

成功响应统一为：

```json
{
  "code": "ok",
  "message": "success",
  "data": {}
}
```

这意味着：

- `code`
  给前端程序、日志系统、监控系统做稳定判断。
- `message`
  给用户展示或开发调试查看。

前端**不要**依赖 `message` 做分支判断，应优先依赖 `code`。

## 命名规则

错误码统一采用：

`<domain>.<scenario>`

例如：

- `common.invalid_argument`
- `auth.invalid_credentials`
- `user.not_found`
- `friend_request.not_pending`
- `conversation.member_not_found`
- `message.invalid_payload`

## 新增错误码时的规则

新增错误码时遵守以下约束：

1. 先判断是否已有通用错误码可以复用。
2. 只有当错误具有明确业务语义时，才新增领域错误码。
3. 错误码保持稳定，避免前端和日志系统失效。
4. `message` 可以迭代优化，但 `code` 应尽量保持兼容。
5. 同一类错误在 HTTP、WebSocket、日志中应尽量使用同一个错误码。

## 前后端协作原则

前端收到错误响应后，建议按下面顺序处理：

1. 先判断 `code`
2. 再决定业务动作，例如：
   - 跳转登录页
   - 刷新列表
   - 停止重试
   - 提示用户补参数
3. 最后再用 `message` 做提示文案

推荐原则：

- `auth.*`
  以“登录态处理”为主。
- `common.invalid_*`
  以前端参数校验 / 表单提示为主。
- `friend.*`、`friend_request.*`
  以好友关系、申请状态刷新为主。
- `conversation.*`
  以刷新会话列表、重建当前会话状态为主。
- `message.*`
  以修正发送参数、刷新消息视图为主。

## 当前错误码清单

下面这张表建议前后端都遵守。`前端建议动作` 不是唯一方案，但应至少满足同类错误同类处理。

### 通用错误

| 错误码 | HTTP | 含义 | 前端建议动作 |
| --- | --- | --- | --- |
| `ok` | 200 | 请求成功 | 正常渲染数据 |
| `common.invalid_argument` | 400 | 请求参数不合法 | 提示用户修正输入，不自动重试 |
| `common.invalid_body` | 400 | 请求体格式错误 | 提示表单错误，不自动重试 |
| `common.unauthorized` | 401 | 未授权 | 清理登录态并跳转登录页 |
| `common.forbidden` | 403 | 无权限访问 | 弹出提示，不自动重试 |
| `common.not_found` | 404 | 资源不存在 | 提示资源不存在，可选择刷新页面 |
| `common.conflict` | 409 | 当前状态冲突 | 刷新对应列表或重新拉取状态 |
| `common.internal` | 500 | 服务内部错误 | 统一提示“服务异常”，记录日志 |
| `common.rate_limited` | 429 | 请求过于频繁 | 提示稍后再试，可做节流 |
| `common.timeout` | 504 | 请求超时 | 提示超时，可允许用户手动重试 |
| `common.service_unavailable` | 503 | 服务不可用 | 提示服务繁忙，可退避重试 |

### 认证相关

| 错误码 | HTTP | 含义 | 前端建议动作 |
| --- | --- | --- | --- |
| `auth.credentials_required` | 400 | 用户名或密码为空 | 在登录/注册表单上高亮必填项 |
| `auth.invalid_credentials` | 401 | 用户名或密码错误 | 停留在登录页，提示重新输入 |
| `auth.token_missing` | 401 | 缺少 token | 清理本地登录态并跳转登录页 |
| `auth.token_invalid` | 401 | token 无效 | 清理本地登录态并跳转登录页 |
| `auth.token_expired` | 401 | token 已过期 | 清理本地登录态并跳转登录页 |
| `auth.token_blacklisted` | 401 | token 已失效或已登出 | 清理本地登录态并跳转登录页 |

### 用户相关

| 错误码 | HTTP | 含义 | 前端建议动作 |
| --- | --- | --- | --- |
| `user.not_found` | 404 | 用户不存在 | 提示用户不存在，清空对应表单输入 |
| `user.already_exists` | 409 | 用户名已存在 | 注册页提示“用户名已被占用” |

### 好友相关

| 错误码 | HTTP | 含义 | 前端建议动作 |
| --- | --- | --- | --- |
| `friend.cannot_add_self` | 400 | 不能添加自己为好友 | 直接提示，不发起重试 |
| `friend.not_friends` | 403 | 双方不是好友，不能查看历史或聊天 | 禁止发送消息，提示先建立好友关系 |
| `friend.already_exists` | 409 | 好友关系已存在 | 刷新好友列表或直接提示“已经是好友” |

### 好友申请相关

| 错误码 | HTTP | 含义 | 前端建议动作 |
| --- | --- | --- | --- |
| `friend_request.already_pending` | 409 | 已存在待处理申请 | 刷新申请列表，提示不要重复发送 |
| `friend_request.already_friends` | 409 | 双方已经是好友 | 刷新好友列表和会话列表 |
| `friend_request.not_pending` | 409 | 申请已被处理，不能重复同意/拒绝 | 刷新申请列表 |
| `friend_request.no_permission` | 403 | 无权处理该申请 | 提示无权限，并刷新申请列表 |
| `friend_request.not_found` | 404 | 好友申请不存在 | 刷新申请列表 |

### 会话相关

| 错误码 | HTTP | 含义 | 前端建议动作 |
| --- | --- | --- | --- |
| `conversation.not_found` | 404 | 会话不存在 | 刷新会话列表，必要时关闭当前会话 |
| `conversation.invalid_single_key` | 500 | 单聊会话索引异常 | 提示服务异常，记录日志 |
| `conversation.member_not_found` | 404 | 当前用户不是该会话成员，或成员数据缺失 | 刷新会话列表，必要时关闭当前聊天窗口 |
| `conversation.member_update_failed` | 500 | 会话成员游标更新失败 | 提示操作失败，可稍后重试 |
| `conversation.not_accessible` | 403 | 当前用户不能访问该会话 | 提示无权限，并返回会话列表 |

### 消息相关

| 错误码 | HTTP | 含义 | 前端建议动作 |
| --- | --- | --- | --- |
| `message.invalid_peer_id` | 400 | `peer_id` 非法 | 修正参数，不自动重试 |
| `message.invalid_payload` | 400 | WebSocket 发送的消息体不合法 | 校验前端消息结构，停止自动发送 |
| `message.msg_id_required` | 400 | 消息缺少 `msg_id` | 重新生成消息 ID 后再发 |
| `message.conversation_required` | 400 | 缺少 `conversation_id` | 先确保会话存在，再发送消息 |
| `message.not_found` | 404 | 消息不存在 | 刷新消息列表或忽略当前操作 |

## WebSocket 错误处理建议

当前 WebSocket 错误帧格式与 HTTP 保持一致语义：

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

前端收到 WebSocket 错误后，建议按下面处理：

- `auth.token_invalid`
  主动关闭连接，清理登录态，回到登录页。
- `friend.not_friends`
  停止当前会话发送，提示用户“你们还不是好友”。
- `message.invalid_payload`
  说明前端发送消息结构有问题，应打印日志并阻止继续发送同类消息。
- `conversation.member_not_found`
  刷新会话列表和当前聊天窗口状态。
- `common.internal`
  提示“服务异常，请稍后再试”，不要无限重发。

## 推荐前端封装方式

前端建议封装一个统一错误处理函数，例如：

```js
function handleBusinessError(error) {
  const code = error?.response?.data?.code || error?.code;
  const message = error?.response?.data?.message || error?.message || '操作失败';

  switch (code) {
    case 'auth.token_missing':
    case 'auth.token_invalid':
    case 'auth.token_blacklisted':
    case 'auth.token_expired':
      logout();
      alert('登录已失效，请重新登录');
      return;

    case 'friend_request.not_pending':
      loadRequests();
      alert(message);
      return;

    case 'conversation.member_not_found':
      loadConversations();
      resetChatView();
      alert(message);
      return;

    default:
      alert(message);
  }
}
```

## 后续新增模块时的文档要求

以后如果新增如下模块，也要把错误码写进本文件：

- 群聊
- 文件上传
- 消息撤回
- 已读回执
- 通知 / 推送
- 管理后台

新增模块时至少补充四项信息：

1. 错误码
2. HTTP 状态码
3. 含义
4. 前端建议动作

## 当前目录

统一错误码定义位于：

- `internal/apperr/code.go`
- `internal/apperr/error.go`

HTTP 响应适配位于：

- `pkg/response/response.go`

错误日志透出位于：

- `internal/middleware/logging.go`
