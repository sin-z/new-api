# Seedance 2.0 原生接口 OpenAI 中转代码开发计划

Date: 2026-07-02

Task: Seedance 2.0 原生接口 OpenAI 中转代码开发

Branch: `feature_codex/seedance-2-native-openai-bridge`

Scope:

- 新增 Seedance native `/api/v3/contents/generations/tasks` create / get / list 路由和 handler。
- 将 native create 请求在对外 API handler 内转换为现有 OpenAI Video task request，并复用现有 relay task 提交、计费、落库和轮询链路。
- 拆分 Doubao task adaptor 的创建响应解析和 HTTP response 渲染，保持既有 OpenAI Video 行为。
- 新增 native renderer，list 过滤在 native API handler 内完成，不扩展原 task model 查询接口。

Affected:

- `router/video-router.go`
- `controller/**`
- `relay/channel/task/doubao/**`
- `docs/plans/`、`docs/plan_index.md`、`docs/changes.md`

Risks:

- 不可表达 native 字段必须进入 metadata 与 canonical `Task.Data.request`，不得静默丢弃。
- OpenAI wrapper 写出拆分必须不破坏 `/v1/videos` 与 `/v1/video/generations`。
- native get/list 必须只按当前 user 和 public task id 渲染，不泄漏 upstream id 或其他用户任务。
- 本地真实 HTTP 接口测试可能受服务配置阻塞，阻塞时必须记录证据。

Acceptance:

- create 返回 `{ "id": "<public task id>" }`。
- native get/list 返回 BytePlus native task object 和 native error shell。
- OpenAI Video get 仍可读取 canonical task data 并返回 OpenAI wrapper。
- list 支持最近 7 天、分页、status、task_ids、model、service_tier 过滤。
- 单测覆盖参数校验、请求映射、adaptor 响应拆分、native renderer 和四组合互查。

Docs:

- `docs/api_contract.md` 已是本次 Coding 的 contract-first 输入，不在 Coding 阶段改协议。
- 不生成 `api/openapi.yaml`。
- 完成后更新 `docs/changes.md`、self review 和验证证据。

Confirmation:

- 用户已确认执行方案并要求开始代码开发。
- `new-api` 主仓已 `git pull --ff-only`，本 worktree 基于 `main` 创建。

Implementation:

- 新增 `controller.SeedanceNativeTaskCreate`，在 handler 内解析 Seedance native body，转换为内部 `relay/common.TaskSubmitReq`，保留 native-only 字段到 metadata，并改写为 `/v1/video/generations` task request。
- native create / get / list 鉴权在 `controller/seedance_native.go` 内复用现有 token / user cache / group / model limit 规则并输出 native error shell；不修改 `middleware/utils.go`。
- 新增 `POST /api/v3/contents/generations/tasks`、`GET /api/v3/contents/generations/tasks/:task_id`、`GET /api/v3/contents/generations/tasks`。
- 新增 `controller.SeedanceNativeTaskGet` / `SeedanceNativeTaskList` 和 native renderer / error shell；list 复用既有 `TaskGetAllUserTask` 时间窗口查询后在 handler 内过滤 task_ids、status、model 和可渲染平台。
- Doubao task adaptor create response 按 native mode 返回 public id，同时 `Task.Data` 写入 canonical request snapshot；OpenAI Video response 保持 public id wrapper。
- 2026-07-03 回改：native create 不再把 `duration` 写入内部 `TaskSubmitReq.Duration/Seconds`，Doubao 上游请求体只以 metadata 的 native `duration` 为准。
- 2026-07-03 回改：Doubao 上游 `requestPayload` 补 `priority`，metadata 中 `priority` 会透传到实际上游 JSON body。
- 2026-07-03 回改：Doubao adaptor 在 metadata 已包含原生 `content` 时保序透传全部 content，不删除多个 text，也不再只追加第一个 prompt；旧 OpenAI Video 路径无原生 `content` 时仍按 `images + prompt` 兼容组装。
- 2026-07-03 回改：为降低未来同步 new-api 上游代码的冲突面，`DoResponse` 仅前置新增 native response 分支并提前返回，默认 OpenAI Video response 原代码路径保持原样；`convertToRequestPayload` 仅在 `hasNativeContent` 时提前返回，其余原逻辑保持原样。
- 未新增转换 middleware，未修改 `model/task.go`、`middleware/utils.go`、`controller/relay.go`、`relay/common/relay_utils.go`；保持账务、预扣、结算、退款、轮询状态机和 DB schema 不变。

