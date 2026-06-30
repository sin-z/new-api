# Seedance OpenAI 中转 new-api Coding 计划

## Date

2026-06-29

## Task

S1.4 · Seedance OpenAI 中转 `new-api` Coding。

## Branch

- 实际写入仓：`new-api`
- Worktree：`.worktree/seedance-bridge-coding-new-api`
- 分支：`feature_codex/seedance-bridge-coding`
- 基线：`origin/main` / `2880bec1adf69c449d7bc6647e4ae2ed1f2fcbc1`

## Scope

### 目标

基于已通过 Review 的 S1.3 技术方案与 `new-api/docs/api_contract.md`，在 `new-api` 内实现 Seedance native create / get / list：

- `POST /api/v3/contents/generations/tasks`
- `GET /api/v3/contents/generations/tasks/{id}`
- `GET /api/v3/contents/generations/tasks`

实现路线固定为 native request -> OpenAI Video internal request -> relay task -> task adaptor -> upstream native。

### 改动范围

- 路由：在视频路由中补 `/api/v3/contents/generations/tasks` create/get/list。
- Handler / 协议适配：新增 Seedance native request 校验、native -> OpenAI Video internal request 转换、native error renderer、native response renderer。
- Relay / adaptor：复用 `TokenAuth()`、`Distribute()`、`RelayTaskSubmit` 和现有 task adaptor；仅在必要处解耦上游响应解析与 HTTP response 渲染，保持 OpenAI Video 行为不回退。
- Task data：维护 canonical `Task.Data`，保持 Doubao `responseTask` 顶层字段语义，新增 native request snapshot 只放入 `request` 子对象。
- List：只包装现有用户维度 task 查询事实源，增加必要 filter 映射，不新建 list 存储能力。
- 测试：按 TDD 补单元 / 集成级测试，覆盖 create/get/list、OpenAI/native create-get 四组合互查、native-only 字段保真、public id / upstream id 隔离。
- 文档：更新本计划、`docs/plan_index.md`、`docs/changes.md`、自 review 留痕。

### 排除范围

- 不实现 `DELETE /api/v3/contents/generations/tasks/{id}`。
- 不实现 cancel / delete。
- 不新增内部 `CANCELLED` / `EXPIRED` / `DELETED` 状态。
- 不改 `PreConsumeBilling`、`RefundTaskQuota`、`RecalculateTaskQuota`、`RecalculateTaskQuotaByTokens`、轮询终态结算 / 退款状态机。
- 不生成 OpenAPI；如发现必须变更 contract 或生成 OpenAPI，停止 Coding 并退回技术方案 Review。

## Affected

预计触达文件：

- `router/video-router.go`
- `controller/relay.go` 或新增 `controller/seedance_native.go`
- `relay/relay_task.go`
- `relay/channel/adapter.go`
- `relay/channel/task/doubao/adaptor.go`
- `model/task.go`
- 对应 `*_test.go`
- `docs/plans/2026-06-29_SeedanceOpenAI中转Coding.md`
- `docs/plan_index.md`
- `docs/changes.md`

实际触达文件以最终 diff 为准；若实现中证明需要触达核心账务、轮询或状态机文件，将停止并退回 Review。

## Risks

- native create 需要返回 `{ "id": "<public task id>" }`，但现有 adaptor `DoResponse` 会直接写 OpenAI Video wrapper；需最小解耦解析与响应写出，避免 OpenAI Video create 行为回退。
- canonical `Task.Data` 既要支撑 OpenAI wrapper，又要支撑 native renderer；需保证既有 `responseTask` 顶层字段不变，native-only 字段不丢失。
- list 只能包装现有用户维度 task 查询事实源；若现有 DAO filter 不足，只允许最小补齐查询条件，不改变生命周期和状态机。
- 真实 HTTP API Testing 依赖后续 S1.5 用例包与本地服务启动条件；本任务优先完成仓内 Go 测试和可执行验证。

## Acceptance

- create/get/list 路由存在并复用 `TokenAuth()` / `Distribute()`。
- create 返回 public `Task.TaskID`，不暴露 `TaskPrivateData.UpstreamTaskID`。
- OpenAI/native create-get 四组合互查测试通过。
- native-only 字段进入 metadata / canonical `Task.Data`，无静默丢弃。
- native get/list 只渲染 Seedance native 可渲染任务；非本人 / 不存在 / 不可渲染统一 native 404。
- list 包装现有用户维度 task 查询事实源，限定当前用户和最近 7 天。
- 未新增 DELETE/cancel，未新增内部 `CANCELLED` / `EXPIRED` / `DELETED`。
- 目标 Go 测试、`gofmt`、`go test`、`go vet`、`git diff --check`、CodeGraph 状态通过，或记录明确环境阻断。
- 完成自 review 并提交 @arch Review。

## Docs

- 计划：`docs/plans/2026-06-29_SeedanceOpenAI中转Coding.md`
- 自 review：`docs/plans/2026-06-29_SeedanceOpenAI中转Coding自review.md`
- 索引：`docs/plan_index.md`
- 变更记录：`docs/changes.md`
- Contract 输入：`docs/api_contract.md`
- 上游 spec 位于 token168 root：`../../docs/plan/2026-06-29-seedance-bridge-next-S1-4-seedance-bridge-coding-01KW99.md`
- S1.3 技术方案位于 token168 root：`../../docs/tech-design/token-gateway/seedance-2-native-openai-bridge-server-tech-design.md`

## Confirmation

- 任务线程 `#dev-discussion:t45` 中 @arch 明确 Task #45 分配给 @doer。
- 本计划提交 @arch 确认后进入 TDD 与实现。

## Status

completed; submitted to @arch Review.
