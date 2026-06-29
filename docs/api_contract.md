# API Contract

## 1. 文档说明

本文是 `new-api` 当前服务端技术方案阶段维护的正式 Markdown 协议文档。S1.3 本轮新增 Seedance 2.0 原生接口 OpenAI 中转候选方案契约。

S1.3 不覆盖 S1.2 已验收 / 已合入的直连 native 方案。S1.3 候选路线为：

```text
native /api/v3/contents/generations/tasks
  -> internal OpenAI Video request
  -> existing relay task flow
  -> task adaptor
  -> upstream native /api/v3/contents/generations/tasks
```

本阶段只做 contract-first 和技术方案，不进入 Coding / API Testing。

## 2. 公共约定

### 2.1 Auth

所有接口使用 API Key：

```http
Authorization: Bearer <token>
```

鉴权链路复用 `TokenAuth()`。create 复用 `Distribute()` 做渠道分发；get / list 使用当前 user / API Key 权限域查询本地 task。非本人 task 与不存在 task 均返回 404，不泄漏存在性。

### 2.2 ID

| Field | Type | Required | Description | Constraints |
| --- | --- | --- | --- | --- |
| `id` | string | Yes | 对外 public task id | 必须使用 `Task.TaskID`；不得返回 `TaskPrivateData.UpstreamTaskID` |

### 2.3 Status

响应 status 可见值：

| Field | Type | Required | Description | Constraints |
| --- | --- | --- | --- | --- |
| `status` | string | Yes | native task status | `queued` / `running` / `cancelled` / `succeeded` / `failed` / `expired` |

P0 不主动产出 `cancelled`，也不新增内部 `CANCELLED` / `EXPIRED` / `DELETED` task status。`expired` 仅保留为响应兼容值，不作为 P0 list 过滤入参。

### 2.4 Error Wrapper

native API 使用 native error wrapper，不返回 OpenAI error wrapper。

| Field | Type | Required | Description | Constraints |
| --- | --- | --- | --- | --- |
| `error.code` | string | Yes | 错误码 | 使用 native / token168 映射后的错误码 |
| `error.message` | string | Yes | 错误说明 | 不包含 secret、upstream key、upstream task id |

错误码来源边界：

- 官方原生错误码：来自 BytePlus / ModelArk 官方协议或 C1 测试渠道实测，可作为官方兼容口径记录。
- token168 自定义映射码：由本服务为鉴权、分发、能力准入、本地解析失败等场景映射产生，不得写成 BytePlus 官方错误码。
- Coding 前必须在错误码表或测试用例中保留来源标记，避免把自定义码误作为官方契约承诺。

### 2.5 OpenAPI

当前仓库事实：

- 无 `api/openapi.yaml`。
- 现有参考文件为 `docs/openapi/api.json`、`docs/openapi/relay.json`。
- 未发现 `docs/api_contract.md -> api/openapi.yaml` 的统一生成入口。

因此 S1.3 本阶段不生成 OpenAPI。Markdown contract 是本阶段正式 contract-first 产物。

## 3. 接口目录

| Method | Path | Description |
| --- | --- | --- |
| `POST` | `/api/v3/contents/generations/tasks` | Create Seedance task |
| `GET` | `/api/v3/contents/generations/tasks/{id}` | Get Seedance task |
| `GET` | `/api/v3/contents/generations/tasks` | List Seedance tasks |

P0 不支持 `DELETE /api/v3/contents/generations/tasks/{id}`、cancel、delete、删除后不可查、取消退款。

## 4. POST /api/v3/contents/generations/tasks

### 4.1 Route

`POST /api/v3/contents/generations/tasks`

### 4.2 Description

创建 Seedance 2.0 native task。对外保持 BytePlus / ModelArk native request / response 心智；内部候选实现先转换为 OpenAI Video internal request，再复用 relay task 和 task adaptor 调用上游 native。

### 4.3 Auth

`Authorization: Bearer <token>`，复用 `TokenAuth()`。create 必须复用 `Distribute()` 做渠道分发，不允许请求指定或绕过 channel id。

### 4.4 Request Parameters

无。

| Name | In | Type | Required | Description | Constraints |
| --- | --- | --- | --- | --- | --- |
| 无 | 无 | 无 | 无 | 无 path / query / header 专属参数；公共 auth header 见 2.1 | 无 |

### 4.5 Request Body Fields

