# new-api API Contract

## 文档说明

- 本文档是本仓 HTTP API 的 Markdown 契约源。
- 本文档当前包含邮箱验证码登录接口与 Seedance 2.0 BytePlus / ModelArk native 接口契约。
- 邮箱验证码登录接口不改变现有密码登录、注册验证码、密码重置或 OAuth 回调接口。
- Seedance 2.0 native 接口为技术方案阶段 contract-first 产物；不改变既有 `/v1/video/generations`、`/v1/videos`、`/kling/v1` 或 `jimeng` 对外契约。
- 当前仓已有 `docs/openapi/*.json` 参考产物，但未发现 `docs/api_contract.md -> api/openapi.yaml` 的统一生成入口；本次不新增 OpenAPI 生成产物。

## 公共约定

以下约定适用于 `/api` dashboard / 用户接口。Seedance 2.0 native 接口的响应外壳以本文档后续专章为准，不使用 `{success,message,data}` envelope。

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
| GET | `/api/user/email_login/code` | 向邮箱发送登录 / 注册验证码 |
| POST | `/api/user/email_login` | 使用邮箱验证码完成登录或自动创建账号后登录 |

## GET /api/user/email_login/code

### 描述

向邮箱发送一次性登录验证码。邮箱已绑定启用用户时用于登录；邮箱未注册时，在系统 `RegisterEnabled=true` 且邮箱未被现有或软删除账号占用的前提下，可用于后续自动创建账号并登录。该验证码用途与注册邮箱验证、密码重置隔离。

### 鉴权

无登录态要求。接口受关键操作限流保护；如 Turnstile 已启用，则按现有中间件校验 `turnstile` query。

### 请求参数

| Name | In | Type | Required | Description | Constraints |
| --- | --- | --- | --- | --- | --- |
| email | query | string | yes | 接收登录 / 注册验证码的邮箱地址 | 必须为合法邮箱；最长 254 字符；已注册邮箱必须属于启用用户；未注册邮箱必须允许注册且未被现有或软删除账号占用 |
| turnstile | query | string | no | Turnstile token | 仅在站点启用 Turnstile 时校验 |

### 请求体字段

无。

### 成功响应字段

无，成功时 `data` 为空。

### 错误码表

| Condition | HTTP Status | success | message |
| --- | ---: | --- | --- |
| 邮箱格式非法 | 200 | false | 本地化参数错误文案 |
| 注册关闭且邮箱未注册 | 200 | false | 本地化注册关闭文案 |
| 邮箱或同名 username 已被现有 / 软删除账号占用 | 200 | false | 本地化用户已存在文案 |
| 邮箱对应用户被禁用 | 200 | false | 本地化用户被禁用文案 |
| SMTP 发送失败 | 200 | false | 发送失败错误信息 |
| Turnstile 校验失败 | 按现有中间件 | false | 按现有中间件 |

### 兼容性说明

- 发送验证码阶段不创建用户；未注册邮箱只在验证码登录成功后创建用户。
- 未注册邮箱自动创建账号受 `RegisterEnabled` 控制，不受 `PasswordRegisterEnabled` 控制。
- 自动创建账号使用完整邮箱作为 `username`、`email`、`display_name`，角色为普通用户，状态为启用。
- 不复用 `EmailVerificationPurpose`，避免注册验证码可直接登录。

## POST /api/user/email_login

### 描述

校验邮箱登录验证码。邮箱已绑定启用用户时直接登录；邮箱未注册时，在系统 `RegisterEnabled=true` 且邮箱未被现有或软删除账号占用的前提下，自动创建普通启用用户后写入与密码登录一致的 session，并返回登录用户基础信息。

### 鉴权

无登录态要求。接口受关键操作限流保护。

### 请求参数

无。

### 请求体字段

