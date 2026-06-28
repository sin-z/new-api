# XRTokenArkVideo 薄 Task Adaptor 技术方案

## 0. 需求类型判定

- 类型：服务端现有异步视频 task adaptor 扩展。
- 不新增公开 HTTP API。
- 不新增数据库表或迁移。
- 不新增通用渠道 profile 字段。
- 不改变现有 DoubaoVideo 行为。

## 0.1 方案档位判定

档位：
- [x] 轻量方案（Lite）
- [ ] 标准方案（Standard）
- [ ] 完整方案（Full）

原因：
- 本次是单一 provider 协议变体的薄 adaptor 增量。
- 不新增用户侧 HTTP API，不修改正式公开 API 契约。
- 不涉及数据库结构、消息、缓存或跨服务接口变更。

## 1. 背景与目标

`doubao-seedance-2-0-260128`、`doubao-seedance-2-0-fast-260128` 需要接入 XRToken ARK 下游。XRToken 下游与现有 DoubaoVideo 的主要差异是任务路径和查询响应结构不同。

目标：
- 新增 `XRTokenArkVideo` 专用 ChannelType `101` 和薄 task adaptor；`59-100` 在 `ChannelBaseURLs` 中作为预留空字符串占位。
- 对外继续使用现有 OpenAI Video surface：`POST /v1/video/generations`、`GET /v1/video/generations/:task_id`、`POST /v1/videos`、`GET /v1/videos/:task_id`。
- 后台渠道通过 `model_mapping` 把公开模型名映射到 XRToken 上游 `volcengine/...` 模型名。
- 保持现有 DoubaoVideo `/api/v3` adaptor 行为不回退。

## 2. 事实真相与证据链