| Field | Type | Required | Description | Constraints |
| --- | --- | --- | --- | --- |
| `model` | string | Yes | Seedance model name | 用于模型映射和 `Distribute()`；不得固定 channel |
| `content` | array<object> | Yes | native multimodal content | 至少 1 项；必须在 OpenAI internal request metadata 和 canonical `Task.Data.request` 中保真 |
| `content[].type` | string | Yes | content item type | `text` / `image_url` / `video_url` / `audio_url`，最终值域以 C1 实测锁定 |
| `content[].text` | string | No | text prompt | `type=text` 时使用；可派生 OpenAI internal `prompt` |
| `content[].image_url.url` | string | No | image URL | `type=image_url` 时使用 |
| `content[].video_url.url` | string | No | video URL | `type=video_url` 时使用 |
| `content[].audio_url.url` | string | No | audio URL | `type=audio_url` 时使用 |
| `callback_url` | string | No | callback URL | P0 只允许缺省或空字符串；非空返回 400 |
| `return_last_frame` | boolean | No | 是否返回最后一帧 | OpenAI 顶层不可表达时放 metadata / canonical data |
| `service_tier` | string | No | service tier | 只允许缺省或 `default`；`flex` 或其他值返回 400 |
| `execution_expires_after` | integer | No | 上游任务过期秒数 | 仅作为 metadata / request snapshot；不新增本地 `EXPIRED` 状态 |
| `generate_audio` | boolean | No | 是否生成音频 | 透传给具备能力的 task adaptor |
| `resolution` | string | No | 分辨率 | 保留 native 原值；不得因 OpenAI schema 不可表达而丢弃 |
| `ratio` | string | No | 画幅比例 | 保留 native 原值 |
| `duration` | integer | No | 视频时长，单位秒 | 可同步到 OpenAI internal `seconds`；原值必须保留 |
| `frames` | integer | No | 帧数 | OpenAI 顶层不可表达时放 metadata |
| `seed` | integer | No | 随机种子 | 可表达则写 OpenAI internal 顶层；同时写 canonical data |
| `camera_fixed` | boolean | No | 是否固定镜头 | OpenAI 顶层不可表达时放 metadata |
| `watermark` | boolean | No | 是否添加水印 | OpenAI 顶层不可表达时放 metadata |
| `priority` | integer | No | 调用方优先级提示 | P0 不允许用于绕过 `Distribute()` 或指定 channel |

### 4.6 Success Response Fields

| Field | Type | Required | Description | Constraints |
| --- | --- | --- | --- | --- |
| `id` | string | Yes | public task id | 必须是 `Task.TaskID`；不得是上游真实 id |

示例：

```json
{
  "id": "task_public_id"
}
```

### 4.7 Error Codes

| Field | Type | Required | Description | Constraints |
| --- | --- | --- | --- | --- |
| `InvalidParameter` | error code | No | request body、`model`、`content`、`callback_url`、`service_tier` 或字段值域错误 | HTTP 400 |
| `UnsupportedChannel` | error code | No | `Distribute()` 选中的 channel 不具备 Seedance native task adaptor / renderer 能力 | HTTP 400 / 503 |
| `ChannelUnavailable` | error code | No | 无可用 channel 或渠道分发失败 | 沿用 relay 错误语义后映射 native wrapper |
| `UpstreamError` | error code | No | 上游创建 task 返回错误 | 不暴露 upstream key / upstream task id |

### 4.8 Compatibility

- response 不返回 OpenAI Video wrapper。
- native-only 字段必须通过 OpenAI internal request metadata 和 canonical `Task.Data.request` 保真。
- create 只能返回 public id。
- 本接口不支持 callback 调用；非空 `callback_url` 返回 400。
- 本接口不承诺幂等键。
- 本接口不触发 DELETE/cancel 或 cancel refund 语义。

## 5. GET /api/v3/contents/generations/tasks/{id}

### 5.1 Route

`GET /api/v3/contents/generations/tasks/{id}`

### 5.2 Description

按 public task id 查询当前用户可见的 Seedance task，并按 BytePlus native task object 渲染。

### 5.3 Auth

`Authorization: Bearer <token>`，复用 `TokenAuth()`。查询范围限定当前 user / API Key 权限域。

### 5.4 Request Parameters

| Name | In | Type | Required | Description | Constraints |
| --- | --- | --- | --- | --- | --- |
| `id` | path | string | Yes | public task id | 必须是 `Task.TaskID`；非本人 / 不存在均返回 404 |

### 5.5 Request Body Fields

无。

| Field | Type | Required | Description | Constraints |
| --- | --- | --- | --- | --- |
| 无 | 无 | 无 | GET 无请求体 | 无 |

### 5.6 Success Response Fields

