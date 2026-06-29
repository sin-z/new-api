# service-inference.ai 视频 Task Adaptor 实施计划

## Date

2026-06-29

## Task

S 1.1 实现 service-inference.ai 视频 task adaptor。

## Branch

`feature_codex/service_inference_video_adaptor`

## Scope

- 实际写入仓：`new-api/`
- 当前实施 worktree：`.worktree/service-inference-video-adaptor`
- 目标服务 / 工程：`new-api`
- 需求来源：
  - 当前 root 相对路径：`docs/plan/2026-06-29-service-inference-video-adaptor-S1-1-service-inference-video-task-adaptor-01KW7W.md`
  - 从 `new-api` worktree 出发的相对路径：`../../docs/plan/2026-06-29-service-inference-video-adaptor-S1-1-service-inference-video-task-adaptor-01KW7W.md`
  - 该 spec 属于上层 token168 编排输入，不属于 `new-api` 实际写入仓。
- 默认对外 surface：继续使用现有 OpenAI Video `/v1/video/generations`、`/v1/videos` 创建 / 查询能力。
- 非目标：
  - 不新增通用 `api_profile` 字段。
  - 不走 `AdvancedCustom`。
  - 不新增删除 / 取消能力。
  - 默认不新增用户侧 `POST /v1/video/generate`、`GET /v1/video/tasks`、`GET /v1/video/tasks/{id}` 路由。
  - 不改变 DoubaoVideo `/api/v3` native adaptor 语义。
  - 不改变 XRTokenArkVideo adaptor 语义。

## Affected

计划内候选写集：

- `constant/channel.go`
  - 新增 service-inference.ai 专用 ChannelType、默认 base URL、后台渠道名称。
- `relay/relay_adaptor.go`
  - 注册新 ChannelType 到专用 task adaptor。
- `relay/relay_adaptor_xrtokenarkvideo_test.go`
  - 增加 service-inference.ai adaptor 路由测试，并保留 DoubaoVideo / XRTokenArkVideo 不回退断言。
- `model/channel_serviceinferencevideo_test.go`
  - 验证默认 base URL。
- `web/default/src/features/channels/constants.ts`
  - 在新版后台渠道类型、展示顺序或类型映射中补充 `service-inference.ai`，支撑后台新增渠道。
- `web/default/src/features/channels/lib/channel-utils.ts`
  - 在新版后台渠道图标 / 展示辅助映射中补充 `service-inference.ai`。
- `web/classic/src/constants/channel.constants.js`
  - 在 classic 后台渠道类型下拉中补充 `service-inference.ai`。
- `web/classic/src/helpers/render.jsx`
  - 在 classic 后台渠道图标映射中补充 `service-inference.ai`。
- `relay/channel/task/serviceinferencevideo/constants.go`
  - 定义渠道名 `service-inference.ai` 与公开模型列表。
- `relay/channel/task/serviceinferencevideo/adaptor.go`
  - 新增专用薄 task adaptor。
- `relay/channel/task/serviceinferencevideo/adaptor_test.go`
  - 覆盖 URL、create response、fetch status、outputs 保留、OpenAI Video 转换、异常状态映射。
- `docs/tech-design/service-inference-video-task-adaptor.md`
  - 技术方案阶段产物，编码前提交 @arch Review。
- `docs/plans/2026-06-29_service-inference视频TaskAdaptor.md`
  - 本计划文件。
- `docs/plan_index.md`
  - 追加计划索引。
- `docs/changes.md`
  - 实现完成后追加实际变更记录。

计划外但需要保持只读核对：

- `relay/channel/task/doubao/adaptor.go`
- `relay/channel/task/xrtokenarkvideo/adaptor.go`
- `relay/channel/task/doubao/adaptor_test.go`
- `relay/channel/task/xrtokenarkvideo/adaptor_test.go`
- `model/task.go`
- `service/task_polling.go`
- `relay/relay_task.go`

## Facts