| Field | Type | Required | Description | Constraints |
| --- | --- | --- | --- | --- |
| email | string | yes | 用户邮箱地址 | 必须为合法邮箱；最长 254 字符；已注册邮箱必须属于启用用户；未注册邮箱必须允许注册且未被现有或软删除账号占用 |
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
| 验证码错误或过期 | 200 | false | 本地化验证码错误文案 |
| 用户被禁用 | 200 | false | 本地化用户被禁用文案 |
| 注册关闭且邮箱未注册 | 200 | false | 本地化注册关闭文案 |
| 邮箱或同名 username 已被现有 / 软删除账号占用 | 200 | false | 本地化用户已存在文案 |
| 自动创建用户失败 | 200 | false | 创建失败错误信息 |
| session 保存失败 | 200 | false | 本地化 session 保存失败文案 |

### 兼容性说明

- 成功响应复用现有 `setupLogin` 输出结构。
- 验证成功后删除本邮箱登录验证码，避免同一验证码重复使用。
- 已存在启用用户不受 `RegisterEnabled` 影响，仍可使用邮箱验证码登录。
- 自动创建账号使用随机内部密码；该密码不通过接口返回，用户后续仍通过邮箱验证码、OAuth 或其他可用登录方式登录。

## 依赖与约束

- 依赖现有 SMTP 发送配置。
- 依赖现有进程内验证码存储，验证码在多实例之间不共享；该边界与现有注册验证码、密码重置验证码一致。

## 待确认项

无。

---

# Seedance 2.0 BytePlus / ModelArk Native API Contract

## 文档说明

- 本节是 Seedance 2.0 BytePlus / ModelArk native 端点直替契约源。
- 本节只定义 P0 create / get / list；`DELETE` / cancel 不在本阶段支持范围内；不改变既有 `/v1/video/generations`、`/v1/videos`、`/kling/v1` 或 `jimeng` 对外契约。
- 本节为技术方案阶段 contract-first 产物；Coding 阶段必须直接消费本节。若实现时发现字段、状态、错误或 OpenAPI 适用性需变更，必须退回技术方案阶段更新并 Review。
- 当前仓已有 `docs/openapi/*.json` 参考产物，但未发现 `docs/api_contract.md -> api/openapi.yaml` 的统一生成入口，且未发现正式 `api/openapi.yaml` 消费证据；本阶段不生成 `api/openapi.yaml`。

## 公共约定

### Base URL 与协议定位

- Base path: `/api/v3/contents/generations`
- 协议目标：调用方使用 ZZ123 服务域名替换 BytePlus / ModelArk 域名，path、字段、状态和错误语义按 BytePlus native 心智对齐。
- 响应不得使用 OpenAI Video wrapper，不得使用 `/api` dashboard `{success,message,data}` envelope。

### 鉴权

| Name | In | Type | Required | Description | Constraints |
| --- | --- | --- | --- | --- | --- |
| Authorization | header | string | yes | API Key Bearer token | 格式为 `Bearer <api_key>`；复用现有 API Key `TokenAuth()` 信任链 |

鉴权规则：

- create 使用 API Key 鉴权和渠道分发。
- get / list 只允许访问当前 API Key 所属用户的 public task。
- 非本人 task、非 Seedance native 可渲染 task 均返回 404，不泄漏存在性。
- 外部 request path 中的 `{id}` 只接受 public task id，不接受 BytePlus 上游真实 task id。

### Task ID 与上游 ID 隔离

| 标识 | 对外可见 | 存储位置 | 用途 |
| --- | --- | --- | --- |
| public task id | yes | `model.Task.TaskID` | create response、get/list path、OpenAI/native 跨协议互查 |
| upstream task id | no | `model.TaskPrivateData.UpstreamTaskID` | 与 BytePlus / ModelArk 上游通信 |

任何 native response、OpenAI wrapper、callback、日志和错误消息都不得暴露 upstream task id。

### 状态枚举

| Native status | Meaning | Implementation boundary |
| --- | --- | --- |
| `queued` | 任务排队中 | 由现有内部 `SUBMITTED` / `QUEUED` 渲染，不新增内部状态 |
| `running` | 任务执行中 | 由现有内部 `IN_PROGRESS` 渲染，不新增内部状态 |
| `cancelled` | 已取消 | P0 不主动产生；仅作为官方协议值域 / C1 实测核对项，不新增内部 `CANCELLED` |
| `succeeded` | 成功 | 由现有内部 `SUCCESS` 渲染 |
| `failed` | 失败 | 由现有内部 `FAILURE` 渲染 |
| `expired` | 过期 | P0 不主动产生；仅作为官方响应值域 / C1 实测核对项，不新增内部 `EXPIRED` |