| Field | Type | Required | Description | Constraints |
| --- | --- | --- | --- | --- |
| `id` | string | Yes | public task id | `Task.TaskID` |
| `model` | string | Yes | Seedance model name | 优先来自 canonical `Task.Data.model` / task properties |
| `status` | string | Yes | native task status | `queued` / `running` / `cancelled` / `succeeded` / `failed` / `expired` |
| `content.video_url` | string | No | result video URL | 成功后返回；未知为空或省略，Coding 需统一 |
| `content.last_frame_url` | string | No | last frame URL | 仅任务产生时返回 |
| `seed` | integer | No | random seed | 未知为 0 或省略，Coding 需统一 |
| `resolution` | string | No | resolution | 来自 canonical request / upstream result |
| `duration` | integer | No | duration seconds | 来自 canonical request / upstream result |
| `ratio` | string | No | aspect ratio | 来自 canonical request / upstream result |
| `framespersecond` | integer | No | fps | 上游返回时渲染；是否必返由 C1 锁定 |
| `service_tier` | string | No | service tier | P0 仅 `default` |
| `usage.completion_tokens` | integer | No | completion token count | 未知为 0 或省略，Coding 需统一 |
| `usage.total_tokens` | integer | No | total token count | 未知为 0 或省略，Coding 需统一 |
| `error.code` | string | No | task error code | failed 时返回 |
| `error.message` | string | No | task error message | 不包含 secret |
| `created_at` | integer | Yes | created Unix seconds | 来自 task submit time / canonical data |
| `updated_at` | integer | Yes | updated Unix seconds | 来自 task update time / canonical data |

### 5.7 Error Codes

| Field | Type | Required | Description | Constraints |
| --- | --- | --- | --- | --- |
| `NotFound` | error code | No | task 不存在、非本人、或不能按 Seedance native renderer 渲染 | HTTP 404，不泄漏存在性 |
| `InvalidParameter` | error code | No | path id 为空或格式非法 | HTTP 400 |
| `InternalError` | error code | No | 本地 task data 无法解析 | HTTP 500 或按实现映射；不得泄露内部字段 |

### 5.8 Compatibility

- 支持 OpenAI->native 互查：OpenAI Video create 的 Seedance task 可用 public id 从 native get 查询。
- 支持 native->native 查询。
- response wrapper 由查询入口决定；本接口永远返回 native wrapper。
- 不通过 upstream task id 对外查询。
- 非本人 task 与不存在 task 均返回 404。
- `framespersecond`、`safety_identifier` 等字段是否属于官方必返，继续作为 C1 实测项锁定；未锁定前不新增确定必返承诺。

## 6. GET /api/v3/contents/generations/tasks

### 6.1 Route

`GET /api/v3/contents/generations/tasks`

### 6.2 Description

分页列出当前用户可见的 Seedance native tasks。P0 限定最近 7 天任务，且只返回可按 Seedance native renderer 渲染的任务。

### 6.3 Auth

`Authorization: Bearer <token>`，复用 `TokenAuth()`。查询范围限定当前 user / API Key 权限域。

### 6.4 Request Parameters

| Name | In | Type | Required | Description | Constraints |
| --- | --- | --- | --- | --- | --- |
| `page_num` | query | integer | No | 页码 | 从 1 开始；默认值由实现固定 |
| `page_size` | query | integer | No | 每页数量 | 需有实现上限 |
| `filter.status` | query | string | No | status filter | 仅允许 `queued` / `running` / `cancelled` / `succeeded` / `failed`；不含 `expired` |
| `filter.task_ids` | query | array<string> | No | public task id filter | 只能使用 `Task.TaskID` |
| `filter.model` | query | string | No | model filter | Seedance model name |
| `filter.service_tier` | query | string | No | service tier filter | P0 仅 `default` |

### 6.5 Request Body Fields

无。

| Field | Type | Required | Description | Constraints |
| --- | --- | --- | --- | --- |
| 无 | 无 | 无 | GET 无请求体 | 无 |

### 6.6 Success Response Fields

| Field | Type | Required | Description | Constraints |
| --- | --- | --- | --- | --- |
| `items` | array<object> | Yes | task list | 当前 user 可见且可 native 渲染的 task |
| `items[].id` | string | Yes | public task id | `Task.TaskID` |
| `items[].model` | string | Yes | Seedance model name | 同 get |
| `items[].status` | string | Yes | native task status | 响应可含 `queued` / `running` / `cancelled` / `succeeded` / `failed` / `expired` |
| `items[].content.video_url` | string | No | result video URL | 成功后返回 |
| `items[].content.last_frame_url` | string | No | last frame URL | 上游返回时渲染 |
| `items[].seed` | integer | No | random seed | 同 get |
| `items[].resolution` | string | No | resolution | 同 get |
| `items[].duration` | integer | No | duration seconds | 同 get |
| `items[].ratio` | string | No | aspect ratio | 同 get |
| `items[].framespersecond` | integer | No | fps | C1 锁定 |
| `items[].service_tier` | string | No | service tier | P0 仅 `default` |
| `items[].usage.completion_tokens` | integer | No | completion token count | 同 get |
| `items[].usage.total_tokens` | integer | No | total token count | 同 get |
| `items[].error.code` | string | No | task error code | failed 时返回 |
| `items[].error.message` | string | No | task error message | failed 时返回 |
| `items[].created_at` | integer | Yes | created Unix seconds | 同 get |
| `items[].updated_at` | integer | Yes | updated Unix seconds | 同 get |
| `total` | integer | Yes | total count | 当前过滤条件下总数 |

