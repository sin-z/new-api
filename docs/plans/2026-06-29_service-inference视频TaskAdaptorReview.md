# service-inference.ai 视频 Task Adaptor Phase 1 计划 Review

## Date

2026-06-29

## Review 对象

- 任务 spec：`docs/plan/2026-06-29-service-inference-video-adaptor-S1-1-service-inference-video-task-adaptor-01KW7W.md`
- Phase 1 计划：`.worktree/service-inference-video-adaptor/docs/plans/2026-06-29_service-inference视频TaskAdaptor.md`
- 计划索引：`.worktree/service-inference-video-adaptor/docs/plan_index.md`
- 目标 worktree：`.worktree/service-inference-video-adaptor`
- 目标分支：`feature_codex/service_inference_video_adaptor`

## Review 结论

首轮结论：`有问题需回改`

当前计划方向正确：默认不新增用户侧 service-inference.ai 公开路由、不改 `docs/api_contract.md`、先补技术方案再进入 TDD 编码，这些边界与 spec 一致。但计划遗漏后台渠道可选入口相关写集，且任务 spec 路径在 `new-api` worktree 内不可追溯；这两项会影响后续实现完整性和交付可验证性，需先回改 Phase 1 计划后再进入技术方案 Review。

复审结论：`通过`

@doer 已回改 `.worktree/service-inference-video-adaptor/docs/plans/2026-06-29_service-inference视频TaskAdaptor.md` 与 `.worktree/service-inference-video-adaptor/docs/plan_index.md`。本轮复审确认上一轮 P0 / P1 问题均已闭环，Phase 1 计划可进入技术方案阶段；技术方案通过 Review 前仍不得进入编码。

## 复审记录

- `P0-1` 已修复：计划 `Affected` 已补充 `web/default/src/features/channels/constants.ts`、`web/default/src/features/channels/lib/channel-utils.ts`、`web/classic/src/constants/channel.constants.js`、`web/classic/src/helpers/render.jsx`，并在 `Acceptance`、`Risks`、`Docs` 和 Phase 2 计划中补齐 default / classic 后台渠道入口与图标映射要求。
- `P0-2` 已修复：计划已明确当前 root 相对路径 `docs/plan/2026-06-29-service-inference-video-adaptor-S1-1-service-inference-video-task-adaptor-01KW7W.md`，以及从 `new-api` worktree 出发的相对路径 `../../docs/plan/2026-06-29-service-inference-video-adaptor-S1-1-service-inference-video-task-adaptor-01KW7W.md`，并说明该 spec 属于上层 token168 编排输入，不属于 `new-api` 实际写入仓。
- `P1-1` 已修复：`docs/plan_index.md` 已调整为 `Date / Task / Status / Plan / OverallPlan / ScopeSummary` 字段，并兼容补齐历史 XRToken 行。

## P0 问题

### P0-1：计划漏列两套后台渠道类型与图标映射写集

状态：`已修复`

事实依据：

- Spec 明确要求“能在后台新增 `service-inference.ai` 视频渠道并配置 `model_mapping` 后，创建 / 查询 Seedance 2.0 视频任务成功”，见 `docs/plan/2026-06-29-service-inference-video-adaptor-S1-1-service-inference-video-task-adaptor-01KW7W.md` 第 37 行。
- Phase 1 计划的计划内候选写集只列出 `constant/channel.go`、`relay/relay_adaptor.go`、`model/channel_serviceinferencevideo_test.go`、`relay/channel/task/serviceinferencevideo/*`、文档文件，见 `.worktree/service-inference-video-adaptor/docs/plans/2026-06-29_service-inference视频TaskAdaptor.md` 第 34-55 行。
- 当前 `new-api` 已有两套后台渠道类型配置：`web/default/src/features/channels/constants.ts` 第 75-86 行包含 `DoubaoVideo` / `XRTokenArkVideo` 类型与展示顺序；`web/classic/src/constants/channel.constants.js` 第 173-180 行包含 classic 渠道类型下拉。
- 当前 `new-api` 已有两套图标映射：`web/default/src/features/channels/lib/channel-utils.ts` 第 95-105 行包含视频类渠道图标；`web/classic/src/helpers/render.jsx` 第 404-406 行把 DoubaoVideo / XRTokenArkVideo 映射到 Doubao 图标。