约束：本阶段不新增 `model.TaskStatus` 枚举；不得新增内部 `CANCELLED` / `EXPIRED` / `DELETED`。`DELETED` 不是官方 response status，本阶段不实现删除后不可查语义。

### canonical `Task.Data` schema

实现必须将 Seedance task 的 `Task.Data` 维护为两个 builder 均可读取的 canonical schema：

```json
{
  "id": "task_public_id",
  "model": "seedance-2-0-pro",
  "status": "queued",
  "created_at": 1710000000,
  "updated_at": 1710000100,
  "content": {
    "video_url": "",
    "last_frame_url": ""
  },
  "error": {
    "code": "",
    "message": ""
  },
  "request": {
    "resolution": "1080p",
    "ratio": "16:9",
    "duration": 5,
    "generate_audio": true,
    "watermark": false,
    "return_last_frame": false
  },
  "usage": {
    "completion_tokens": 0,
    "total_tokens": 0
  },
  "seed": 0,
  "priority": 0,
  "service_tier": "default",
  "execution_expires_after": 172800
}
```

约束：

- `id` 必须是 public task id。
- upstream task id 只能进入 `TaskPrivateData.UpstreamTaskID`。
- native builder 读取 `id` / `model` / `status` / `content.video_url` / `error`。
- OpenAI video builder 读取同一份 schema 生成 `id` / `task_id` / `model` / `status` / `url` / `error`。

### OpenAI / native 跨协议互查

同一用户 / API Key 权限域内，Seedance task 支持以下四种组合：

| Create entry | Get entry | Response shell |
| --- | --- | --- |
| OpenAI `/v1/videos` | OpenAI `/v1/videos/{id}` | OpenAI Video wrapper |
| OpenAI `/v1/videos` | Seedance native `/api/v3/contents/generations/tasks/{id}` | BytePlus native task object |
| Seedance native `/api/v3/contents/generations/tasks` | OpenAI `/v1/videos/{id}` | OpenAI Video wrapper |
| Seedance native `/api/v3/contents/generations/tasks` | Seedance native `/api/v3/contents/generations/tasks/{id}` | BytePlus native task object |

跨协议互查要求：

- 四组合均使用同一 public task id。
- 查询入口决定 response / error 外壳。
- native get 只渲染 Seedance native 可渲染任务；未接入 BytePlus native task adaptor 能力的 channel type 返回 native 404。当前代码事实为 DoubaoVideo / VolcEngine 复用 `doubao` task adaptor；后续 xrtoken 或其他兼容 channel type 必须先完成 task adaptor / native contract / C1 验证后再纳入。
- 四组合测试必须断言 `id` / `model` / `status` / `url` 或 `content.video_url` / `error` 字段完整且状态语义一致。

### 错误对象

错误响应使用 native error shell：

```json
{
  "error": {
    "code": "InvalidParameter.Unsupported",
    "message": "service_tier is not configurable for Seedance 2.0",
    "type": "BadRequest"
  }
}
```

通用错误码：

| Condition | HTTP Status | error.code | error.type |
| --- | ---: | --- | --- |
| 请求 JSON 非法 | 400 | `InvalidParameter.InvalidJSON` | `BadRequest` |
| 必填字段缺失或值域非法 | 400 | `InvalidParameter.InvalidValue` | `BadRequest` |
| Seedance 2.0 不支持该参数 | 400 | `InvalidParameter.Unsupported` | `BadRequest` |
| `callback_url` 非空 | 400 | `OperationDenied.CallbackNotSupported` | `BadRequest` |
| 模型或 endpoint 不存在 / 不可用 | 404 | `InvalidEndpointOrModel.NotFound` | `NotFound` |
| task 不存在、非本人或非 Seedance task | 404 | `ResourceNotFound.Task` | `NotFound` |
| 服务未开通或权限不足 | 403 | `OperationDenied.ServiceNotOpen` | `Forbidden` |
| 账户限流 | 429 | `AccountRateLimitExceeded` | `TooManyRequests` |
| 内部服务错误 | 500 | `InternalServiceError` | `InternalServerError` |

