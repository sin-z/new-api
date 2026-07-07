# Seedance 2.0 原生 API 连通性修复计划

Date: 2026-07-07

Task: Seedance 2.0 原生 API 连通性修复

Branch: `main`

Scope:

- 修复 Seedance native create 请求向 Doubao / VolcEngine 上游透传不被支持的默认 `service_tier`。
- 补齐 Seedance native create 的 `duration` 入参校验，拒绝上游不接受的过短时长。
- 修复 Doubao task 上游响应中 `created_at` / `updated_at` 可能返回 RFC3339 字符串导致本地反序列化失败的问题。
- 使用本地 `new-api` 服务复测 `/api/v3/contents/generations/tasks` create / get / list 主链路。
- 远端 `testnapi.token168.ai` 仅在代码部署到该环境后才能体现本地修复；本计划记录远端复测事实，不把未部署的本地修改写成远端已生效。

Affected:

- `common/flexible_unix_time.go`
- `controller/seedance_native.go`
- `controller/seedance_native_test.go`
- `relay/channel/task/doubao/adaptor.go`
- `relay/channel/task/doubao/adaptor_native_test.go`
- `relay/channel/task/xrtokenarkvideo/adaptor.go`
- `relay/channel/task/xrtokenarkvideo/adaptor_test.go`
- `docs/plans/2026-07-07_Seedance2原生API连通性修复.md`
- `memories/important-findings.md`

Risks:

- 上游响应时间字段存在 number、数字字符串、RFC3339 字符串多形态，兼容解析必须不破坏既有 number 形态。
- `service_tier=default` 仍需保留在本地 native canonical data 中，不能继续透传给上游。
- 本地服务启动依赖指定环境文件和前端 embed 占位产物，验证过程不能泄漏敏感环境配置。
- 远端测试环境未部署本地代码时，远端 curl 仍会返回旧错误。

Acceptance:

- 用户给出的 native create 请求改为 1 秒测试时，本地 handler 返回明确 400 duration 校验错误，避免继续消耗上游。
- 用户给出的合法时长 native create 请求在本地不再出现 `service_tier` 上游拒绝错误。
- 上游 create 响应包含 RFC3339 `created_at` / `updated_at` 字符串时，`DoResponse` 不再返回 `unmarshal_response_body_failed`。
- `go test ./controller ./router ./relay/channel/task/doubao -count=1` 通过。
- 本地服务 HTTP create / get / list 验证完成并记录状态码、响应结构和任务 id 形态，不记录密钥。

Implementation:

- 在 `validateSeedanceNativeCreateRequest` 中约束 Seedance 2.0 `duration` 为缺省、`-1` 或 `4..15`。
- 在 `convertToRequestPayload` 中将本地默认 `service_tier=default` 清空，依赖 `omitempty` 从上游请求体中省略。
- 新增 `common.FlexibleUnixTime`，支持 Unix 秒数、数字字符串和 RFC3339 字符串。
- Doubao 与 XRToken/ARK video task response 均改用 `common.FlexibleUnixTime` 解析 `created_at` / `updated_at`。
- XRToken/ARK video 在 Seedance native create 模式下返回 public task id，并复用 Doubao canonical Task.Data 快照构建。
- Seedance native get/list 将 XRToken/ARK video 纳入可渲染平台，并支持读取顶层 `video_url`。
- 补齐回归测试，先验证红灯，再实现最小修复。

Verification:

- 红灯验证：
  - `go test ./relay/channel/task/doubao -run 'TestParseTaskResultAcceptsStringTimestamps|TestDoResponseAcceptsNativeCreateStringTimestamps' -count=1`：修复前 `TestParseTaskResultAcceptsStringTimestamps` 失败，错误为 `json: cannot unmarshal string into Go struct field responseTask.created_at of type int64`。
  - `go test ./relay/channel/task/xrtokenarkvideo -run 'TestDoResponseAcceptsStringTimestamps|TestParseTaskResultAcceptsStringTimestamps' -count=1`：修复前 create 和 parse 均因 `responseTask.created_at` 字符串反序列化失败。