影响：

- 只新增后端 ChannelType 与 task adaptor，不能保证管理员在 default / classic 后台实际选择 `service-inference.ai` 渠道。
- 计划验收项“后台可新增”无法被现有候选写集支撑。

回改要求：

- 在计划 `Affected` 中补齐至少以下候选写集：`web/default/src/features/channels/constants.ts`、`web/default/src/features/channels/lib/channel-utils.ts`、`web/classic/src/constants/channel.constants.js`、`web/classic/src/helpers/render.jsx`。
- 在 `Acceptance` / 验证项中补充 default / classic 后台渠道下拉和图标映射不回退检查；如新增可见文案需要 i18n 键，明确是否需要同步 locale 文件及依据。
- 技术方案阶段必须明确 `service-inference.ai` ChannelType 编号、`ChannelTypeDummy` 处理、`ChannelBaseURLs` 下标安全，以及 default / classic 两套后台展示入口。

### P0-2：计划中的任务 spec 路径在目标 worktree 内不可追溯

状态：`已修复`

事实依据：

- Phase 1 计划在 `Scope` 与 `Implementation Plan` 中写任务 spec 为 `docs/plan/2026-06-29-service-inference-video-adaptor-S1-1-service-inference-video-task-adaptor-01KW7W.md`，见 `.worktree/service-inference-video-adaptor/docs/plans/2026-06-29_service-inference视频TaskAdaptor.md` 第 20 行和第 114 行。
- 在目标 worktree `.worktree/service-inference-video-adaptor` 中执行 `test -f docs/plan/2026-06-29-service-inference-video-adaptor-S1-1-service-inference-video-task-adaptor-01KW7W.md` 返回不存在。
- 同一文件在当前 root 下存在：`docs/plan/2026-06-29-service-inference-video-adaptor-S1-1-service-inference-video-task-adaptor-01KW7W.md`。

影响：

- 计划文件落在 `new-api` 独立 worktree 中，但计划内写的 spec 路径像是 `new-api` 仓内路径；后续只拿 `new-api` worktree / source branch Review 时无法按该路径追溯需求输入。

回改要求：

- 在计划中明确区分两类路径：当前 root 相对路径 `docs/plan/...01KW7W.md`，以及从 `new-api` worktree 出发的相对路径 `../../docs/plan/...01KW7W.md`。
- 写明该 spec 属于上层 token168 编排输入，不属于 `new-api` 实际写入仓；后续技术方案引用 spec 时使用可解析路径。

## P1 问题

### P1-1：`docs/plan_index.md` 字段不满足服务端 workflow 最小索引字段

状态：`已修复`

事实依据：

- `harness-engineering/server_golang_workflow.md` 要求 `docs/plan_index.md` 的计划摘要索引至少包含 `Date`、`Task`、`Status`、`Plan`、`OverallPlan`、`ScopeSummary`。
- 当前 `.worktree/service-inference-video-adaptor/docs/plan_index.md` 只有 `Date`、`Plan`、`Status`、`Path` 四列。

回改要求：

- 将本次新增索引行调整为 workflow 要求的字段；如为兼容既有历史行暂不全量改表，至少补充本任务行可表达的 `Task`、`Plan`、`OverallPlan`、`ScopeSummary` 信息，并在计划中说明兼容处理。

## 已独立验证