C1 门禁：API 锁定前必须用官方示例或测试渠道真实响应逐条复核字段、状态、错误 shell、错误 code、HTTP status、type和值域。若实测与本节冲突，必须先更新 contract 并 Review。

### Seedance 2.0 字段限制

- `service_tier`：Seedance 2.0 only supports online inference mode and does not support configuring `service_tier`。本契约只允许缺省或 `default`；任何非 `default`，包括 `flex`，返回 400。
- `callback_url`：P0 禁用非空值，返回 400。后续如启用，必须先完成 SSRF 防护、白名单、签名、重试、脱敏和安全 Review。
- `draft` / `draft_task_id`：C2 待补证。未通过官方文档或测试渠道证实前，不作为 Seedance 2.0 get / list 必返字段。
- 价格、可售性、SLA、账户折扣：不在本 API response 中承诺。对外展示必须来自 `new-api` 实配、测试渠道实测或 owner 确认。

## 接口目录

| Method | Path | Description |
| --- | --- | --- |
| POST | `/api/v3/contents/generations/tasks` | 创建 Seedance 2.0 video generation task |
| GET | `/api/v3/contents/generations/tasks/{id}` | 查询单个 Seedance task |
| GET | `/api/v3/contents/generations/tasks` | 列出当前用户最近 7 天 Seedance task |

## POST /api/v3/contents/generations/tasks

### 描述

创建 Seedance 2.0 video generation task。成功后返回 public task id，任务异步执行。

### 鉴权

API Key `Authorization: Bearer <api_key>`。接口复用渠道分发与计费链路。

### 请求参数

除 `Authorization` header 外无 query / path 参数。

### 请求体字段

| Field | Type | Required | Description | Constraints |
| --- | --- | --- | --- | --- |
| model | string | yes | Seedance 2.0 对外模型 id 或 endpoint id | 必须通过常规 `Distribute()` 在当前 API Key / user group 下选中可用 Seedance channel；选中 channel type 必须已接入 BytePlus native task adaptor 能力 |
| content | object[] | yes | 输入内容数组 | 至少包含一个 text item；支持的 item 见下表 |
| content[].type | string | yes | 内容类型 | `text` / `image_url` / `video_url` / `audio_url` / sample task id 类型；C1 前需实测值域 |
| content[].text | string | conditional | 文本 prompt | `type=text` 时必填 |
| content[].image_url.url | string | conditional | 图片 URL | `type=image_url` 时必填；必须为公网可访问 URL |
| content[].video_url.url | string | conditional | 视频 URL | `type=video_url` 时必填；必须为公网可访问 URL |
| content[].audio_url.url | string | conditional | 音频 URL | `type=audio_url` 时必填；必须为公网可访问 URL |
| callback_url | string | no | 状态变更回调 URL | P0 禁用非空值；非空返回 `OperationDenied.CallbackNotSupported` |
| return_last_frame | boolean | no | 是否返回 last frame | 默认 `false` |
| service_tier | string | no | 服务层级 | Seedance 2.0 不可配置；缺省或 `default` 允许；其他值返回 400 |
| execution_expires_after | integer | no | 任务过期秒数 | 默认 172800；范围 `[3600,259200]` |
| generate_audio | boolean | no | 是否生成音频 | 默认按官方值；C1 前需实测默认 |
| priority | integer | no | 优先级 | Seedance 2.0 支持；范围 `0-9` |
| resolution | string | no | 分辨率 | `480p` / `720p` / `1080p` / `4k`；模型限制按官方实测 |
| ratio | string | no | 画幅比例 | `16:9` / `4:3` / `1:1` / `3:4` / `9:16` / `21:9` / `adaptive` |
| duration | integer | no | 视频时长秒数 | Seedance 2.0 series 为 `[4,15]` 或 `-1` |
| frames | integer | no | 帧数 | Seedance 2.0 不支持；传入返回 400 |
| seed | integer | no | 随机种子 | Seedance 2.0 不支持；传入返回 400 |
| camera_fixed | boolean | no | 固定相机 | Seedance 2.0 当前不支持；传入返回 400 |
| watermark | boolean | no | 是否加水印 | `true` / `false` |