- `new-api/` 是独立 Git 仓，当前 worktree 分支为 `feature_codex/service_inference_video_adaptor`。
- `new-api` 已有 `go.mod` / `go.sum`，依赖模式为 Go Modules。
- 任务 spec 文件位于上层 token168 root 的 `docs/plan/`，不是 `new-api` 仓内文件；`new-api` worktree 内需通过 `../../docs/plan/...01KW7W.md` 追溯。
- 当前 `new-api/docs/api_contract.md` 已存在，但本任务默认不新增或修改用户侧 HTTP API；因此默认不更新 `docs/api_contract.md`、不生成 OpenAPI。
- 当前仓已有 `docs/openapi/api.json` 和 `docs/openapi/relay.json` 参考产物，未发现 `docs/api_contract.md -> api/openapi.yaml` 的统一生成入口。
- 现有 XRTokenArkVideo 已作为同类薄 adaptor 存在：新增 ChannelType、默认 BaseURL、`GetTaskAdaptor` 注册、独立 `relay/channel/task/xrtokenarkvideo` 包和测试。
- `new-api` 现有两套后台渠道入口：新版后台在 `web/default/src/features/channels/constants.ts` 与 `web/default/src/features/channels/lib/channel-utils.ts` 维护渠道类型 / 图标；classic 后台在 `web/classic/src/constants/channel.constants.js` 与 `web/classic/src/helpers/render.jsx` 维护渠道类型 / 图标。
- CodeGraph 在 `.worktree/service-inference-video-adaptor` 初始化并完成 `codegraph sync && codegraph status`，索引状态为 up to date。
- 基线测试 `go test ./relay/channel/task/doubao ./relay/channel/task/xrtokenarkvideo ./relay ./model` 已通过。

## Risks

- ChannelType 数值必须避开现有 `ChannelTypeDummy = 102` 与保留区，避免后台渠道枚举冲突。
- 新增 ChannelType 时必须同步处理 `ChannelTypeDummy`、`ChannelBaseURLs` 下标和 default / classic 后台渠道入口，否则管理员无法在后台稳定新增 `service-inference.ai` 渠道。
- 新版后台或 classic 后台如存在 locale / i18n 键约束，技术方案阶段必须基于源码确认是否需要同步 locale 文件；若现有渠道名直接使用常量展示，则不额外新增 locale。
- `model_mapping` 必须只改变上游 `model`，不能暴露上游 `dreamina-seedance-2-0-260128` 为公开模型 ID。
- service-inference.ai create response 包裹为 `{task:{...}}`，不能误按 XRToken 顶层 `id` 解析。
- service-inference.ai fetch response 的主结果位于 `task.outputs[0]`，但全部 `outputs[]` 必须保留在 `Task.Data` 或轮询写回数据中，不能只保留第一个。
- 状态映射必须明确 `pending`、运行态、`completed`、失败态和未知态；未知态应继续轮询，不应误触发失败退款。
- 当前 `dto.OpenAIVideo` 无原生 `last_frame_url` 字段；计划默认不新增外壳字段，仅在原始任务数据中保留上游字段。
- task adaptor 现有 `FetchTask` 使用 `service.GetHttpClientWithProxy` + `http.NewRequest` 风格；本任务沿用 task adaptor 既有模式，不引入新 HTTP client 抽象。
- 如果 Review / owner 要新增 service-inference.ai 兼容用户侧路径，必须先回到技术方案和 contract-first 阶段更新 `docs/api_contract.md`，本计划当前实现范围不包含该路径。

## Acceptance

- 后台可新增 `service-inference.ai` 视频渠道，并通过 `model_mapping` 将公开模型映射到 `dreamina-seedance-2-0-260128`。
- default 后台渠道类型下拉、渠道展示顺序和图标映射包含 `service-inference.ai`，且 DoubaoVideo / XRTokenArkVideo 展示不回退。
- classic 后台渠道类型下拉和图标映射包含 `service-inference.ai`，且 DoubaoVideo / XRTokenArkVideo 展示不回退。
- 创建任务调用 `POST /v1/video/generate`，请求体兼容 `model`、`content[]`、`duration`、`resolution`、`ratio`、`generate_audio`、`watermark`、`return_last_frame` 等字段。
- 创建响应从 `{task:{id,...}}` 保存上游 task id，对用户返回 public task id 的 OpenAI Video 外壳。
- 查询任务调用 `GET /v1/video/tasks/{taskId}`；如技术方案确认列表兜底必要，则实现并测试 `GET /v1/video/tasks` 兜底。
- 轮询成功时 `task.outputs[0]` 写入主结果 URL，完整 `outputs[]` 保留在任务数据 / 日志可追溯位置。
- 兼容 `last_frame_url`、`usage.completion_tokens`、`usage.total_tokens`、`duration_seconds`、`created_at`、`completed_at`。
- 状态映射、失败 reason、未知态继续轮询均有单元测试覆盖。
- DoubaoVideo 和 XRTokenArkVideo 现有测试继续通过。
- `go test`、`gofmt`、`git diff --check` 等适用验证通过。

## Docs