- `git -C harness-engineering pull --ff-only`：通过，输出 `Already up to date.`。
- `python3 harness-engineering/tools/harness_env.py doctor --profile bootstrap`：通过，输出 `bootstrap: pass`、`app_key is set`。
- `.worktree/service-inference-video-adaptor` Git 状态：分支为 `feature_codex/service_inference_video_adaptor`，当前为计划 / 索引 / Review 留痕变更，未修改业务代码。
- `git check-ignore -v docs/plans/2026-06-29_service-inference视频TaskAdaptor.md docs/plan_index.md docs/changes.md`：无输出，本次计划文件未被 `.gitignore` 忽略。
- `go test ./relay/channel/task/doubao ./relay/channel/task/xrtokenarkvideo ./relay ./model -count=1`：通过。
- `git diff --check -- docs/plans/2026-06-29_service-inference视频TaskAdaptor.md docs/plans/2026-06-29_service-inference视频TaskAdaptorReview.md docs/plan_index.md`：无输出。
- `codegraph sync && codegraph status`：通过，输出 `Index is up to date`；`.codegraph` 未被 Git 跟踪。
- `test -f ../../docs/plan/2026-06-29-service-inference-video-adaptor-S1-1-service-inference-video-task-adaptor-01KW7W.md`：通过，输出 `spec-ok`。

## 后续动作

@doer 可进入 `docs/tech-design/service-inference-video-task-adaptor.md` 技术方案阶段，并再次提交 @arch Review。技术方案 Review 通过前，不进入编码。

## 技术方案 Review

### Date

2026-06-29

### Review 对象

- 技术方案：`.worktree/service-inference-video-adaptor/docs/tech-design/service-inference-video-task-adaptor.md`
- 计划文件：`.worktree/service-inference-video-adaptor/docs/plans/2026-06-29_service-inference视频TaskAdaptor.md`
- 计划索引：`.worktree/service-inference-video-adaptor/docs/plan_index.md`
- 目标 worktree：`.worktree/service-inference-video-adaptor`
- 目标分支：`feature_codex/service_inference_video_adaptor`

### Review 结论

结论：`通过`

本轮 Review 未发现阻断编码的 P0 / P1 问题。技术方案已覆盖服务端技术方案规范要求的事实证据链、现有能力盘点、API 复用、命名域、字段语义、包装边界、信任链、一致性扫描、待裁决项、接口协议产物边界、影响分析、风险和自审报告。方案边界与 Phase 1 计划一致：本任务默认不新增用户侧 service-inference.ai 公开路由，不修改 `docs/api_contract.md`，不生成 OpenAPI 产物。

通过依据：

- service-inference.ai 被定义为独立 provider task adaptor，新增 ChannelType、task adaptor 包和 `GetTaskAdaptor` 分发，不走 AdvancedCustom，不复用 Doubao / XRToken 的错误路径解析。
- 用户侧入口继续复用现有 OpenAI Video `/v1/video/generations`、`/v1/videos` 创建和查询路由，上游 `/v1/video/generate`、`/v1/video/tasks/{taskId}` 仅作为 provider API。
- 方案明确 `ChannelTypeServiceInferenceVideo = 102`，`ChannelTypeDummy` 后移到 `103`，并要求补齐 `ChannelBaseURLs[102]` 与 `ChannelTypeNames`，覆盖直接下标读取风险。
- 方案明确 create response 从 `task.id` 取上游 task id，fetch response 从 `task.outputs[0]` 取主结果 URL，完整上游响应含 `outputs[]` 保留到 `Task.Data`。
- 方案明确 `last_frame_url` 不进入 OpenAI Video 外壳，仅随原始响应保留，避免扩大公开契约。
- 方案同步覆盖 default / classic 两套后台渠道入口与图标映射写集。
- 技术方案阶段实际写集未触碰 adaptor、测试或后台入口实现代码。

### 裁决项

- service-inference.ai `GET /v1/video/tasks` list 端点：本次不进入默认实现。轮询已有明确 upstream task id，`GET /v1/video/tasks/{taskId}` 能满足精准查询；如后续真实上游 get by id 不稳定，再新开或补充 list 兜底方案和测试。
- default 后台 locale：本次不补 locale 文件。现有 `XRTokenArkVideo` 已以常量选项方式接入且未检索到对应 locale 键，本任务保持最小变更；若运行态展示异常，再最小补齐 locale。

### P1 实现提醒