### 成功响应

HTTP 200:

```json
{
  "id": "task_abcdefghijklmnopqrstuvwxyz123456"
}
```

| Field | Type | Required | Description | Constraints |
| --- | --- | --- | --- | --- |
| id | string | yes | public task id | 必须是 `model.Task.TaskID`；不得是上游真实 id |

### 错误码表

| Condition | HTTP Status | error.code | error.type |
| --- | ---: | --- | --- |
| JSON 非法 | 400 | `InvalidParameter.InvalidJSON` | `BadRequest` |
| `model` 缺失 | 400 | `InvalidParameter.InvalidValue` | `BadRequest` |
| `content` 缺失或为空 | 400 | `InvalidParameter.InvalidValue` | `BadRequest` |
| `service_tier` 为非 `default` | 400 | `InvalidParameter.Unsupported` | `BadRequest` |
| Seedance 2.0 不支持 `frames` / `seed` / `camera_fixed` | 400 | `InvalidParameter.Unsupported` | `BadRequest` |
| `callback_url` 非空 | 400 | `OperationDenied.CallbackNotSupported` | `BadRequest` |
| 模型不可用 | 404 | `InvalidEndpointOrModel.NotFound` | `NotFound` |
| 服务未开通 | 403 | `OperationDenied.ServiceNotOpen` | `Forbidden` |
| 限流 | 429 | `AccountRateLimitExceeded` | `TooManyRequests` |
| 上游或内部错误 | 500 | `InternalServiceError` | `InternalServerError` |

### 兼容性说明

- 不复用 OpenAI Video create response。
- `id` 在 OpenAI video 和 Seedance native 两套查询入口中共用。
- C2 未补证前，create response 不返回 `draft_task_id`。

## GET /api/v3/contents/generations/tasks/{id}

### 描述

查询当前用户名下单个 Seedance task 的 native 状态和结果。

### 鉴权

API Key `Authorization: Bearer <api_key>`。

### 请求参数

| Name | In | Type | Required | Description | Constraints |
| --- | --- | --- | --- | --- | --- |
| id | path | string | yes | public task id | 只接受 public id；不接受 upstream id |

### 请求体字段

无。

### 成功响应字段

| Field | Type | Required | Description | Constraints |
| --- | --- | --- | --- | --- |
| id | string | yes | public task id | 与 path id 一致 |
| model | string | yes | 对外模型 id | 原始请求模型 |
| status | string | yes | task 状态 | `queued` / `running` / `cancelled` / `succeeded` / `failed` / `expired` |
| error | object/null | yes | 失败信息 | 成功或未完成时为 `null`；失败时包含 `code`、`message` |
| error.code | string | conditional | 错误 code | `status=failed` 时返回 |
| error.message | string | conditional | 错误说明 | `status=failed` 时返回 |
| created_at | integer | yes | 创建时间戳 | Unix seconds |
| updated_at | integer | yes | 更新时间戳 | Unix seconds |
| content | object | yes | 结果内容 | 未成功时 `content.video_url` 可为空 |
| content.video_url | string | conditional | 视频 URL | `status=succeeded` 时必填；有效期按上游 |
| content.last_frame_url | string | no | 最后一帧 URL | 仅 `return_last_frame=true` 且上游返回时出现 |
| seed | integer | no | seed | Seedance 2.0 未支持时可省略 |
| resolution | string | no | 分辨率 | 来自请求快照或上游 |
| ratio | string | no | 画幅比例 | 来自请求快照或上游 |
| duration | integer | no | 视频时长 | 来自请求快照或上游 |
| framespersecond | integer | no | 视频帧率 | 上游返回时出现 |
| generate_audio | boolean | no | 是否生成音频 | 来自请求快照或上游 |
| priority | integer | no | 优先级 | 来自请求快照或上游；为 `0` 时按零值省略 |
| service_tier | string | no | 服务层级 | 若返回只能为 `default` |
| execution_expires_after | integer | no | 任务过期秒数 | 来自请求快照或上游 |
| usage.completion_tokens | integer | no | 输出 token 数 | 成功或上游返回 usage 时出现 |
| usage.total_tokens | integer | no | 总 token 数 | 成功或上游返回 usage 时出现 |
| draft | boolean | no | draft 标记 | 上游返回且为 `true` 时出现；`false` 按零值省略 |
| draft_task_id | string | no | draft task id | C2 未补证前不作为 Seedance 2.0 必返 |