| 来源类型 | 来源位置 | 关键结果 | 设计结论 |
| --- | --- | --- | --- |
| 需求交付说明书（SDD Spec） | `docs/plan/2026-06-28-xrtoken-ark-video-adaptor-S1-1-xrtoken-ark-video-task-adaptor-01KW73.md` | 要新增专用 ChannelType / task adaptor，不新增通用 `api_profile`，不走 AdvancedCustom，默认不新增用户侧 ARK 公开路由。 | 本方案只做内部 adaptor，不修改公开 API contract。 |
| 需求交付说明书（SDD Spec） | `docs/plan/2026-06-28-xrtoken-ark-video-adaptor-S1-1-xrtoken-ark-video-task-adaptor-01KW73.md` | 已知 XRToken 下游协议差异：create / query / delete 路径为 `/v1/contents/generations/tasks`，查询响应 `video_url` 位于顶层，删除成功为 `204`。 | create / fetch adaptor 必须覆盖 `/v1` 路径和顶层 `video_url`；delete 仅记录为非本阶段公开能力。 |
| 人类确认 | `#dev-discussion:t37` 消息 `01KW73RX`、`01KW73SF` | @xy zh 同意“增加一个薄 XRTokenArkVideo task adaptor”，并要求新开顶层任务。 | 本任务可进入独立实现，不在 #37 继续编码。 |
| 人类确认 | `#dev-discussion:t40` 消息 `01KW77K9`、`01KW77RJ`、`01KW77S9` | @xy zh 最终确认 `XRTokenArkVideo = 101`，`59-100` 作为预留空档；@arch 收口为 `ChannelBaseURLs` 补空字符串占位，不新增 BaseURL accessor。 | 最终实现和文档统一采用 ChannelType `101`，保持现有直接下标读取模式。 |
| 方案讨论 | `#dev-discussion:t37` 消息 `01KW73N3` | @arch 复核当前项目 `new-api/`：DoubaoVideo / VolcEngine 当前映射到 `taskdoubao.TaskAdaptor`，Doubao 路径为 `/api/v3/...`，XRToken 文档路径为 `/v1/...`，顶层 `video_url` 与现有 `content.video_url` 不兼容。 | 不能靠后台新增 DoubaoVideo 渠道配置完整支持，正确方案是新增薄 task adaptor。 |
| 方案讨论 | `#dev-discussion:t37` 消息 `01KW73V6` | @claude 判断默认不对用户暴露 ARK 兼容 `DELETE /v1/contents/generations/tasks/{id}`，只在 adaptor 内部适配上游。 | 本方案不新增公开 ARK 路由，不在本阶段定义 delete/cancel 本地状态和退款语义。 |
| 项目代码 | `relay/relay_adaptor.go` | `GetTaskAdaptor` 按 ChannelType 返回不同 task adaptor；`ChannelTypeDoubaoVideo` 和 `ChannelTypeVolcEngine` 当前共用 `taskdoubao.TaskAdaptor`。 | XRToken 应通过新增 ChannelType 映射到新薄 adaptor。 |
| 项目代码 | `relay/channel/task/doubao/adaptor.go` | DoubaoVideo create / fetch 路径硬编码为 `/api/v3/contents/generations/tasks`，查询结果读取 `content.video_url`。 | XRToken 不能只靠 BaseURL 配置跑通，需单独覆盖路径和响应解析。 |
| 项目代码 | `constant/channel.go` | 当前 `ChannelTypeDoubaoVideo = 54`，`ChannelBaseURLs[54] = https://ark.cn-beijing.volces.com`，`ChannelTypeNames[54] = DoubaoVideo`。 | 新增 ChannelType 时必须同步常量、默认 BaseURL 和名称，避免下标错位。 |
| 项目代码 | `relay/helper/model_mapped.go`、`model/channel.go` | 渠道已有 `ModelMapping`，请求提交阶段会把 `OriginModelName` 映射为 `UpstreamModelName`。 | 无需新增模型名字段；继续用 `model_mapping` 表达公开模型名到上游模型名。 |
| 项目代码 | `router/video-router.go`、`relay/channel/adapter.go`、`relay/relay_task.go` | 当前公开路由为 OpenAI Video 兼容路径；`TaskAdaptor` 接口没有 delete/cancel 动作。 | 默认不暴露 ARK `DELETE /v1/contents/generations/tasks/{id}`；删除语义不在本阶段实现。 |
| 项目代码 | `service/task_polling.go` | 轮询成功时写入 `PrivateData.ResultURL`；失败终态会退款，成功终态会按 adaptor 或 token 结算。 | XRToken fetch 解析到顶层 `video_url` 后进入既有成功写 URL 和结算流程。 |
| 实际执行结果 | 在 `.worktree/xrtoken-ark-video-adaptor` 执行 `codegraph status` | Project 为 `/Users/ai/workbench/projects/token168/.worktree/xrtoken-ark-video-adaptor`；Files 1,961、Nodes 29,363、Edges 83,855；`Index is up to date`。 | CodeGraph 可作为本 worktree 的定位辅助证据。 |
| 实际执行结果 | 在 `.worktree/xrtoken-ark-video-adaptor` 执行 `codegraph query "GetTaskAdaptor ChannelTypeDoubaoVideo task doubao ParseTaskResult ConvertToOpenAIVideo"` | 命中 `service/task_polling.go:30`、`relay/channel/task/doubao/adaptor.go:306`、`relay/channel/task/doubao/adaptor.go:344` 等符号。 | 图谱定位到 task adaptor 与轮询接口；最终设计仍以源码读取为准。 |
| 实际执行结果 | 在 `.worktree/xrtoken-ark-video-adaptor` 执行 `git check-ignore -v docs/plans/2026-06-28_XRTokenArkVideo薄TaskAdaptor.md` | 回改前命中 `.gitignore:25:plans`，计划文件被忽略。 | 必须增加精确 `.gitignore` 例外，确保计划文件可被本分支正常纳入交付。 |
| 实际执行结果 | Phase 2/3 Review 反馈与 owner 裁决 | 后台 default / classic 渠道类型下拉必须同步；OpenAI Video `metadata.url` 需在 `video_url` 为空时回退到 `Task.GetResultURL()`。 | 编码回改需同步前端常量 / 图标，并补 fallback 单测。 |

## 2.2 变更点 / 存量简析

### 现有能力盘点表