- 绿灯验证：
  - `go test ./common ./controller ./router ./relay/channel/task/doubao ./relay/channel/task/xrtokenarkvideo -count=1`：通过。
  - `go vet ./controller ./router ./relay/channel/task/doubao ./relay/channel/task/xrtokenarkvideo`：通过。
  - `git diff --check`：通过。
- 本地服务验证：
  - 使用 `PORT=30169 NODE_TYPE=slave UPDATE_TASK=false MEMORY_CACHE_ENABLED=false GIN_MODE=release go run .` 启动当前工作树服务；`GET /api/status` 返回 200。
  - 本地 `POST /api/v3/contents/generations/tasks`，`duration=1`：返回 400，错误码 `InvalidParameter.InvalidValue`。
  - 本地 `POST /api/v3/contents/generations/tasks`，`duration=4`：返回 200，响应形态 `{"id":"task_..."}`。
  - 本地 `GET /api/v3/contents/generations/tasks/{task_id}`：返回 200，包含 public task id、`status=queued`、`ratio=16:9`、`duration=4`、`service_tier=default`。
  - 本地 `GET /api/v3/contents/generations/tasks?page_num=1&page_size=5`：返回 200，列表包含刚创建的 public task id。
- 远端复测事实：
  - 对 `https://testnapi.token168.ai/api/v3/contents/generations/tasks` 使用 `duration=1` 复测，返回 400，错误码 `fail_to_fetch_task`，错误消息仍为上游拒绝 `service_tier`。
  - 结论：远端当前行为仍是旧错误；本地修复尚未在该远端环境体现。
- 其他验证：
  - `go vet ./common ./controller ./router ./relay/channel/task/doubao ./relay/channel/task/xrtokenarkvideo` 未作为通过项，原因是 `./common` 中存在既有 vet 问题：`common/custom-event.go` 复制 `sync.Mutex`、`common/email_test.go` IPv6 地址格式；这些文件本次未修改。
  - 敏感扫描未发现真实密钥写入本次触达文件；仅命中测试用例中的假值 `sk-test`。
  - 本次启动的 30169 本地服务已停止。

Self Review:

- 结论：`通过`。
- 事实依据 / 证据充分性检查：根因由远端旧错误、本地 HTTP 500 错误、红灯单测和代码路径共同确认；修复后由定向测试、相关包单测、本地真实 HTTP create/get/list、远端复测和敏感扫描验证。
- 变更边界完整性检查：改动限定在 Seedance native handler / renderer、Doubao task adaptor、XRToken/ARK video adaptor、共享 JSON 时间解析类型和对应测试；未改数据库 schema、计费模型、轮询状态机、路由表和环境配置。
- 负向关联影响检查：Doubao OpenAI video 默认 response、XRToken/ARK OpenAI video response、ParseTaskResult、native get/list 均有回归测试；`go test` 覆盖相关包通过。
- 上下游依赖检查：上游不接受显式 `service_tier=default`，本地只保留 canonical data 中的 default；XRToken/ARK video 复用 Doubao native 请求快照，避免 create 后 get/list 数据格式不一致。
- 新增风险 / 问题 / 矛盾点 / 模糊项 / 重叠项检查：远端环境仍返回旧错误，需部署本地代码后再以远端成功作为验收；`./common` 包存在非本次引入的 vet 问题，已记录不纳入本次修复范围。
- 问题处置状态检查：`service_tier` 上游拒绝、`duration=1` 消耗风险、`created_at` 字符串解析失败、XRToken/ARK native create/get/list 兼容问题均已修复并补测试；远端未体现修复为 `待部署验证`。
- 问题清单与状态：
  - `service_tier=default` 透传上游：`已修复`。
  - Seedance 2.0 `duration=1` 请求继续打上游：`已修复`。
  - RFC3339 `created_at` / `updated_at` 反序列化失败：`已修复`。
  - XRToken/ARK video native create 响应与 Task.Data 格式不一致：`已修复`。
  - 远端 `testnapi.token168.ai` 仍返回旧错误：`待部署验证`。

Rollback:

- 回滚上述代码和测试文件即可恢复旧行为。
- 本次不改数据库 schema、计费模型、轮询状态机或配置文件，无需数据迁移。

Status: completed