- 编码阶段检查 `controller/channel-test.go` 中普通渠道测试的 unsupported 列表是否需要加入 `ChannelTypeServiceInferenceVideo`，避免后台“测试渠道”误走 chat / ordinary test 链路；若现有链路不需要加入，需在自 review 中给出源码依据。
- `ChannelTypeDummy` 后移到 `103` 后，测试需覆盖 `ChannelBaseURLs[102]`、`ChannelBaseURLs[103]` 或相关枚举不 panic / 不错位，尤其关注 `controller/model.go` 通过 `ChannelTypeDummy` 上界枚举渠道类型的行为。
- 若编码时发现必须新增或修改用户侧 HTTP API、错误码、响应字段或公开协议，立即停止 Coding，退回技术方案阶段补 `docs/api_contract.md` / OpenAPI 产物并重新 Review。

### 已独立验证

- `git -C harness-engineering pull --ff-only`：通过，输出 `Already up to date.`。
- 目标 worktree `python3 ../../harness-engineering/tools/harness_env.py doctor --profile bootstrap`：通过，输出 `bootstrap: pass`、`app_key is set`。
- 目标 worktree `codegraph sync && codegraph status`：通过，输出 `Index is up to date`。
- `git diff --check -- docs/plan_index.md`：无输出。
- `git diff --no-index --check /dev/null docs/tech-design/service-inference-video-task-adaptor.md`：无 whitespace 输出；退出码 1 来自 no-index 差异本身。
- trailing whitespace 专项检查覆盖技术方案、计划、索引：无输出。
- 实现写集越界检查 `git status --porcelain=v1 | rg '(^ M|^MM|^A |^\?\?) (constant/|relay/|model/|service/|router/|web/default/|web/classic/|.*_test\.go)' || true`：无输出，确认技术方案阶段未触碰业务代码、测试或后台入口实现代码。
- `.codegraph` 跟踪检查 `git ls-files .codegraph && git status --short .codegraph`：无输出。

### 后续动作

@doer 可按技术方案进入 TDD 编码阶段。编码阶段不得新增用户侧 service-inference.ai 公开路由，不得修改 `docs/api_contract.md` / OpenAPI；如发现协议边界必须扩大，停止编码并退回技术方案 Review。

## Phase 2 编码复审

### Date

2026-06-29

### Review 对象

- 实现写集：
  - `constant/channel.go`
  - `relay/relay_adaptor.go`
  - `relay/channel/task/serviceinferencevideo/constants.go`
  - `relay/channel/task/serviceinferencevideo/adaptor.go`
  - `relay/channel/task/serviceinferencevideo/adaptor_test.go`
  - `relay/relay_adaptor_xrtokenarkvideo_test.go`
  - `model/channel_xrtokenarkvideo_test.go`
  - `controller/channel-test.go`
  - `controller/service_inference_video_channel_test.go`
  - `web/default/src/features/channels/constants.ts`
  - `web/default/src/features/channels/lib/channel-utils.ts`
  - `web/classic/src/constants/channel.constants.js`
  - `web/classic/src/helpers/render.jsx`
- 文档写集：
  - `docs/changes.md`
  - `docs/plans/2026-06-29_service-inference视频TaskAdaptor.md`
  - `docs/plan_index.md`
  - `docs/tech-design/service-inference-video-task-adaptor.md`
  - `docs/plans/2026-06-29_service-inference视频TaskAdaptorReview.md`
- 目标 worktree：`.worktree/service-inference-video-adaptor`
- 目标分支：`feature_codex/service_inference_video_adaptor`

### Review 结论

结论：`通过`

本轮复审未发现阻断合入或必须回改的问题。实现与已通过 Review 的技术方案一致：service-inference.ai 作为独立薄 task adaptor 接入，用户侧继续复用现有 OpenAI Video 路由，不新增 service-inference.ai 公开路由，不修改 `docs/api_contract.md` / OpenAPI。

通过依据：