| 能力 | 现有实现 | 证据 | 复用结论 | 限制 |
| --- | --- | --- | --- | --- |
| 渠道级模型名映射 | `Channel.ModelMapping` + `ModelMappedHelper` | `model/channel.go`、`relay/helper/model_mapped.go` | 复用 | 只负责模型名，不表达路径和响应结构。 |
| 任务提交主流程 | `RelayTaskSubmit` | `relay/relay_task.go` | 复用 | 通过选中渠道的 platform 获取 adaptor。 |
| task adaptor 分发 | `GetTaskAdaptor` | `relay/relay_adaptor.go` | 扩展 | 需新增 XRToken ChannelType 分支。 |
| 视频公开 create / fetch 路由 | `/v1/video/generations`、`/v1/videos` | `router/video-router.go` | 复用 | 不新增 ARK 私有公开路由。 |
| OpenAI Video 查询外壳 | `ConvertToOpenAIVideo` 分支 | `relay/relay_task.go` | 复用 | 新 adaptor 需实现 converter。 |
| DoubaoVideo Seedance 请求体 | `taskdoubao.TaskAdaptor.convertToRequestPayload` | `relay/channel/task/doubao/adaptor.go` | 包装或最小复制 | 未导出，编码阶段需权衡最小导出还是独立复制。 |
| DoubaoVideo 视频输入计费估算 | `EstimateBilling` / `GetVideoInputRatio` | `relay/channel/task/doubao/adaptor.go`、`relay/channel/task/doubao/constants.go` | 包装 | 模型比率只覆盖 Seedance 2.0 两个模型。 |
| 任务轮询与结果 URL 写入 | `UpdateVideoTasks` / `updateVideoSingleTask` | `service/task_polling.go` | 复用 | 依赖 adaptor 返回 `TaskInfo.Url`。 |
| 失败退款 / 成功结算 | `RefundTaskQuota` / `settleTaskBillingOnComplete` | `service/task_polling.go` | 复用 | 本阶段不新增 delete/cancel 退款入口。 |
| 删除 / 取消任务 | 无公开路由，无 `TaskAdaptor` delete/cancel 方法 | `router/video-router.go`、`relay/channel/adapter.go` | 本次不涉及 | 后续暴露需新 contract-first。 |

### 新增能力

- 新增 `ChannelTypeXRTokenArkVideo = 101`，默认 BaseURL 为 `https://api.xrtoken.net`。
- `ChannelBaseURLs[59..100]` 填充空字符串作为预留空档，保持现有直接下标读取模式不越界。
- 新增 `relay/channel/task/xrtokenarkvideo` 包。
- `GetTaskAdaptor` 将 `ChannelTypeXRTokenArkVideo` 映射到新 adaptor。
- 新 adaptor：
  - `BuildRequestURL` 使用 `/v1/contents/generations/tasks`。
  - `FetchTask` 使用 `/v1/contents/generations/tasks/{task_id}`。
  - `DoResponse` 兼容 XRToken create response，并返回上游 task id。
  - `ParseTaskResult` 读取顶层 `video_url`、`duration`、`created_at`、`updated_at`。
  - `ConvertToOpenAIVideo` 把顶层 `video_url` 放入 `metadata.url`；当原始 `video_url` 为空时，回退到 `originTask.GetResultURL()`。
- 同步 default / classic 管理后台渠道类型下拉和图标映射，不加入普通模型拉取类型集合。

## 3. 影响范围摘要

| 范围 | 影响 |
| --- | --- |
| 渠道类型 | 新增 ChannelType `101`，`59-100` 为预留空档，不改变已有类型语义。 |
| 用户侧 API | 不新增、不修改公开 HTTP API。 |
| 后台配置 | default / classic 管理后台均可选择 `XRTokenArkVideo` 渠道并配置 `model_mapping`。 |
| 任务提交 | 仅选中 XRTokenArkVideo 渠道时走新 adaptor。 |
| 任务轮询 | 仅 XRTokenArkVideo 任务解析顶层 `video_url`。 |
| 计费 | 提交预扣和完成结算沿用现有视频 task 计费链路。 |
| DoubaoVideo | 保持 `/api/v3` 路径和 `content.video_url` 解析。 |

## 4. 需求交付说明书依据

- 输入 spec：`docs/plan/2026-06-28-xrtoken-ark-video-adaptor-S1-1-xrtoken-ark-video-task-adaptor-01KW73.md`
- 关键约束：
  - 新增专用 ChannelType / task adaptor。
  - 不新增通用 `api_profile` 字段。
  - 不走 AdvancedCustom。
  - 公开模型名继续使用 `doubao-seedance-2-0-260128`、`doubao-seedance-2-0-fast-260128`。
  - 通过 `model_mapping` 映射到 `volcengine/...`。
  - 默认不新增用户侧 ARK 兼容公开路由。
  - 删除成功 `204` 的本地任务状态、退款 / 计费语义需明确。

## 5. API 复用矩阵