- 先产出技术方案：`docs/tech-design/service-inference-video-task-adaptor.md`。
- 技术方案必须明确 `service-inference.ai` ChannelType 编号、`ChannelTypeDummy` 处理、`ChannelBaseURLs` 下标安全、default / classic 两套后台展示入口、图标映射、是否需要 locale / i18n 文件。
- 默认不更新 `docs/api_contract.md`，依据是本任务默认不新增用户侧 HTTP API，不改变既有 OpenAI Video surface。
- 实现完成后更新 `docs/changes.md`。
- 若实现发现必须改变 HTTP 契约，停止编码并先更新技术方案与 `docs/api_contract.md` 后提交 Review。

## Implementation Plan

### Phase 1：计划与技术方案

1. 已完成规范同步：`git -C harness-engineering pull --ff-only`。
2. 已读取入口规范：`AGENTS.md`、`harness-engineering/global_workflow.md`、`harness-engineering/server_golang_rules.md`、`harness-engineering/server_golang_workflow.md`、`harness-engineering/server_tech_design_rules.md`。
3. 已读取服务端代码改动必读规则与命中规则：可读性 / 日志 / 安全、服务边界、工程结构、manager、model、contract-first、错误响应、幂等、测试交付、文档同步、基础库复用、外部 HTTP、依赖模式。
4. 已读取任务 spec：当前 root 相对路径 `docs/plan/2026-06-29-service-inference-video-adaptor-S1-1-service-inference-video-task-adaptor-01KW7W.md`；从 `new-api` worktree 出发为 `../../docs/plan/2026-06-29-service-inference-video-adaptor-S1-1-service-inference-video-task-adaptor-01KW7W.md`。
5. 已完成 `new-api` Git 基线与独立 worktree 准备。
6. 本计划提交 @arch Review。
7. Review 通过后，在同一分支补齐技术方案 `docs/tech-design/service-inference-video-task-adaptor.md`，并再次提交 @arch Review。

### Phase 2：TDD 实现

1. 先写失败测试：
   - service-inference.ai ChannelType / BaseURL / Name / GetTaskAdaptor。
   - `BuildRequestURL` 使用 `/v1/video/generate`。
   - `FetchTask` 使用 `/v1/video/tasks/{taskId}`。
   - `DoResponse` 解析 `{task:{id,...}}` 并返回上游 task id。
   - `ParseTaskResult` 映射 `pending`、运行态、`completed`、失败态、未知态。
   - `ParseTaskResult` 在成功时取 `outputs[0]` 为主 URL，并保留原始 `outputs[]` 在任务数据中。
   - `ConvertToOpenAIVideo` 使用 public task id、origin model、duration_seconds、created_at、completed_at、usage 与失败 error。
   - DoubaoVideo / XRTokenArkVideo adaptor 类型不回退。
   - default / classic 后台渠道下拉和图标映射包含 `service-inference.ai`，且既有视频渠道不回退。
2. 运行目标测试，确认新增测试先失败。
3. 最小实现：
   - 新增 `relay/channel/task/serviceinferencevideo` 包。
   - 复用 Doubao Seedance 请求体转换与计费估算。
   - 覆盖 service-inference.ai 的 create / fetch URL、create response、fetch response 解析。
   - 新增 ChannelType 与默认 base URL。
   - 同步 default / classic 两套后台渠道类型和图标映射。
   - 接入 `GetTaskAdaptor`。
4. 运行目标测试并保持全绿。
5. `gofmt` 触达 Go 文件。

### Phase 3：验证、自 review 与复审

1. 运行目标单元测试：
   - `go test ./relay/channel/task/serviceinferencevideo`
   - `go test ./relay/channel/task/doubao ./relay/channel/task/xrtokenarkvideo ./relay ./model`
2. 根据变更范围补充必要包测试；若耗时可控，执行 `go test ./...`。
3. 执行 `git diff --check`。
4. 执行 `codegraph sync && codegraph status`，并用 CodeGraph / 源码复核关键入口。
5. 按 `harness-engineering/docs/rules/self_review_rules.md` 做自 review，输出自 review 文件。
6. 更新 `docs/changes.md` 和本计划状态。
7. 提交 @arch Review。

## Confirmation

Phase 1 计划已通过 @arch Review。技术方案已通过 @arch Review，编码阶段按 TDD 完成。

## Status

completed

## Implementation Result