- `constant/channel.go` 已新增 `ChannelTypeServiceInferenceVideo = 102`，`ChannelTypeDummy = 103`，并补齐 `ChannelBaseURLs[102] = "https://model.service-inference.ai"`、`ChannelBaseURLs[103] = ""` 与渠道名 `service-inference.ai`。
- `relay/relay_adaptor.go` 已将 `ChannelTypeServiceInferenceVideo` 分发到 `relay/channel/task/serviceinferencevideo.TaskAdaptor`，DoubaoVideo 和 XRTokenArkVideo 分发保持独立。
- 新 adaptor 创建任务路径为 `/v1/video/generate`，轮询路径为 `/v1/video/tasks/{taskId}`，未实现 list 兜底，符合技术方案裁决。
- create response 从 `{task:{id}}` 取上游 task id，并对外返回 public task id；`model_mapping` 仍由 `RelayTaskSubmit` 先设置 `info.UpstreamModelName`，再由复用的 Doubao `BuildRequestBody` 写入上游请求体。
- fetch response 成功态取 `task.outputs[0]` 为主 URL，usage 传给 `TaskInfo`；完整原始响应由现有轮询 `redactVideoResponseBody` 写入 `Task.Data`，不会裁掉 `task.outputs[]`。
- OpenAI Video 查询外壳使用 public task id 和 origin model；`last_frame_url` 未写入 `metadata`，只随原始响应保留。
- `controller/channel-test.go` 已将该渠道加入普通 channel test unsupported 列表；`controller/service_inference_video_channel_test.go` 覆盖 `ChannelTypeDummy` 上界、普通模型列表排除和 default / classic 后台入口文件内容。
- default / classic 后台入口已同步渠道常量、展示顺序和 Doubao 图标映射；default locale 按技术方案裁决未补。

### 剩余风险

- `go test ./... -count=1` 根包仍因 `main.go:44:12: pattern web/classic/dist: no matching files found` setup 失败；过滤根包后的全量 Go 测试已通过。该问题属于当前仓 embed 静态目录前置缺失，不是本次 adaptor 实现引入。
- 前端 lint / typecheck 未拿到通过结果，复跑确认当前 `web/` 依赖门禁和本地工具链缺失：
  - `bun install --frozen-lockfile`：`lockfile had changes, but lockfile is frozen`
  - `bun --filter newapi-web lint`：`oxlint: command not found`
  - `bun --filter newapi-web typecheck`：`tsgo: command not found`
  - `bun --filter react-template lint`：`prettier: command not found`
- 上述前端风险已在计划 self review 中标记为 `接受风险`；本轮复审接受该风险，原因是本次前端写集仅为渠道常量和图标映射，且已用 Go controller 测试静态覆盖入口内容。

### 已独立验证

- `git -C harness-engineering pull --ff-only`：通过，输出 `Already up to date.`。
- `go test ./relay/channel/task/serviceinferencevideo ./relay/channel/task/doubao ./relay/channel/task/xrtokenarkvideo ./relay ./model ./controller -count=1`：通过。
- `go test $(go list -e ./... 2>/dev/null | grep -v '^github.com/QuantumNous/new-api$') -count=1`：通过。
- `go test ./... -count=1`：根包 setup 失败，原因仅为 `web/classic/dist` 缺失；其余包通过。
- `gofmt -l constant/channel.go relay/relay_adaptor.go relay/relay_adaptor_xrtokenarkvideo_test.go model/channel_xrtokenarkvideo_test.go controller/channel-test.go controller/service_inference_video_channel_test.go relay/channel/task/serviceinferencevideo/*.go`：无输出。
- `git diff --check`：无输出。
- `codegraph sync && codegraph status`：通过，输出 `Index is up to date`；`.codegraph` 未被 Git 跟踪。
- 前端脚本复跑结果与 @doer 汇报一致，失败原因均为 lockfile frozen 或缺本地工具链。

### 后续动作

@doer 请确认本轮 Review 结论和剩余风险记录无遗漏；如无问题，可等待 @xy zh 最终验收后进入后续收口流程。

## Phase 3 收口复审

### Date

2026-06-29

### Review 对象

- `#43.3` 验证、自 review 与提交复审材料。
- 计划与自 review：`docs/plans/2026-06-29_service-inference视频TaskAdaptor.md`
- 变更记录：`docs/changes.md`
- 计划索引：`docs/plan_index.md`
- Review 留痕：`docs/plans/2026-06-29_service-inference视频TaskAdaptorReview.md`
- 根仓 memory：`/Users/ai/workbench/projects/token168/memories/important-findings.md`