| 能力 | 现有能力 | 结论 | 证据 | Owner | 说明 |
| --- | --- | --- | --- | --- | --- |
| 用户提交视频任务 | `/v1/video/generations`、`/v1/videos` | 复用 | `router/video-router.go` | `new-api` relay | 用户侧入口不变。 |
| 用户查询视频任务 | `/v1/video/generations/:task_id`、`/v1/videos/:task_id` | 复用 | `router/video-router.go`、`relay/relay_task.go` | `new-api` relay | 查询外壳由现有 `RelayTaskFetch` 提供。 |
| XRToken create 上游调用 | DoubaoVideo `/api/v3/...` | 新增 | `relay/channel/task/doubao/adaptor.go`、spec | `new-api` relay | 路径不同，需新 adaptor 使用 `/v1/...`。 |
| XRToken fetch 上游调用 | DoubaoVideo `/api/v3/...` | 新增 | `relay/channel/task/doubao/adaptor.go`、spec | `new-api` relay | 路径不同，响应字段位置不同。 |
| OpenAI Video 外壳转换 | 各 video adaptor 自行实现 `ConvertToOpenAIVideo` | 扩展 | `relay/channel/adapter.go`、`relay/relay_task.go` | `new-api` relay | 新 adaptor 需实现同一接口。 |
| 模型名映射 | `model_mapping` | 复用 | `model/channel.go`、`relay/helper/model_mapped.go` | `new-api` channel config | 无需新增字段。 |
| 计费估算 | DoubaoVideo 视频输入估算 | 包装 | `relay/channel/task/doubao/adaptor.go`、`relay/channel/task/doubao/constants.go` | `new-api` relay | 新 adaptor 复用或最小复制同一比率。 |
| 任务轮询结果处理 | `service/task_polling.go` | 复用 | `service/task_polling.go` | `new-api` service | adaptor 返回 `TaskInfo.Url` 后复用。 |

## 6. 命名域矩阵

| 使用场景 | 业务领域 | 功能模块 | 后端实现标识 | 服务边界 | 证据 / 约束 |
| --- | --- | --- | --- | --- | --- |
| XRToken ARK 视频渠道 | AI 视频生成 | task relay adaptor | `XRTokenArkVideo` / `xrtokenarkvideo` | `new-api` relay 内部渠道适配 | 任务 spec 要求专用 ChannelType / task adaptor。 |
| 公开模型名 | 模型目录 / 计费 / 用户输入 | model mapping | `doubao-seedance-2-0-260128`、`doubao-seedance-2-0-fast-260128` | `new-api` 公开模型和渠道配置 | spec 要求公开模型名保持不变。 |
| 上游模型名 | XRToken ARK provider | request payload | `volcengine/...` | XRToken 上游请求体内 | spec 要求通过 `model_mapping` 映射。 |
| 用户侧查询外壳 | OpenAI Video compatible API | video fetch | `OpenAIVideo` | `new-api` 现有公开视频路由 | `relay/relay_task.go` 已按 `/v1/videos/` 分支转换。 |
| 非本阶段删除能力 | XRToken ARK provider | task delete | 不新增公开标识 | 不进入本阶段服务边界 | 现有接口无 delete/cancel 方法；若进入后续任务需 contract-first。 |

## 7. 数据字段语义矩阵