Verification:

- 2026-07-03 红灯验证：`GOCACHE=$PWD/.gocache go test ./relay/channel/task/doubao -run 'TestConvertToRequestPayloadKeepsNativeMetadataFields|TestConvertToRequestPayloadBuildsLegacyOpenAIVideoContentWithoutNativeMetadata' -count=1` 在实现前因 `requestPayload` 缺 `Priority` 编译失败。
- 2026-07-03 红灯验证：`GOCACHE=$PWD/.gocache go test ./controller -run 'TestSeedanceNativeBuildOpenAIRequestKeepsNativeFieldsInMetadata' -count=1` 在实现前因 native handler 仍设置 `TaskSubmitReq.Duration=5` 失败。
- 2026-07-03：`GOCACHE=$PWD/.gocache go test ./relay/channel/task/doubao -run 'TestConvertToRequestPayloadKeepsNativeMetadataFields|TestConvertToRequestPayloadBuildsLegacyOpenAIVideoContentWithoutNativeMetadata' -count=1`：通过。
- 2026-07-03：`GOCACHE=$PWD/.gocache go test ./controller -run 'TestSeedanceNativeBuildOpenAIRequestKeepsNativeFieldsInMetadata' -count=1`：通过。
- 2026-07-03：`GOCACHE=$PWD/.gocache go test ./relay/channel/task/doubao -count=1`：通过。
- 2026-07-03：`GOCACHE=$PWD/.gocache go test ./controller ./router ./relay/channel/task/doubao -count=1`：通过。
- 2026-07-03：`GOCACHE=$PWD/.gocache go test ./controller ./relay ./relay/channel/task/doubao ./model ./middleware ./router -count=1`：通过。
- 2026-07-03：`GOCACHE=$PWD/.gocache go test ./router ./controller ./middleware ./relay ./relay/channel ./relay/channel/task/doubao ./model ./service ./dto ./setting/config ./setting/model_setting ./setting/operation_setting -count=1`：通过。
- 2026-07-03：`GOCACHE=$PWD/.gocache go vet ./controller ./relay ./relay/channel/task/doubao ./model ./middleware ./router`：通过。
- 2026-07-03：`codegraph sync && codegraph status`：同步 4 个变更文件，Index is up to date。
- 2026-07-03：`git diff -- middleware/utils.go controller/relay.go model/task.go relay/common/relay_utils.go controller/task.go controller/task_video.go controller/video.go`：无输出，确认用户点名 shared 文件和相邻 controller 文件无 diff。
- 2026-07-03 最小 diff 形态复验：`GOCACHE=$PWD/.gocache go test ./relay/channel/task/doubao -run 'TestConvertToRequestPayloadKeepsNativeMetadataFields|TestConvertToRequestPayloadBuildsLegacyOpenAIVideoContentWithoutNativeMetadata|TestDoResponseCanReturnNativeCreateBodyWithoutChangingTaskData|TestDoResponseKeepsOpenAIVideoBodyByDefault' -count=1`：通过。
- 2026-07-03 最小 diff 形态复验：`GOCACHE=$PWD/.gocache go test ./relay/channel/task/doubao -count=1`：通过。
- 2026-07-03 最小 diff 形态复验：`GOCACHE=$PWD/.gocache go test ./controller ./router ./relay/channel/task/doubao -count=1`：通过。
- 2026-07-03 最小 diff 形态复验：`GOCACHE=$PWD/.gocache go test ./controller ./relay ./relay/channel/task/doubao ./model ./middleware ./router -count=1`：通过。
- 2026-07-03 最小 diff 形态复验：`GOCACHE=$PWD/.gocache go test ./router ./controller ./middleware ./relay ./relay/channel ./relay/channel/task/doubao ./model ./service ./dto ./setting/config ./setting/model_setting ./setting/operation_setting -count=1`：通过。
- 2026-07-03 最小 diff 形态复验：`GOCACHE=$PWD/.gocache go vet ./controller ./relay ./relay/channel/task/doubao ./model ./middleware ./router`：通过。
- 2026-07-03 最小 diff 形态复验：`codegraph sync && codegraph status`：同步 2 个变更文件，Index is up to date。
- `GOCACHE=$PWD/.gocache go test ./controller ./router ./relay/channel/task/doubao -count=1`：通过。
- `GOCACHE=$PWD/.gocache go test ./middleware ./model ./relay/common -count=1`：通过，确认回退后的 shared 包仍通过。
- `GOCACHE=$PWD/.gocache go test ./controller ./relay ./relay/channel/task/doubao ./model ./middleware ./router -count=1`：通过。
- `GOCACHE=$PWD/.gocache go test ./router ./controller ./middleware ./relay ./relay/channel ./relay/channel/task/doubao ./model ./service ./dto ./setting/config ./setting/model_setting ./setting/operation_setting -count=1`：通过。
- `GOCACHE=$PWD/.gocache go vet ./controller ./relay ./relay/channel/task/doubao ./model ./middleware ./router`：通过。
- `git diff --check`：通过。
- 新增文件 `git diff --no-index --check /dev/null <file>`：通过，覆盖 `controller/seedance_native.go`、`controller/seedance_native_test.go`、`relay/channel/task/doubao/adaptor_native_test.go`、`router/video_router_test.go`、本计划文件。
- `codegraph sync`：同步 1 个变更文件；`codegraph status`：Index is up to date；`.codegraph` 未被 Git 跟踪。
- `git diff -- middleware/utils.go controller/relay.go model/task.go relay/common/relay_utils.go`：无输出，确认用户点名 shared 文件无 diff。
- `GOCACHE=$PWD/.gocache go test . -run '^$'`：未通过，根包失败原因为 `main.go:44:12: pattern web/classic/dist: no matching files found`，当前 worktree 缺前端嵌入产物，真实 HTTP 启动验证阻塞。

