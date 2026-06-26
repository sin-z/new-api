# new-api API Contract

## 文档说明

- 本文档是本仓 HTTP API 的 Markdown 契约源。
- 本次仅新增邮箱验证码登录相关接口，不改变现有密码登录、注册验证码、密码重置或 OAuth 回调接口。
- 当前仓已有 `docs/openapi/*.json` 参考产物，但未发现 `docs/api_contract.md -> api/openapi.yaml` 的统一生成入口；本次不新增 OpenAPI 生成产物。

## 公共约定

- Base path: `/api`
- 返回 envelope:

| Field | Type | Required | Description | Constraints |
| --- | --- | --- | --- | --- |
| success | boolean | yes | 业务是否成功 | `true` 或 `false` |
| message | string | yes | 用户可见消息 | 成功时为空字符串 |
| data | object/null | no | 响应数据 | 按接口定义 |

## 接口目录

| Method | Path | Description |
| --- | --- | --- |
| GET | `/api/user/email_login/code` | 向已存在用户邮箱发送登录验证码 |
| POST | `/api/user/email_login` | 使用邮箱验证码完成登录 |

## GET /api/user/email_login/code

### 描述

向已存在且允许登录的用户邮箱发送一次性登录验证码。该验证码用途与注册邮箱验证、密码重置隔离。

### 鉴权

无登录态要求。接口受关键操作限流保护；如 Turnstile 已启用，则按现有中间件校验 `turnstile` query。

### 请求参数

| Name | In | Type | Required | Description | Constraints |
| --- | --- | --- | --- | --- | --- |
| email | query | string | yes | 接收登录验证码的邮箱地址 | 必须为合法邮箱；必须属于现有用户 |
| turnstile | query | string | no | Turnstile token | 仅在站点启用 Turnstile 时校验 |

### 请求体字段

无。

### 成功响应字段

无，成功时 `data` 为空。

### 错误码表

| Condition | HTTP Status | success | message |
| --- | ---: | --- | --- |
| 邮箱格式非法 | 200 | false | 本地化参数错误文案 |
| 邮箱不存在 | 200 | false | 本地化邮箱登录用户不存在文案 |
| 邮箱对应用户被禁用 | 200 | false | 本地化用户被禁用文案 |
| SMTP 发送失败 | 200 | false | 发送失败错误信息 |
| Turnstile 校验失败 | 按现有中间件 | false | 按现有中间件 |

### 兼容性说明

- 不自动注册不存在邮箱。
- 不复用 `EmailVerificationPurpose`，避免注册验证码可直接登录。

## POST /api/user/email_login

### 描述

校验邮箱登录验证码，成功后写入与密码登录一致的 session，并返回登录用户基础信息。

### 鉴权

无登录态要求。接口受关键操作限流保护。

### 请求参数

无。

### 请求体字段

| Field | Type | Required | Description | Constraints |
| --- | --- | --- | --- | --- |
| email | string | yes | 用户邮箱地址 | 必须为合法邮箱；必须属于现有用户 |
| code | string | yes | 邮箱登录验证码 | 必须匹配未过期的邮箱登录验证码 |

### 成功响应字段

| Field | Type | Required | Description | Constraints |
| --- | --- | --- | --- | --- |
| id | integer | yes | 用户 ID | 与现有登录响应一致 |
| username | string | yes | 用户名 | 与现有登录响应一致 |
| display_name | string | yes | 展示名 | 与现有登录响应一致 |
| role | integer | yes | 用户角色 | 与现有登录响应一致 |
| status | integer | yes | 用户状态 | 与现有登录响应一致 |
| group | string | yes | 用户分组 | 与现有登录响应一致 |

### 错误码表

| Condition | HTTP Status | success | message |
| --- | ---: | --- | --- |
| 请求体非法 | 200 | false | 本地化参数错误文案 |
| 邮箱不存在 | 200 | false | 本地化邮箱登录用户不存在文案 |
| 验证码错误或过期 | 200 | false | 本地化验证码错误文案 |
| 用户被禁用 | 200 | false | 本地化用户被禁用文案 |
| session 保存失败 | 200 | false | 本地化 session 保存失败文案 |

### 兼容性说明

- 成功响应复用现有 `setupLogin` 输出结构。
- 验证成功后删除本邮箱登录验证码，避免同一验证码重复使用。

## 依赖与约束

- 依赖现有 SMTP 发送配置。
- 依赖现有进程内验证码存储，验证码在多实例之间不共享；该边界与现有注册验证码、密码重置验证码一致。

## 待确认项

无。
