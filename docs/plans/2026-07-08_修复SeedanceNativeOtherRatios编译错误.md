# 修复 Seedance Native OtherRatios 编译错误

Date: 2026-07-08

Task: 修复 Seedance Native OtherRatios 编译错误

Branch: `fix_codex/seedance_native_other_ratios_compile`

Scope:

- 修复 `controller/seedance_native.go` 中 `relayInfo.PriceData.OtherRatios` 的旧字段访问。
- 保持 `controller/relay.go` 已使用的 `relayInfo.PriceData.OtherRatios()` 快照方法口径一致。
- 不改 HTTP API、数据库、配置、计费语义或 `types.PriceData` 接口。

Affected:

- `controller/seedance_native.go`
- `docs/plans/2026-07-08_修复SeedanceNativeOtherRatios编译错误.md`
- `docs/plan_index.md`
- `docs/changes.md`
- `memories/important-findings.md`

Risks:

- 若只修复 controller 编译，不复查全仓旧字段访问，可能遗漏其他同类调用点。
- `go build .` 依赖 ignored 前端 `dist` embed 产物；当前 worktree 使用最小 ignored 占位文件满足编译门禁，不纳入 Git。

Acceptance:

- `go test ./controller -count=1` 通过。
- `go test ./service -run 'TestPriceDataOtherRatiosFilterAndSnapshot|TestPriceDataReplaceAndApplyOtherRatios|TestTaskBillingContextPriceDataFiltersMultiplier' -count=1` 通过。
- `go build -o /tmp/token168-new-api-build-check .` 通过。
- `git diff --check` 通过。
- `rg -n "PriceData\\.OtherRatios(?!\\()" --pcre2 --glob '*.go' .` 不再命中 Go 代码旧字段访问。

Implementation:

- 将 `controller/seedance_native.go` 的 `OtherRatios: relayInfo.PriceData.OtherRatios` 改为 `OtherRatios: relayInfo.PriceData.OtherRatios()`。
- 不新增测试；既有 `service/task_billing_test.go` 已覆盖 `OtherRatios()` 过滤和快照语义，本次编译回归由 controller 编译和根包构建验证。

Verification:

- 红灯：
  - `go test ./controller`：失败，错误为 `controller/seedance_native.go:336:21: cannot use relayInfo.PriceData.OtherRatios (value of type func() map[string]float64) as map[string]float64 value in struct literal`。
  - `go build -o /tmp/token168-new-api-build-check .`：在补齐 ignored dist 占位后失败，错误同上。
- 绿灯：待执行。
  - `go test ./controller -count=1 -timeout=60s`：通过。
  - `go test ./service -run 'TestPriceDataOtherRatiosFilterAndSnapshot|TestPriceDataReplaceAndApplyOtherRatios|TestTaskBillingContextPriceDataFiltersMultiplier' -count=1 -timeout=60s`：通过。
  - `go build -o /tmp/token168-new-api-build-check .`：通过。
  - `git diff --check`：通过。
  - `rg -n "PriceData\\.OtherRatios(?!\\()" --pcre2 --glob '*.go' .`：无输出。

Self Review:

- 结论：`通过`。
- 事实依据 / 证据充分性检查：编译错误由 `go test ./controller` 与根包 `go build` 红灯复现；根因由 `fc1259f5`、`types/price_data.go`、`controller/relay.go` 和 `controller/seedance_native.go` 对比确认。
- 变更边界完整性检查：代码改动仅一行，限定在 Seedance native 任务计费上下文快照；未改 HTTP API、配置、数据库、计费状态机或 `PriceData` 实现。
- 负向关联影响检查：`controller` 包测试、相关 `service` 计费快照测试、根包构建和 Go 代码旧字段访问扫描均通过；`CodeGraph status` 为 `Index is up to date`。
- 上下游依赖检查：`TaskBillingContext.OtherRatios` 仍接收 `map[string]float64`，`PriceData.OtherRatios()` 返回过滤后的快照，和 `controller/relay.go` 既有口径一致。
- 新增风险 / 问题 / 矛盾点 / 模糊项 / 重叠项检查：无新增接口、配置、数据迁移或运行时行为风险；根包构建依赖 ignored 前端 dist，本 worktree 使用最小 ignored 占位，仅用于编译验证，不纳入提交。
- 问题处置状态检查：目标编译错误 `已修复`；未发现未闭环问题。

Rollback:

- 回滚 `controller/seedance_native.go`、本计划、计划索引、变更记录和记忆记录即可恢复本次修改前状态。
- 本次不改数据库 schema、运行配置、HTTP 契约或计费状态机，无需数据迁移。

Status: completed
