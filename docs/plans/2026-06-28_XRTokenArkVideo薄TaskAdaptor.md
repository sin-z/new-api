# 2026-06-28 XRTokenArkVideo 薄 Task Adaptor 计划

## Date

2026-06-28

## Task

实现 XRToken ARK Seedance 视频下游渠道专用薄 task adaptor，使创建 / 查询协议由专用 adaptor 承接，并默认保持现有 OpenAI Video 用户侧 surface。

## Branch

`feature_codex/xrtoken-ark-video-adaptor`

## Scope

- 新增 `XRTokenArkVideo` 专用 ChannelType，最终编号为 `101`；`59-100` 在 `ChannelBaseURLs` 中作为预留空字符串占位。
- 同步两套管理后台渠道类型下拉和图标映射，允许管理员选择 `XRTokenArkVideo` 类型创建渠道。
- 新增薄 task adaptor，复用 DoubaoVideo 请求体、模型列表和视频输入计费估算逻辑。
- 上游 create / fetch 路径使用 `/v1/contents/generations/tasks`。
- create response 兼容 `{id, model, status, created_at}`。
- fetch response 兼容顶层 `video_url`、`duration`、`created_at`、`updated_at`。
- 默认不新增用户侧 `POST/GET/DELETE /v1/contents/generations/tasks...` 公开路由。
- 不新增通用 `api_profile` 字段，不走 AdvancedCustom。
- 不改变现有 DoubaoVideo `/api/v3/contents/generations/tasks` 行为。

## Affected

- `constant/channel.go`
- `relay/relay_adaptor.go`
- `relay/channel/task/xrtokenarkvideo/`
- `relay/channel/task/doubao/`（只在必要时做最小复用导出或保持不改）
- `service/task_polling.go` 相关测试覆盖（如需要验证轮询解析）
- `web/default/src/features/channels/constants.ts`
- `web/default/src/features/channels/lib/channel-utils.ts`
- `web/classic/src/constants/channel.constants.js`
- `web/classic/src/helpers/render.jsx`
- `docs/tech-design/xrtoken-ark-video-task-adaptor.md`
- `docs/plans/2026-06-28_XRTokenArkVideo薄TaskAdaptor.md`
- `docs/plan_index.md`
- `docs/changes.md`
- `.gitignore`（增加本计划文件的精确例外，确保 Git 收口可追踪）

## Risks

- ChannelType 最终按 owner 确认使用 `101`，需在 `ChannelBaseURLs` 中补齐 `59-100` 预留空字符串占位，避免现有直接下标读取越界。
- 若复用 DoubaoVideo 代码时改动原 adaptor，可能回退现有 `/api/v3` 行为；实现优先新增独立包隔离风险。
- XRToken 查询响应 `video_url` 在顶层，不能继续只读取 `content.video_url`。
- 当前 `TaskAdaptor` 接口没有 delete/cancel 动作，默认不实现公开删除路由；删除成功 `204` 仅作为后续 owner 确认后扩展项，不在本阶段对外承诺。
- 外部 HTTP 调用沿用项目既有 task adaptor 的 `service.GetHttpClientWithProxy` 模式；本次是补齐既有 adaptor 体系内的新 provider，不引入 `manager + server_client` 新链路。

## Acceptance

- XRTokenArkVideo 渠道可配置 `model_mapping`，公开模型名可映射到 `volcengine/...` 上游模型名。
- 管理后台 default / classic 均可在渠道类型下拉中选择 `XRTokenArkVideo`，且不加入普通模型拉取类型集合。
- 创建任务请求 URL 为 `<base_url>/v1/contents/generations/tasks`。
- 查询任务请求 URL 为 `<base_url>/v1/contents/generations/tasks/{upstream_task_id}`。
- 创建响应向用户返回 OpenAI Video 兼容对象，任务内部保存上游原始响应和私有上游 task id。
- 查询成功后顶层 `video_url` 写入任务结果 URL，并可通过 OpenAI Video 查询外壳返回 `metadata.url`；若原始 `video_url` 为空，则回退到 `Task.GetResultURL()`。
- DoubaoVideo 仍使用 `/api/v3/contents/generations/tasks` 并读取 `content.video_url`。
- 单元测试覆盖 create URL / fetch URL / create response / fetch parse / OpenAI Video 转换 / adaptor 路由选择 / DoubaoVideo 不回退。
- `gofmt`、`go test` 定向包验证通过。