| 字段 | 类型 | 单位 | NULL / NOT NULL | 默认值 / sentinel | 合法值冲突 | 采集 / 写入时点 | 响应表达 | 历史数据 / 回滚 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| `Channel.Type` | int | 无 | NOT NULL | 由后台创建渠道时写入 | `XRTokenArkVideo = 101`，`59-100` 为预留空档，不能与既有 ChannelType 数值重复 | 管理员新增渠道 | 后台渠道详情返回 | 回滚时禁用或删除该类型渠道配置。 |
| `Channel.BaseURL` | string | URL | 可空 | 空时使用 `ChannelBaseURLs` 默认值 | 需为 XRToken API base，不应配置到 Doubao ARK base | 管理员配置渠道 | 后台渠道详情返回 | 不影响已有渠道。 |
| `Channel.ModelMapping` | JSON string | 无 | 可空 | 空表示不映射 | 映射循环由 `ModelMappedHelper` 拦截 | 管理员配置渠道，请求提交时读取 | 不直接返回用户 | 复用既有字段，无迁移。 |
| `Task.TaskID` | string | 无 | NOT NULL | `GenerateTaskID()` | 不应写上游 id | 任务提交前本地生成 | 用户侧 task id | 沿用现有任务模型。 |
| `Task.PrivateData.UpstreamTaskID` | string | 无 | 可空；成功 create 后非空 | 空表示上游未返回 | 不应暴露给用户 | create response `id` 解析后写入 | 不直接返回用户 | 回滚代码后存量任务仍保留私有字段。 |
| `Task.PrivateData.ResultURL` | string | URL | 可空 | 空表示尚无结果 URL | 不接受 `data:` 时由轮询转 proxy URL；`video_url` 为空时 OpenAI Video 外壳以此为 fallback | fetch response 成功后写入 | OpenAI Video `metadata.url` | 回滚不改变表结构。 |
| `response.id` | string | 无 | create / fetch 响应应非空 | 空为 invalid response | 必须作为上游 task id，不覆盖 public task id | 上游 create / fetch 返回 | 内部使用 | 回滚无结构影响。 |
| `response.status` | string | 无 | 可空；空视作未知 | 空 / 未知状态继续按 in_progress 处理 | 本地仅映射到现有 `QUEUED / IN_PROGRESS / SUCCESS / FAILURE` | 上游 fetch 返回 | 用户侧映射为 OpenAI Video status | 未知状态不触发退款，避免误终态。 |
| `response.video_url` | string | URL | 可空 | 空表示结果未就绪或上游缺字段 | 与 Doubao `content.video_url` 不同，必须顶层读取 | 上游 fetch 返回 | OpenAI Video `metadata.url` | 仅新 adaptor 读取。 |
| `response.duration` | number/string | 秒 | 可空 | 空表示未知 | 不驱动本阶段计费，避免与预扣倍率冲突 | 上游 fetch 返回 | 可作为 OpenAI Video `seconds` 可选值 | 无结构迁移。 |
| `response.created_at` / `updated_at` | int64 | Unix 秒 | 可空 | 空时使用本地时间字段 | 不覆盖本地审计时间 | 上游 create / fetch 返回 | 可用于 OpenAI Video `created_at` / `completed_at` | 无结构迁移。 |

## 8. 接口设计与协议产物

本次不新增用户侧 HTTP API，不修改 `docs/api_contract.md`。

依据：
- 任务 spec 明确默认对外继续保持现有 OpenAI Video / `/v1/video/generations` surface。
- `#dev-discussion:t37` 中 @claude 的产品判断与 @arch 收口均倾向不暴露 ARK 私有路径。
- 现有 `TaskAdaptor` 接口没有 delete/cancel 动作，新增公开 `DELETE /v1/contents/generations/tasks/{id}` 会扩大公开契约和计费 / 退款语义。

若 owner 后续确认暴露 `POST/GET/DELETE /v1/contents/generations/tasks...`，必须先进入技术方案阶段更新 `docs/api_contract.md`，并补公开路由、权限、错误码、接口测试和删除语义。

## 8.1 既有能力包装边界矩阵

| 包装点 | 底层执行能力 | 适配层职责 | 权限 / 鉴权 | 幂等 | 审计 / 日志 | 状态映射 | 恢复语义 Owner |
| --- | --- | --- | --- | --- | --- | --- | --- |
| 用户 create | `RelayTaskSubmit`、渠道分发、预扣费 | XRToken adaptor 只转换上游 URL / body / response | 复用 `middleware.TokenAuth()` 和 `Distribute()` | 复用现有 public task id 与预扣费流程；不新增幂等键 | 复用现有 request log / task log | create 后本地仍为既有 task 状态流转 | `new-api` relay / billing |
| 用户 fetch | `RelayTaskFetch`、`ConvertToOpenAIVideo` | XRToken adaptor 只把顶层 `video_url` 转为 `metadata.url` | 复用 task userId 查询隔离 | GET 查询幂等 | 复用现有查询日志 | 本地 `TaskStatus` 到 OpenAI Video status | `new-api` relay |
| 任务轮询 | `service.UpdateVideoTasks` | XRToken adaptor 返回 `TaskInfo`，不直接写库 | 系统内部任务，不新增用户鉴权 | 轮询 CAS 由 `UpdateWithStatus` 承接 | 复用 `logger` 和系统日志 | XRToken status 映射到现有四类终态 / 进行态 | `new-api` service |
| 模型名映射 | `ModelMappedHelper` | adaptor 只消费 `info.UpstreamModelName` | 渠道权限仍由现有分发决定 | 映射无写副作用 | 现有日志记录 upstream / origin model | 不涉及任务状态 | `new-api` channel config |
| 删除 / 取消 | 无现有公开能力 | 本次不包装 | 本次不涉及 | 本次不涉及 | 本次不涉及 | 本次不涉及 | 后续 owner 裁决 |

