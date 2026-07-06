# Seedance 2.0 分辨率 USD 价格表调整

Date: 2026-07-06

Task: 将 Doubao Seedance 2.0 分辨率价格表从人民币口径改为 BytePlus 海外官方 USD 口径。

Branch: `fix_codex/seedance_usd_pricing`

Scope:

- 修改 `relay/channel/task/doubao/constants.go` 中 Seedance 2.0 / fast 的分辨率与视频输入价格表。
- 新增 `relay/channel/task/doubao/constants_test.go`，覆盖价格倍率计算。
- 更新 `docs/plan_index.md`、`docs/changes.md` 记录本次变更。

Affected:

- `doubao-seedance-2-0-260128` 的 `video_input` OtherRatio。
- `doubao-seedance-2-0-fast-260128` 的 `video_input` OtherRatio。
- `relay/channel/task/xrtokenarkvideo` 通过复用 Doubao adaptor 间接受影响。

Facts:

- BytePlus 官方文档 `https://docs.byteplus.com/en/docs/ModelArk/1544106`，页面最后更新时间为 `2026-07-02 11:56:43`。
- 官方 `Video generation models / Pricing` 表中 `dreamina-seedance-2-0-260128` 在线推理价格为：
  - 480p / 720p，输入不含视频：7.0 USD / M tokens。
  - 480p / 720p，输入含视频：4.3 USD / M tokens。
  - 1080p，输入不含视频：7.7 USD / M tokens。
  - 1080p，输入含视频：4.7 USD / M tokens。
  - 4k，输入不含视频：4.0 USD / M tokens。
  - 4k，输入含视频：2.4 USD / M tokens。
- 官方 `dreamina-seedance-2-0-fast-260128` 在线推理价格为：
  - 输入不含视频：5.6 USD / M tokens。
  - 输入含视频：3.3 USD / M tokens。
  - 官方说明 1080p output is not supported。
- `GetVideoInputRatio` 返回实际单价 / 基准价；基准价是 480p / 720p 且输入不含视频。

Risks:

- 若后台基础模型价格仍按人民币配置，会导致结算口径混用；本任务前置决策为展示、结算、计费统一使用 USD。
- fast 模型未配置 1080p / 4k 价格时仍返回 1.0，保持现有上游自行报错或按基准倍率估算的行为。

Acceptance:

- `doubao-seedance-2-0-260128` 的倍率按 7.0 为基准计算。
- `doubao-seedance-2-0-fast-260128` 的倍率按 5.6 为基准计算。
- 未知模型返回 `ok=false`。
- fast 1080p / 4k 缺省组合返回 `1.0, true`，保持原行为。
- 相关 Go 单测通过。

Verification:

- 红灯：旧人民币价格表下执行 `go test ./relay/channel/task/doubao -count=1`，新增 USD 倍率测试按预期失败。
- 绿灯：`go test ./relay/channel/task/doubao -count=1` 通过。
- 复用包回归：`go test ./relay/channel/task/xrtokenarkvideo ./relay/channel/task/doubao -count=1` 通过。
- 静态检查：`go vet ./relay/channel/task/doubao ./relay/channel/task/xrtokenarkvideo` 通过。
- Diff 检查：`git diff --check` 通过。
- CodeGraph：`codegraph sync` 后 `codegraph status` 显示索引最新。

SelfReview:

- 结论：通过。
- 事实依据：价格来自 BytePlus 官方页面和当前源码，调用链由 CodeGraph 与源码交叉验证。
- 变更边界：仅修改 Doubao 价格常量、对应单测与计划 / 变更留痕。
- 负向影响：后台基础价需继续按 USD 配置；fast 1080p / 4k 缺省回退保持原行为。
- 问题处置：自 review 发现旧注释把基准单价误写为 `ModelRatio`，已修复为“模型基准单价”。

Rollback:

- 回滚本分支中 `relay/channel/task/doubao/constants.go`、`relay/channel/task/doubao/constants_test.go`、`docs/plans/2026-07-06_Seedance2分辨率USD价格表调整.md`、`docs/plan_index.md`、`docs/changes.md` 的本次改动即可恢复旧价格表。

Docs:

- 更新本计划文件。
- 更新 `docs/plan_index.md`。
- 更新 `docs/changes.md`。

Confirmation:

- 用户已确认“修改为按 USD 的价格表”。

Status: completed