### 6.7 Error Codes

| Field | Type | Required | Description | Constraints |
| --- | --- | --- | --- | --- |
| `InvalidParameter` | error code | No | query 参数非法，如 `filter.status=expired` | HTTP 400 |
| `InternalError` | error code | No | 本地 task data 无法解析或查询失败 | HTTP 500 或按实现映射 |

### 6.8 Compatibility

- `filter.status` 入参不支持 `expired`；响应 status 可包含 `expired`。
- 当前无 cancel 产出源，`filter.status=cancelled` 可以返回空结果。
- list 不返回非当前 user task。
- list 不返回不能由 Seedance native renderer 渲染的 channel / task。
- `framespersecond`、`safety_identifier` 等字段是否属于官方必返，继续作为 C1 实测项锁定；未锁定前不新增确定必返承诺。

## 7. Canonical Task.Data

OpenAI Video response builder 和 native response builder 必须读取同一份 canonical data。

兼容性要求：canonical `Task.Data` 必须是既有 Doubao `responseTask` 顶层结构的超集。不得改动既有顶层 `id`、`model`、`status`、`content.video_url`、`seed`、`usage`、`error`、`created_at`、`updated_at` 字段名与语义；新增 native request snapshot 只能放入 `request` 子对象。Coding 必须保证现有 `ConvertToOpenAIVideo` 仍可读取顶层字段，否则四组合互查中的 native->OpenAI / OpenAI->OpenAI 渲染会取空。

| Field | Type | Required | Description | Constraints |
| --- | --- | --- | --- | --- |
| `id` | string | Yes | public task id | `Task.TaskID` |
| `model` | string | Yes | model name | origin model |
| `status` | string | Yes | native status | 与内部状态映射，不新增内部 CANCELLED / EXPIRED / DELETED |
| `created_at` | integer | Yes | created Unix seconds | task submit time |
| `updated_at` | integer | Yes | updated Unix seconds | task update time |
| `content.video_url` | string | No | result video URL | 不含 upstream task id |
| `content.last_frame_url` | string | No | last frame URL | 上游返回时写入 |
| `request.content` | array<object> | Yes | original native content | native-only 字段保真 |
| `request.prompt` | string | No | derived prompt | 从 text content 派生 |
| `request.resolution` | string | No | original resolution | 保留 native 原值 |
| `request.ratio` | string | No | original ratio | 保留 native 原值 |
| `request.duration` | integer | No | original duration | seconds |
| `request.frames` | integer | No | original frames | frames |
| `request.generate_audio` | boolean | No | original generate_audio | 保留 native 原值 |
| `request.return_last_frame` | boolean | No | original return_last_frame | 保留 native 原值 |
| `request.camera_fixed` | boolean | No | original camera_fixed | 保留 native 原值 |
| `request.watermark` | boolean | No | original watermark | 保留 native 原值 |
| `usage.completion_tokens` | integer | No | completion tokens | 未知为 0 或省略，Coding 需统一 |
| `usage.total_tokens` | integer | No | total tokens | 未知为 0 或省略，Coding 需统一 |
| `error.code` | string | No | error code | failed 时返回 |
| `error.message` | string | No | error message | 不含 secret |
| `seed` | integer | No | seed | request / upstream result |
| `priority` | integer | No | priority snapshot | 不用于绕过 `Distribute()` |
| `service_tier` | string | No | service tier | P0 `default` |
| `execution_expires_after` | integer | No | upstream expiration seconds | 不新增本地 `EXPIRED` 状态 |

`Task.Data` 不得包含 upstream task id。上游真实 id 只允许存储在 `TaskPrivateData.UpstreamTaskID`。

## 8. Billing And Task State Boundaries

S1.3 必须复用：

- `RelayTaskSubmit`
- `PreConsumeBilling`
- submit billing adjustment
- task polling by `TaskPrivateData.UpstreamTaskID`
- terminal `SUCCESS` / `FAILURE`
- `RefundTaskQuota`
- `RecalculateTaskQuota`
- `RecalculateTaskQuotaByTokens`

S1.3 不得新增 native-specific pre-consume、settlement branch、cancel refund、内部 `CANCELLED` / `EXPIRED` / `DELETED` 或 upstream task id 对外暴露。