## Docs

- 技术方案：`docs/tech-design/xrtoken-ark-video-task-adaptor.md`
- 本计划：`docs/plans/2026-06-28_XRTokenArkVideo薄TaskAdaptor.md`
- 计划索引：`docs/plan_index.md`
- 变更记录：`docs/changes.md`

## Confirmation

- Phase 1 先提交 @arch Review。
- Review 通过后进入 TDD 编码：先写失败单测，再实现最小代码，再运行验证。

## Status

`completed`

## Self Review

- 结论：`通过`
- 事实依据 / 证据充分性检查：已依据任务 spec、已通过 Review 的技术方案、@xy zh 在 `#dev-discussion:t40` 最终确认的 `101` 编号口径、源码 `constant/channel.go`、`relay/relay_adaptor.go`、`relay/channel/task/doubao/adaptor.go`、`model/ability.go` 和实际测试结果完成实现判断；CodeGraph 已在本 worktree 重新同步并显示 index up to date。
- 变更边界完整性检查：仅新增内部 `XRTokenArkVideo` ChannelType `101`、`59-100` BaseURL 预留占位、默认 BaseURL、task adaptor 分发、后台类型下拉 / 图标和 `relay/channel/task/xrtokenarkvideo` 包；未新增用户侧 ARK 公开路由，未实现 delete/cancel，未新增通用 `api_profile` 字段，未走 AdvancedCustom。
- 负向关联影响检查：已补 DoubaoVideo `/api/v3/contents/generations/tasks` create / fetch URL 回归测试；`GetTaskAdaptor` 仍保持 DoubaoVideo 返回 `taskdoubao.TaskAdaptor`。
- 上下游依赖检查：后台能力表来自管理员配置的 `channel.Models` 和 `model_mapping`，不依赖 `ChannelType2APIType`；本阶段不为 XRToken 视频渠道新增通用 chat/image `APIType`，也不加入前端普通模型拉取类型集合，避免误接普通 relay adaptor / 模型拉取链路。
- 新增风险 / 问题 / 矛盾点 / 模糊项 / 重叠项检查：真实 XRToken 外部接口测试仍依赖 owner 提供测试 key；delete/cancel 语义仍按技术方案归入后续 owner 裁决，不阻塞本阶段内部 adaptor。
- 问题处置状态检查：发现的导出方法中文注释缺口、上游 `created_at` / `duration` 未映射缺口均已修复并通过定向测试。
- 验证证据：
  - `go test ./relay -run 'XRTokenArkVideoChannelMetadata|GetTaskAdaptorReturnsXRTokenArkVideoAdaptor|GetTaskAdaptorKeepsDoubaoVideoAdaptor' -count=1` 通过。
  - `go test ./model -run 'ChannelGetBaseURLUsesXRTokenDefault' -count=1` 通过。
  - `go test ./relay/channel/task/xrtokenarkvideo -run 'DoResponse|ConvertToOpenAIVideo' -count=1` 通过。
  - `go test ./relay/channel/task/xrtokenarkvideo ./relay/channel/task/doubao ./relay -run 'ChannelMetadata|GetTaskAdaptor|BuildRequestURL|FetchTask|DoResponse|ParseTaskResult|ConvertToOpenAIVideo' -count=1` 通过。
  - `go test ./service ./relay/channel/task/xrtokenarkvideo ./relay/channel/task/doubao ./relay -count=1` 通过。
  - `go vet ./model ./service ./relay/channel/task/xrtokenarkvideo ./relay/channel/task/doubao ./relay` 通过。
  - `git diff --check` 无输出。
  - `git diff --check -- memories/important-findings.md` 无输出。
  - `python3 /Users/ai/workbench/projects/token168/harness-engineering/tools/tech_design_sql_lint.py docs/tech-design/xrtoken-ark-video-task-adaptor.md` 输出 `no SQL DDL blocks found`。
  - `python3 harness-engineering/tools/harness_env.py doctor --profile bootstrap` 输出 `bootstrap: pass`。
  - `codegraph sync && codegraph status` 显示本 worktree index up to date。
  - `bun run format:check` / `bun run lint` 未执行成功：当前环境无 `bun`；改用 `npm run format:check` / `npm run lint` 复核时，default 因缺 `oxfmt`、classic 因缺 `prettier` 失败，未安装依赖以避免扩大环境变更。
  - `staticcheck` / `golangci-lint` 本机未安装，未执行全局安装。