## 8.2 信任链 / 统一上下文参数复用检查

| 检查项 | 结论 | 依据 |
| --- | --- | --- |
| 用户身份与 API Key 鉴权 | 复用现有链路 | `router/video-router.go` 对公开视频 route 使用 `middleware.TokenAuth()`、`middleware.Distribute()`。 |
| 渠道选择上下文 | 复用现有链路 | `Distribute()` 写入 channel type / key / base URL；`RelayInfo.InitChannelMeta` 读取统一 context。 |
| 模型名上下文 | 复用现有链路 | `ModelMappedHelper` 使用 context 中 `model_mapping`，不新增 header / query / trust chain。 |
| 上游鉴权 | 复用渠道 key | task adaptor 通过 `Authorization: Bearer <channel key>` 发起上游请求。 |
| 用户侧请求参数 | 不新增公共参数 | 本阶段不新增公开 HTTP API，不新增 `hlhttp.Atom` / `AtomWeb` 等公共参数。 |
| 自定义范围字段 / 新 header | 本次不涉及 | XRToken 差异仅在 provider path / response shape，不需要新增用户侧信任字段。 |

## 8.3 跨章节一致性扫描结果

| 扫描项 | 命中结果 | 处置 |
| --- | --- | --- |
| 旧实现前缀 `/api/v3` | 仅在 DoubaoVideo 存量能力和“不回退”说明中出现 | 保留，作为回归边界。 |
| 新上游路径 `/v1/contents/generations/tasks` | 出现在 XRToken create / fetch 设计、测试计划和非公开路由边界 | 符合本方案。 |
| `api_profile` | 仅出现在“不新增通用字段”约束中 | 保留，说明不采用该方案。 |
| `AdvancedCustom` | 仅出现在“不走 AdvancedCustom”约束中 | 保留，说明不采用该方案。 |
| `DELETE /v1/contents` / delete / cancel | 仅在“本次不涉及 / 后续 owner 裁决”语境中出现 | 不作为本阶段阻塞项。 |
| `TODO` | 无计划引入 TODO | 编码阶段如新增 TODO 需写清触发条件和补齐时机。 |
| `NULL` / `null` | 数据字段矩阵以“可空 / NOT NULL”表达 | 不涉及 SQL DDL。 |
| 新接口 / 新公开路由 | 文档明确“不新增用户侧 HTTP API” | 不修改 `docs/api_contract.md`。 |
| SQL 字段注释 | 本次无 SQL DDL | SQL lint 结果为 no SQL DDL blocks found。 |

## 8.4 非本阶段能力与后续裁决集中表

| 事项 | 本阶段状态 | Owner | 依据 | 后续触发条件 | 当前处置 |
| --- | --- | --- | --- | --- | --- |
| 暴露 ARK 兼容 `POST/GET/DELETE /v1/contents/generations/tasks...` 用户侧路由 | 本次不涉及，不阻塞本阶段 | owner / @arch / @xy zh | spec 默认不新增公开路由；#37 讨论倾向不暴露私有 ARK 路径 | owner 明确要求公开 ARK surface | 新开 contract-first 任务，更新 `docs/api_contract.md` 和接口测试。 |
| XRToken delete 成功 `204` 后本地状态 | 后续 owner 裁决，不阻塞本阶段 | owner / @arch | 当前 `TaskAdaptor` 无 delete/cancel 方法，用户无法触发删除 | 新增公开或内部 delete/cancel 能力时 | 定义终态、幂等、退款，再实现。 |
| 删除 / 取消退款语义 | 后续 owner 裁决，不阻塞本阶段 | owner / billing owner | 当前无删除入口；失败退款 / 成功结算已有现有轮询语义 | 新增 delete/cancel 能力时 | 补技术方案、合同和测试。 |
| 真实 XRToken 外部接口测试 | 依赖测试 key，不阻塞单元级 Phase 1 | owner 提供 key | 当前没有可用 XRToken 测试 key | owner 提供测试渠道 key / 环境 | 编码完成后补真实 create / fetch 验证；无 key 时记录阻塞原因。 |

## 9. 关键技术点

### 9.1 Adaptor 设计

