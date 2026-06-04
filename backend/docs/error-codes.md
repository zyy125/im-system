# Error Codes

## 1. 文档目的

本文档定义项目使用的业务错误码、响应格式和前后端协作约定。

## 2. 响应格式

错误响应格式：

```json
{
  "code": "friend.not_friends",
  "message": "not friends",
  "data": null
}
```

成功响应格式：

```json
{
  "code": "ok",
  "message": "success",
  "data": {}
}
```

字段含义：

- `code`
  用于前端、日志和监控的稳定判断。
- `message`
  用于展示或调试。

前端分支逻辑应优先依赖 `code`，而不是 `message`。

## 3. 命名规则

错误码格式为：

- `<domain>.<scenario>`

示例：

- `common.invalid_argument`
- `auth.invalid_credentials`
- `user.not_found`
- `friend_request.not_pending`
- `conversation.member_not_found`
- `message.invalid_payload`

## 4. 错误码使用约束

新增错误码时遵守以下约束：

1. 优先复用已有通用错误码
2. 仅在错误具有明确业务语义时新增领域错误码
3. `code` 保持稳定，避免前端和日志系统失效
4. `message` 可迭代优化，但应避免影响 `code`
5. 同一类错误在 HTTP、WebSocket、日志中保持一致命名

## 5. 前后端协作约定

前端处理错误响应时，按以下顺序处理：

1. 判断 `code`
2. 决定业务动作
3. 使用 `message` 生成提示文案

不同错误域的建议处理方式如下：

- `auth.*`
  登录态处理
- `common.invalid_*`
  参数校验或表单提示
- `friend.*`、`friend_request.*`
  好友关系与申请状态刷新
- `conversation.*`
  会话列表或当前会话状态刷新
- `message.*`
  发送参数修正或消息视图刷新

## 6. 错误码清单

### 6.1 通用错误

| 错误码 | HTTP | 含义 | 建议动作 |
| --- | --- | --- | --- |
| `ok` | 200 | 请求成功 | 正常渲染数据 |
| `common.invalid_argument` | 400 | 请求参数不合法 | 提示用户修正输入，不自动重试 |
| `common.invalid_body` | 400 | 请求体格式错误 | 提示表单错误，不自动重试 |
| `common.unauthorized` | 401 | 未授权 | 清理登录态并跳转登录页 |
| `common.forbidden` | 403 | 无权限访问 | 提示无权限，不自动重试 |
| `common.not_found` | 404 | 资源不存在 | 提示资源不存在，必要时刷新页面 |
| `common.conflict` | 409 | 当前状态冲突 | 刷新对应列表或重新拉取状态 |
| `common.internal` | 500 | 服务内部错误 | 统一提示服务异常并记录日志 |
| `common.rate_limited` | 429 | 请求过于频繁 | 提示稍后再试，可做节流 |
| `common.timeout` | 504 | 请求超时 | 提示超时，可允许手动重试 |
| `common.service_unavailable` | 503 | 服务不可用 | 提示服务繁忙，可退避重试 |

### 6.2 认证相关

| 错误码 | HTTP | 含义 | 建议动作 |
| --- | --- | --- | --- |
| `auth.credentials_required` | 400 | 账号或密码为空 | 高亮必填项 |
| `auth.invalid_credentials` | 401 | 账号或密码错误 | 停留在登录页并提示重新输入 |
| `auth.token_missing` | 401 | 缺少 token | 清理登录态并跳转登录页 |
| `auth.token_invalid` | 401 | token 无效 | 清理登录态并跳转登录页 |
| `auth.token_expired` | 401 | token 已过期 | 清理登录态并跳转登录页 |
| `auth.token_blacklisted` | 401 | token 已失效或已登出 | 清理登录态并跳转登录页 |

### 6.3 用户相关

| 错误码 | HTTP | 含义 | 建议动作 |
| --- | --- | --- | --- |
| `user.not_found` | 404 | 用户不存在 | 提示用户不存在并清空对应输入 |

### 6.4 好友相关

| 错误码 | HTTP | 含义 | 建议动作 |
| --- | --- | --- | --- |
| `friend.cannot_add_self` | 400 | 不能添加自己为好友 | 直接提示，不自动重试 |
| `friend.not_friends` | 403 | 双方不是好友 | 禁止发送消息并提示先建立好友关系 |
| `friend.already_exists` | 409 | 好友关系已存在 | 刷新好友列表或直接提示 |

### 6.5 好友申请相关

| 错误码 | HTTP | 含义 | 建议动作 |
| --- | --- | --- | --- |
| `friend_request.already_pending` | 409 | 已存在待处理申请 | 刷新申请列表并提示不要重复发送 |
| `friend_request.already_friends` | 409 | 双方已经是好友 | 刷新好友列表和会话列表 |
| `friend_request.not_pending` | 409 | 申请已被处理 | 刷新申请列表 |
| `friend_request.no_permission` | 403 | 无权处理该申请 | 提示无权限并刷新申请列表 |
| `friend_request.not_found` | 404 | 好友申请不存在 | 刷新申请列表 |

### 6.6 会话相关

| 错误码 | HTTP | 含义 | 建议动作 |
| --- | --- | --- | --- |
| `conversation.not_found` | 404 | 会话不存在 | 刷新会话列表，必要时关闭当前会话 |
| `conversation.invalid_single_key` | 500 | 单聊会话索引异常 | 提示服务异常并记录日志 |
| `conversation.member_not_found` | 404 | 当前用户不是该会话成员，或成员数据缺失 | 刷新会话列表，必要时关闭当前聊天窗口 |
| `conversation.member_update_failed` | 500 | 会话成员游标更新失败 | 提示操作失败，可稍后重试 |
| `conversation.not_accessible` | 403 | 当前用户不能访问该会话 | 提示无权限并返回会话列表 |

### 6.7 消息相关

| 错误码 | HTTP | 含义 | 建议动作 |
| --- | --- | --- | --- |
| `message.invalid_peer_id` | 400 | `peer_id` 非法 | 修正参数，不自动重试 |
| `message.invalid_payload` | 400 | WebSocket 消息体不合法 | 校验消息结构并停止自动发送 |
| `message.msg_id_required` | 400 | 缺少 `msg_id` | 重新生成消息 ID 后发送 |
| `message.conversation_required` | 400 | 缺少 `conversation_id` | 先确保会话存在，再发送消息 |
| `message.not_found` | 404 | 消息不存在 | 刷新消息列表或忽略当前操作 |

## 7. WebSocket 错误帧

WebSocket 错误帧沿用统一 envelope，格式如下：

```json
{
  "type": "error",
  "data": {
    "code": "message.invalid_payload",
    "message": "invalid message"
  }
}
```

约束如下：

- 错误帧仅包含 `type` 和 `data`
- 不携带 `version` 字段
- `data` 中使用与 HTTP 一致的 `{code, message}` 语义

## 8. 扩展要求

后续扩展以下模块时，应补充对应错误码：

- 文件上传
- 消息撤回
- 已读增强能力
- 通知 / 推送
- 管理后台

新增模块时至少补充以下信息：

1. 错误码
2. HTTP 状态码
3. 含义
4. 建议动作

## 9. 相关实现

统一错误码定义位于：

- `internal/apperr/code.go`
- `internal/apperr/error.go`

HTTP 响应适配位于：

- `pkg/response/response.go`

错误日志透出位于：

- `internal/middleware/logging.go`