- 新增 `ChannelTypeServiceInferenceVideo = 102`，`ChannelTypeDummy = 103`，并补齐 `ChannelBaseURLs[102]`、`ChannelBaseURLs[103]` 与渠道名称。
- 新增 `relay/channel/task/serviceinferencevideo` 专用薄 adaptor，复用 Doubao Seedance 请求体和计费估算。
- 创建任务调用 `POST /v1/video/generate`；轮询只调用 `GET /v1/video/tasks/{taskId}`，未实现 `GET /v1/video/tasks` list 端点。
- 创建响应解析 `{task:{id,...}}`，内部保存上游 task id，对用户继续返回 public task id。
- 查询响应解析 `task.outputs[]`、`usage`、`duration_seconds`、`created_at`、`completed_at`、`error`、`last_frame_url`；成功主 URL 取 `outputs[0]`，完整原始响应继续由轮询流程写入任务数据。
- 状态映射覆盖 `pending` / `queued`、运行态、成功态、失败态和未知态；未知态继续轮询。
- default / classic 后台渠道入口已加入 `service-inference.ai`，图标复用 Doubao；default locale 按 Review 裁决本次不补。
- `controller/channel-test.go` 已将 `ChannelTypeServiceInferenceVideo` 加入普通 channel test unsupported 列表。
- 未新增用户侧 service-inference.ai 公开路由，未修改 `docs/api_contract.md` / OpenAPI。

## Verification

- `go test ./relay/channel/task/serviceinferencevideo ./relay/channel/task/doubao ./relay/channel/task/xrtokenarkvideo ./relay ./model ./controller -count=1`：通过。
- `go test $(go list -e ./... 2>/dev/null | grep -v '^github.com/QuantumNous/new-api$') -count=1`：通过。
- `go test ./... -count=1`：根包 setup 失败，原因是 `main.go:44:12: pattern web/classic/dist: no matching files found`；除根包外输出包均通过。
- `gofmt -l constant/channel.go relay/relay_adaptor.go relay/relay_adaptor_xrtokenarkvideo_test.go model/channel_xrtokenarkvideo_test.go controller/channel-test.go controller/service_inference_video_channel_test.go relay/channel/task/serviceinferencevideo/*.go`：无输出。
- `git diff --check`：无输出。
- `codegraph sync && codegraph status`：通过，`Index is up to date`。
- `.codegraph` 跟踪检查：`git status --short .codegraph && git ls-files .codegraph` 无输出。
- 前端验证限制：
  - `bun install --frozen-lockfile` 在 `web/` 下失败：`lockfile had changes, but lockfile is frozen`，未擅自更新 `web/bun.lock`。
  - `bun --filter newapi-web lint` 失败：`oxlint: command not found`。
  - `bun --filter newapi-web format:check` 失败，退出码 `1`。
  - `bun --filter newapi-web typecheck` 失败：`tsgo: command not found`。
  - `bun --filter react-template lint` 失败：`prettier: command not found`。
  - `bun --filter react-template eslint` 失败：`bunx` 获取到 ESLint `9.39.2`，当前项目仍使用旧式 `eslintConfig`，报缺少 `eslint.config.(js|mjs|cjs)`。

## Self Review

- 结论：`通过`
- 事实依据 / 证据充分性检查：实现边界来自任务 spec、已通过 Review 的技术方案和 @arch 技术方案裁决；源码核对确认未新增用户侧 service-inference.ai 公开路由，未修改 `docs/api_contract.md` / OpenAPI。
- 变更边界完整性检查：实际写集覆盖 ChannelType、task adaptor、adaptor 分发、默认 BaseURL、default / classic 后台入口、测试、计划、索引和 changes；未触达路由、公开 HTTP 契约或 OpenAPI。
- 负向关联影响检查：保留 DoubaoVideo / XRTokenArkVideo adaptor 分发回归测试；新增 `ChannelTypeDummy` 上界、`ChannelBaseURLs[102]` / `[103]` 与普通 channel test unsupported 覆盖。
- 上下游依赖检查：上游仅按技术方案支持 create 与单任务 fetch；list 兜底未实现，后续真实上游需要时需退回方案和测试补充。
- 新增风险 / 问题 / 矛盾点 / 模糊项 / 重叠项检查：前端 lint / typecheck 未通过验证，根因是当前 worktree 缺本地依赖且 frozen install 会改 lockfile；根包 `go test ./...` 受 `web/classic/dist` 缺失阻断。
- 问题处置状态检查：Go 目标测试、过滤根包全量测试、格式、diff、CodeGraph 已闭环；前端工具链和根包 embed 前置问题标记为 `接受风险`，未擅自修改依赖锁或生成 dist。
- 问题清单与状态：
  - `接受风险`：`go test ./... -count=1` 根包因 `web/classic/dist` 缺失 setup 失败；过滤根包全量 Go 测试已通过。
  - `接受风险`：前端脚本因本地依赖缺失 / frozen lockfile 限制未拿到通过结果；本次仅改渠道常量入口，已用 controller 测试覆盖入口文件内容。