### 示例响应

```json
{
  "id": "task_abcdefghijklmnopqrstuvwxyz123456",
  "model": "seedance-2-0-pro",
  "status": "succeeded",
  "error": null,
  "created_at": 1710000000,
  "updated_at": 1710000100,
  "content": {
    "video_url": "https://example.com/video.mp4",
    "last_frame_url": "https://example.com/last-frame.png"
  },
  "seed": 78674,
  "resolution": "720p",
  "ratio": "16:9",
  "duration": 5,
  "framespersecond": 24,
  "generate_audio": true,
  "service_tier": "default",
  "execution_expires_after": 172800,
  "usage": {
    "completion_tokens": 108900,
    "total_tokens": 108900
  }
}
```

### 错误码表

| Condition | HTTP Status | error.code | error.type |
| --- | ---: | --- | --- |
| task 不存在、非本人 | 404 | `ResourceNotFound.Task` | `NotFound` |
| task 所属 channel type 未接入 Seedance native task adaptor 能力 | 404 | `ResourceNotFound.Task` | `NotFound` |
| 鉴权失败或服务未开通 | 403 | `OperationDenied.ServiceNotOpen` | `Forbidden` |
| 内部错误 | 500 | `InternalServiceError` | `InternalServerError` |

### 兼容性说明

- OpenAI 创建的 Seedance task 可通过本接口查询，返回 native task object。
- native 创建的 Seedance task 可通过 OpenAI `/v1/videos/{id}` 查询，返回 OpenAI Video wrapper。
- 查询入口决定 response / error shell。

## GET /api/v3/contents/generations/tasks

### 描述

列出当前用户最近 7 天内 Seedance native 可渲染任务。

### 鉴权

API Key `Authorization: Bearer <api_key>`。

### 请求参数

| Name | In | Type | Required | Description | Constraints |
| --- | --- | --- | --- | --- | --- |
| page_num | query | integer | no | 页码 | 默认 1；范围 `[1,500]` |
| page_size | query | integer | no | 每页数量 | 默认 10；范围 `[1,500]` |
| filter.status | query | string | no | 状态过滤 | `queued` / `running` / `cancelled` / `succeeded` / `failed` |
| filter.task_ids | query | string[] | no | public task id 过滤 | 支持多个同名 query；只接受 public id |
| filter.model | query | string | no | 模型过滤 | 对外模型 id 或 endpoint id |
| filter.service_tier | query | string | no | 服务层级过滤 | Seedance 2.0 只允许缺省或 `default`；`flex` 返回 400 |

### 请求体字段

无。

说明：响应 `items[].status` 可包含 `expired`，但 P0 不支持按该状态过滤。

### 成功响应字段

| Field | Type | Required | Description | Constraints |
| --- | --- | --- | --- | --- |
| items | object[] | yes | task 列表 | 每项结构同 get 响应 |
| total | integer | yes | 符合条件的总数 | 按当前用户、最近 7 天、过滤条件统计 |

### 示例响应

```json
{
  "items": [
    {
      "id": "task_abcdefghijklmnopqrstuvwxyz123456",
      "model": "seedance-2-0-pro",
      "status": "running",
      "error": null,
      "created_at": 1710000000,
      "updated_at": 1710000100,
      "content": {
        "video_url": ""
      },
      "service_tier": "default"
    }
  ],
  "total": 1
}
```