实现策略：
- 新增独立包 `relay/channel/task/xrtokenarkvideo`，避免修改 DoubaoVideo 主流程。
- 请求体转换优先复用 DoubaoVideo 的字段结构；若复用需要导出类型或函数，只做最小导出，不改变 DoubaoVideo 语义。
- `GetModelList` 保持 Seedance 公开模型名列表。
- `GetChannelName` 返回 `xrtoken-ark-video`。

### 9.2 状态映射

| XRToken status | 本地状态 | Progress | 说明 |
| --- | --- | --- | --- |
| `pending` / `queued` | `QUEUED` | `10%` 或 `20%` | 等待执行。 |
| `processing` / `running` / `in_progress` | `IN_PROGRESS` | `50%` | 生成中。 |
| `succeeded` / `completed` | `SUCCESS` | `100%` | 读取顶层 `video_url`。 |
| `failed` | `FAILURE` | `100%` | 读取错误 message。 |
| 其他未知状态 | `IN_PROGRESS` | `30%` | 避免误退款，继续轮询。 |

### 9.3 删除 / 取消语义

本阶段不实现用户侧删除 / 取消，也不新增上游 delete 调用入口。

明确语义：
- XRToken 文档中的 delete 成功 `204` 是上游能力事实，但当前 `new-api` 公开视频任务没有对应公开删除路由。
- 不新增公开路由时，用户无法触发删除，因此不存在本地任务置取消 / 失败、退款 / 计费变更。
- 后续若 owner 要暴露删除：
  - 必须先补 `TaskAdaptor` delete/cancel 动作或专用 controller。
  - 必须定义本地终态，是 `FAILURE`、新增 `CANCELLED`，还是保持原状态。
  - 必须定义退款：删除未完成任务是否退还预扣、成功任务是否不退、重复删除是否幂等。
  - 必须补 contract-first 和接口测试。

### 9.4 外部 HTTP 调用模式

本次 adaptor 沿用现有 task adaptor 模式：
- create 调用通过 `channel.DoTaskApiRequest`。
- fetch 调用通过 `service.GetHttpClientWithProxy`。

偏离 `manager + proxy.InitHTTP + hlhttp.NewReq + [[server_client]]` 的原因：
- 当前 `new-api` provider task adaptor 体系全部围绕渠道配置动态选择 base URL、key 和 proxy。
- 本次是同一体系内新增 provider adaptor，不是新增固定下游服务 client。
- 为保持与 DoubaoVideo / Kling / Sora 等 task adaptor 一致，本次不引入新的 manager 配置链路。

## 10. 详细影响分析

- 渠道选择：管理员新增 XRTokenArkVideo 渠道后，该渠道参与现有模型分发。
- 模型映射：若未配置 `model_mapping`，上游收到公开模型名；若 XRToken 要求 `volcengine/...`，管理员必须配置映射。
- 日志与审计：`OriginModelName` 保持公开模型名；`UpstreamModelName` 记录映射后的上游名。
- 失败退款：轮询解析为 `FAILURE` 时沿用 `service.task_polling` 现有失败退款逻辑。
- 成功结算：轮询解析为 `SUCCESS` 时沿用现有完成结算逻辑；XRToken 本阶段不使用 `duration` 做额外计费调整。

## 11. 风险与解决方案

| 风险 | 解决方案 | 回滚方式 |
| --- | --- | --- |
| ChannelType 下标错位 | `XRTokenArkVideo` 固定为 `101`；同步补 `ChannelBaseURLs[59..100]` 空字符串占位、`ChannelBaseURLs[101]`、`ChannelTypeNames`；测试断言默认 BaseURL 和 adaptor 路由。 | 回滚常量、映射和 adaptor 包。 |
| 后台无法选择渠道 | 同步 default / classic 渠道类型下拉和图标映射；不加入普通模型拉取集合。 | 回滚前端常量和图标映射。 |
| DoubaoVideo 行为回退 | 独立包实现，单测覆盖 DoubaoVideo URL 仍为 `/api/v3`。 | 回滚 XRToken 包和映射，不改 Doubao 包。 |
| XRToken 返回字段不全 | parse 对空 `video_url` 保持任务成功但无 URL 时走现有 proxy fallback；错误字段只公开 message/code。 | 回滚 adaptor 后停用渠道。 |
| 删除语义未实现 | 技术方案显式不暴露、不实现，避免半成品公开能力。 | 后续按 contract-first 新任务实现。 |
| 测试环境无真实 XRToken key | 以单元测试覆盖协议适配；真实接口测试需 owner 提供 key 后另行执行。 | 无 key 时不做真实外部写调用。 |

