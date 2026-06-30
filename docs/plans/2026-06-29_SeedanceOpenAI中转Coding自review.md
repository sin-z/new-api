# Seedance OpenAI 中转 new-api Coding 自 review

## Date

2026-06-29

## 自 review 结论

- 结论：`通过`
- 事实依据 / 证据充分性检查：通过。实现输入来自 token168 root 上游 spec `../../docs/plan/2026-06-29-seedance-bridge-next-S1-4-seedance-bridge-coding-01KW99.md`、S1.3 技术方案 `../../docs/tech-design/token-gateway/seedance-2-native-openai-bridge-server-tech-design.md`、本仓 contract `docs/api_contract.md` 和任务线程 #45 @arch Phase 1 复审通过结论。
- 变更边界完整性检查：通过。实际触达 `controller/seedance_native.go`、`router/video-router.go`、`middleware/distributor.go`、`model/task.go`、`relay/channel/task/doubao/adaptor.go` 及对应测试 / 文档；未触达核心账务、预扣、提交、轮询、结算状态机文件。
- 负向关联影响检查：通过。OpenAI Video create / get 仍保留原有 wrapper 路径；Doubao adaptor 仅在 `seedance_native_response` context flag 为 true 时返回 native create response 与 canonical task data。
- 上下游依赖检查：通过。create 复用 `TokenAuth()` / `Distribute()` / `RelayTask(c)`；get/list 复用 `TokenAuth()` 和现有用户维度 task 查询，不挂 `Distribute()`，避免可选 `filter.model` 缺省时被 token model limit 误拦截；canonical `Task.Data` 保持 Doubao `responseTask` 顶层字段可读，OpenAI wrapper 可消费同一数据。
- 新增风险 / 问题 / 矛盾点 / 模糊项 / 重叠项检查：无阻塞项。真实 HTTP API Testing 未在本任务执行，按 spec 由后续 API Testing 任务使用已准备的用例包承接；本轮完成仓内 Go 测试、静态检查和路由 / handler / adaptor 单元级覆盖。
- 问题处置状态检查：Review 发现的 6 项问题均已回改，状态均为 `已修复`：业务代码直接调用 `encoding/json`、list 固定 500 条窗口导致 total / 分页不符合契约、分页非法参数未 400、native error code / HTTP status 与契约不一致、GET/list 挂 `Distribute()` 导致可选 `filter.model` 场景误拦截、HTTP handler 缺中文职责注释。二次复审发现的 `filter.status` 非法值落入 500 问题已修复为 native 400 `InvalidParameter.InvalidValue` / `BadRequest`。

## 服务端专项检查

- 契约一致性：通过。只实现 Seedance native create / get / list；未生成 OpenAPI；未修改 `docs/api_contract.md`。
- 鉴权 / 分发复用：通过。create 使用 `TokenAuth()` 与 `Distribute()`；get/list 仅使用 `TokenAuth()`，符合 S1.3 技术方案 get/list 鉴权边界，并避免 list 缺省 `filter.model` 时误触发模型限制校验。
- public id / upstream id 隔离：通过。create client response 使用 `info.PublicTaskID`；upstream id 仅作为 `DoResponse` 返回值进入现有私有存储链路；native / OpenAI renderer 不输出 upstream id。
- native-only 字段保真：通过。native create 的 `content`、`duration`、`ratio`、`resolution`、`generate_audio`、`watermark`、`return_last_frame`、`execution_expires_after`、`priority`、`service_tier` 写入 metadata / canonical `request`。
- list 查询事实源：通过。list 使用 `model.TaskGetAllUserTask` 包装现有用户维度 task 查询；新增 `Platforms` DAO 过滤在分页前限定 DoubaoVideo / VolcEngine native 可渲染平台，分批扫描最近 7 天符合 DB 条件的任务后再做 `filter.model` renderer 过滤和分页，避免固定 500 条窗口冒充 total。
- 排除范围：通过。未新增 DELETE 路由；未实现 cancel / delete；未新增内部 `CANCELLED` / `EXPIRED` / `DELETED`；未修改核心账务、预扣、提交、轮询、结算状态机。

## 验证证据

- `go test ./controller ./middleware ./relay/channel/task/doubao ./model ./router -count=1`：通过。
- `go vet ./controller ./middleware ./relay/channel/task/doubao ./model ./router`：通过，无输出。
- `git diff --check -- docs/changes.md docs/plan_index.md docs/plans/2026-06-29_SeedanceOpenAI中转Coding.md docs/plans/2026-06-29_SeedanceOpenAI中转Coding自review.md controller/seedance_native.go controller/seedance_native_test.go middleware/distributor.go middleware/seedance_native_test.go model/task.go model/task_cas_test.go relay/channel/task/doubao/adaptor.go relay/channel/task/doubao/adaptor_test.go router/video-router.go router/seedance_native_router_test.go`：通过，无输出。
- `python3 ../../harness-engineering/tools/harness_env.py doctor --profile bootstrap`：`bootstrap: pass`。
- `codegraph sync && codegraph status`：状态 `Index is up to date`。
- `git diff -- . | rg -n "DELETE|CANCELLED|EXPIRED|DELETED|PreConsumeBilling|RefundTaskQuota|RecalculateTaskQuota|RecalculateTaskQuotaByTokens|UpdateVideoTasks|service/task_billing|service/task_polling"`：只命中文档中明确排除范围，无代码实现命中。

## 风险与回滚

- 风险：`filter.model` 仍依赖读取任务属性 / canonical `Task.Data` 后在内存中过滤，避免引入跨 DB JSON 查询；最近 7 天内极大任务量会分批扫描，存在延迟 / 内存风险。后续如有性能压力，应在 contract / 技术方案阶段评估 DAO model filter、count 优化、模型字段索引或冗余字段。
- 回滚：回滚本分支即可移除新增 native 路由、handler、middleware 分支、task id 列表过滤和 Doubao native response 分支；不会影响核心账务、轮询或结算状态机。