### 错误码表

| Condition | HTTP Status | error.code | error.type |
| --- | ---: | --- | --- |
| 分页参数超范围 | 400 | `InvalidParameter.InvalidValue` | `BadRequest` |
| `filter.status` 非法 | 400 | `InvalidParameter.InvalidValue` | `BadRequest` |
| `filter.service_tier` 非 `default` | 400 | `InvalidParameter.Unsupported` | `BadRequest` |
| 鉴权失败或服务未开通 | 403 | `OperationDenied.ServiceNotOpen` | `Forbidden` |
| 内部错误 | 500 | `InternalServiceError` | `InternalServerError` |

### 兼容性说明

- list 只返回 Seedance native 可渲染任务；未接入 BytePlus native task adaptor 能力的 channel type 不进入 native list 结果。
- list 不返回其他用户任务，不暴露 channel id、upstream task id、provider key 或内部计费上下文。
- Custom API key 下 `filter.model` 是否必填必须在 C1 / owner 验证中确认；未确认前不作为 ZZ123 P0 必填。

## 依赖与约束

- 依赖 `model.Task.TaskID` public id 与 `TaskPrivateData.UpstreamTaskID` 私有 id 隔离。
- 依赖 `Task.Data` canonical schema 支撑 native builder 和 OpenAI builder。
- 依赖 `TokenAuth()` 提供 user 隔离。
- 依赖后续 Coding 阶段补四组合接口测试。
- native create 必须复用现有 task submit 的渠道选择、预扣、上游 request 构建 / 发送、重试、提交后计费调整和落库语义。
- 当前 task adaptor 可能存在上游响应解析与 OpenAI Video wrapper HTTP 写出耦合；Coding 只允许在不改变既有 OpenAI video 行为的前提下解耦“解析结果”和“HTTP response 渲染”，供 native create 返回 `{"id":"task_xxx"}`。
- 禁止为本 native contract 修改 `PreConsumeBilling`、`SettleBilling`、`RefundTaskQuota`、`RecalculateTaskQuota`、`RecalculateTaskQuotaByTokens`、轮询终态成功差额结算 / 失败退款状态机。
- 禁止新增 cancel / delete / cancelled refund 分支；DELETE/cancel 已移出 S1.2。
- 如果实现时发现必须改变计费、预扣、轮询、结算或提交业务流程语义，必须退回技术方案 / contract-first 重新 Review。
- C1 未完成前，本节字段、错误 shell、HTTP status 和值域不得视为最终 API 锁定。
- C2 未完成前，`draft` / `draft_task_id` 不作为 Seedance 2.0 必返字段。
- 本阶段不支持 `DELETE /api/v3/contents/generations/tasks/{id}`，不实现 cancel / delete，不新增内部 `CANCELLED` / `EXPIRED` / `DELETED` 状态；后续如恢复 DELETE，必须先回到 contract-first 阶段补充方案、协议、计费和幂等设计。
- 价格、可售性、SLA、账户折扣不得从本 contract 推导；必须另有真实来源。

## 待确认项

| ID | Item | Current handling | Blocks |
| --- | --- | --- | --- |
| C1 | 官方字段、状态、错误 shell、HTTP status、type、值域逐条实测 | API 锁定前强制执行 | Coding Review / API Testing |
| C2 | `draft` / `draft_task_id` 是否适用于 Seedance 2.0 | 未证实前 optional 且不必返 | 字段必返口径 |
| D1 | 是否恢复 `DELETE /api/v3/contents/generations/tasks/{id}` | 本阶段 owner 裁决为去掉 DELETE 支持；后续如恢复需重新 contract-first | cancel / delete 后续扩展 |
| B1 | 价格、可售性、SLA、账户折扣真实来源 | 以 `new-api` 实配、测试渠道实测或 owner 确认为准 | 对外报价 / UI |
| O1 | `docs/api_contract.md -> api/openapi.yaml` 统一生成入口 | 当前不存在；不生成 OpenAPI | OpenAPI 产物 |