### Review 结论

结论：`通过`

本轮 #43.3 复审未发现新的实现漂移、边界扩大或未记录风险。自 review 结论为 `通过`，问题状态已按 `接受风险` 记录；#43.3 未新增实现写集，未新增 service-inference.ai 用户侧公开路由，未修改 `docs/api_contract.md` / OpenAPI，未扩大到 list / delete / cancel。

通过依据：

- 边界反查 `git diff --name-only -- docs/api_contract.md docs/openapi api router controller relay/relay_task.go service/task_polling.go service/task_billing.go` 仅命中 `controller/channel-test.go`，该文件改动已在 Phase 2 复审中确认为 ordinary channel test unsupported 列表接入。
- `docs/plans/2026-06-29_service-inference视频TaskAdaptor.md` 已包含 `Implementation Result`、`Verification`、`Self Review`，自 review 结论为 `通过`，并将根包 embed dist 缺失、前端工具链 / lockfile 阻断标为 `接受风险`。
- `docs/changes.md` 已记录 service-inference.ai adaptor、ChannelType、上游 create / fetch 路径、outputs[]、OpenAI Video 外壳、后台入口和 contract 边界。
- `docs/plan_index.md` 当前任务状态为 `completed`，字段满足 `Date / Task / Status / Plan / OverallPlan / ScopeSummary`。
- 根仓 `memories/important-findings.md` 已记录技术方案裁决、Phase 2 实现收口和 Phase 2 复审事实；本轮没有整理或覆盖其他任务的未提交内容。

### 已独立验证

- `git -C harness-engineering pull --ff-only`：通过，输出 `Already up to date.`。
- `go test ./relay/channel/task/serviceinferencevideo ./relay/channel/task/doubao ./relay/channel/task/xrtokenarkvideo ./relay ./model ./controller -count=1`：通过。
- `go test $(go list -e ./... 2>/dev/null | grep -v '^github.com/QuantumNous/new-api$') -count=1`：通过。
- `go test ./... -count=1`：根包 setup 失败，原因仍为 `main.go:44:12: pattern web/classic/dist: no matching files found`；其余包通过。
- `gofmt -l constant/channel.go relay/relay_adaptor.go relay/relay_adaptor_xrtokenarkvideo_test.go model/channel_xrtokenarkvideo_test.go controller/channel-test.go controller/service_inference_video_channel_test.go relay/channel/task/serviceinferencevideo/*.go`：无输出。
- `git diff --check`：无输出。
- `git diff --check -- memories/important-findings.md`：无输出。
- `codegraph sync && codegraph status`：通过，输出 `Already up to date` / `Index is up to date`；`.codegraph` 未被 Git 跟踪。
- 前端验证复跑结果：
  - `cd web && bun install --frozen-lockfile`：失败，`lockfile had changes, but lockfile is frozen`。
  - `cd web && bun --filter newapi-web lint`：失败，`oxlint: command not found`。
  - `cd web && bun --filter newapi-web format:check`：失败，退出码 `1`。
  - `cd web && bun --filter newapi-web typecheck`：失败，`tsgo: command not found`。
  - `cd web && bun --filter react-template lint`：失败，`prettier: command not found`。
  - `cd web && bun --filter react-template eslint`：失败，ESLint `9.39.2` 未找到 `eslint.config.(js|mjs|cjs)`。

### 剩余风险

- 根包 `go test ./...` 仍受 `web/classic/dist` embed 前置产物缺失阻断；过滤根包全量 Go 测试通过。
- 前端脚本仍受 frozen lockfile 和本地工具链缺失阻断；本次前端改动为渠道常量 / 图标映射，已由 Go controller 静态测试覆盖入口内容。
- 以上风险均已在计划 self review 中记录为 `接受风险`，本轮复审继续接受。

### 后续动作

@doer 可按后续指令进入最终 Git 收口 / 分支处理。收口时继续保持当前边界：不新增用户侧 service-inference.ai 公开路由，不修改 `docs/api_contract.md` / OpenAPI，不补 list / delete / cancel，不补 default locale。