Self Review:

- 结论：`通过`。
- 事实依据 / 证据充分性检查：实现依据已通过的 `docs/api_contract.md`、技术方案和用户 2026-07-03 明确要求；核心行为由 controller、router、Doubao adaptor 单测、相关包测试、`go vet`、`git diff --check` 和 CodeGraph status 验证。
- 变更边界完整性检查：仅新增 native bridge handler、renderer、handler 内过滤、Doubao taskData 快照和 Doubao 上游请求体兼容修正；未新增转换 middleware；未修改 `middleware/utils.go`、`controller/relay.go`、`model/task.go`、`relay/common/relay_utils.go`、账务、轮询、DB schema、delete/cancel。
- 负向关联影响检查：OpenAI Video create / fetch 行为由 `DoResponseKeepsOpenAIVideoBodyByDefault`、`ConvertToOpenAIVideoReadsNativeCanonicalTaskData`、`TestConvertToRequestPayloadBuildsLegacyOpenAIVideoContentWithoutNativeMetadata` 和相关包回归测试覆盖。
- 上下游依赖检查：create 在 native handler 内复用 token / group / channel selection 规则和 relay task submit 函数；get/list 仅依赖本地 task public id 与当前 user。
- 新增风险 / 问题 / 矛盾点 / 模糊项 / 重叠项检查：真实 HTTP 接口测试受根包缺 `web/classic/dist` 阻塞；C1 官方字段和值域实测仍按 contract 待后续 owner / 测试渠道确认。
- 问题处置状态检查：发现 `filter.status=cancelled` 不能映射内部 `FAILURE`，已修复为返回空结果并补测试；发现 native 鉴权失败外壳 401/类型不一致，已修复为 403 native shell 并补测试；发现业务代码直接使用 `encoding/json`，已改为项目 `common` JSON wrapper；2026-07-03 发现 native duration 双来源、priority 上游丢失、content 被重建，已修复并补红绿测试。
- 问题清单与状态：真实 HTTP 接口测试阻塞为 `接受风险`，已记录阻塞命令与原因；C1 实测为 `待 owner 确认`，不阻塞本次代码实现；native duration / priority / content 三项回改为 `已修复`。

Rollback:

- 回滚本次分支即可移除 `/api/v3/contents/generations/tasks` 路由、native handler / renderer 和 Doubao create response 快照改动。
- 因未改 DB schema、账务、轮询和 delete/cancel 状态机，回滚不需要数据迁移。

Status: completed