## 12. 测试计划

按 TDD 执行：
1. 新增失败单测：`GetTaskAdaptor(ChannelTypeXRTokenArkVideo)` 返回 XRToken adaptor。
2. 新增失败单测：create URL 为 `/v1/contents/generations/tasks`。
3. 新增失败单测：fetch URL 为 `/v1/contents/generations/tasks/{id}`。
4. 新增失败单测：create response `{id, model, status, created_at}` 返回 public OpenAI Video id，内部 task id 为上游 id。
5. 新增失败单测：fetch response 顶层 `video_url` 映射到 `TaskInfo.Url`。
6. 新增失败单测：`ConvertToOpenAIVideo` 读取顶层 `video_url`，并在 `video_url` 为空时回退 `Task.GetResultURL()`。
7. 新增失败单测：`ChannelTypeXRTokenArkVideo == 101`、`ChannelBaseURLs[59..100]` 为空、`ChannelBaseURLs[101]` 可读、`model.Channel.GetBaseURL()` 对 101 不 panic。
8. 新增回归单测：DoubaoVideo create / fetch URL 仍为 `/api/v3/...`。

验证命令：
- `gofmt` 目标文件。
- `go test ./relay/channel/task/xrtokenarkvideo ./relay/channel/task/doubao ./relay -run 'XRToken|Doubao'`
- `go test ./service ./relay/channel/task/xrtokenarkvideo ./relay/channel/task/doubao ./relay`
- `go test ./model -run 'ChannelGetBaseURLUsesXRTokenDefault' -count=1`
- `git diff --check`

## 13. 自审报告

- 结论：`通过`
- 输入完整性检查：已读取任务 spec、#37 方案裁决、当前 `new-api` 源码、服务端技术方案规则、self review 规则和 Phase 1 Review 反馈。
- 章节覆盖检查：已补齐事实真相与证据链、现有能力盘点表、API 复用矩阵、命名域矩阵、数据字段语义矩阵、既有能力包装边界矩阵、信任链检查、一致性扫描结果、非本阶段能力与后续裁决集中表。
- 影响范围检查：仅新增内部 ChannelType 和 adaptor；不改数据库表、不改公开 API；不新增 delete/cancel。
- 风险与回滚检查：已列出 ChannelType `101`、`59-100` 预留占位、后台下拉、Doubao 回退、删除语义、真实 key 缺失和 Git 忽略风险；回滚方式为停用 / 删除 XRTokenArkVideo 渠道并回滚 adaptor / 后台常量相关代码。
- 事实依据 / 证据充分性检查：关键判断均有 spec、线程消息、源码路径或实际执行结果支撑；XRToken `/v1` 路径、顶层 `video_url`、delete `204` 已补 spec 证据。
- 变更边界完整性检查：默认不实现 delete/cancel；后续 owner 若要公开 ARK 路由，必须新开 contract-first 任务；ChannelType 最终口径已统一为 `101`。
- 负向关联影响检查：重点覆盖 DoubaoVideo `/api/v3` 不回退、`59-100` 预留占位、101 默认 BaseURL 可读、公开视频 surface 不变、后台新增渠道入口可选择。
- 上下游依赖检查：XRToken base URL / key / `model_mapping` 依赖后台渠道配置；真实外部接口测试依赖 owner 提供 key，不阻塞 Phase 1 文档 Review。
- 新增风险 / 问题 / 矛盾点 / 模糊项 / 重叠项检查：delete `204` 已从“阻塞待裁决”调整为“本次不涉及 / 后续 owner 裁决且不阻塞本阶段”；不存在一边阻塞一边宣称无未决的矛盾。
- 问题处置状态检查：
  - P0-1 计划文件被 `.gitignore` 忽略：`已修复`，已增加精确 `.gitignore` 例外。
  - P0-2 技术方案矩阵缺失：`已修复`，已补齐所有适用矩阵和不适用依据。
  - P0-3 delete 口径矛盾：`已修复`，已集中到非本阶段后续裁决表。
  - P0-4 自审报告缺正式结论：`已修复`，已补正式结论和问题状态。
  - P1 证据链粒度不足：`已修复`，已补 spec、线程、源码和 CodeGraph 结果粒度。
- 未决问题 / 假设检查：无影响本阶段进入编码 Review 的未决问题；真实 XRToken 外部接口测试是否执行取决于后续是否提供测试 key。
