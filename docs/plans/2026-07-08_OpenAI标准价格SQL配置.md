# OpenAI 标准计费 SQL 配置计划

Date: 2026-07-08

Task: 根据 OpenAI 官方价格页生成 new-api 后台标准计费 SQL。

Scope:
- 只生成 SQL 文件，不连接数据库，不执行写入。
- 只配置 OpenAI Standard token 价格。
- 不启用 Batch / Flex / Priority / service_tier 自动分层。
- 不修改渠道 allow_service_tier 开关。

Evidence:
- 官方价格页：https://developers.openai.com/api/docs/pricing
- 本地抓取文件：/private/tmp/openai-pricing.html
- new-api 配置表依据：model/option.go 中 Option key/value 主键模型。
- new-api 分层计费依据：setting/billing_setting/tiered_billing.go 中 billing_setting.billing_mode / billing_setting.billing_expr。
- 表达式单位依据：pkg/billingexpr/expr.md 中 $ / 1M tokens 说明。

Result:
- 新增 SQL：docs/sql/openai_standard_pricing_tiered_expr.sql
- 写入方式：MySQL INSERT ... ON DUPLICATE KEY UPDATE + JSON_MERGE_PATCH 合并写入。
- 写入 key：billing_setting.billing_mode、billing_setting.billing_expr。
- 纳入模型数：76。
- gpt-5.5、gpt-5.5-pro、gpt-5.4、gpt-5.4-pro 已按官方短 / 长上下文列生成 len < 272000 条件表达式。

Excluded:
- Batch / Flex / Priority 非本次标准计费范围。
- Regional processing uplift、AWS Bedrock 差异价格未纳入。
- fine-tuning training hour、Sora per-second、tool call/storage、free moderation、per-minute/per-character 价格未纳入。
- 部分 realtime / image 多模态行因不同模态 cached input 单价不同，new-api 当前单一 cr 变量不能精确表达，未纳入。

Verification:
- 已解析官方页面 Astro/SSR 数据并生成 JSON 配置。
- 已完成：SQL 文件 JSON 解析和表达式编译验证。
- `node` JSON 校验：`billing_mode 76`、`billing_expr 76`，并确认 `gpt-realtime-translate`、`gpt-realtime-whisper`、`tts-1`、`tts-1-hd` 未混入 token 计费 SQL。
- `GOCACHE=/private/tmp/openai-pricing-go-build-cache go test ./pkg/billingexpr -tags openai_pricing_check -run TestGeneratedOpenAIStandardPricingExpressions -count=1`：通过，76 条表达式均可由 `billingexpr.RunExpr` 编译运行且结果非负。
- `GOCACHE=/private/tmp/openai-pricing-go-build-cache go test ./pkg/billingexpr -run '^$' -count=1`：通过。

Risk:
- SQL 使用 MySQL JSON_MERGE_PATCH；若目标库不是 MySQL 或版本过低，需要改为对应数据库 JSON merge/upsert 语法。
- Search 相关模型只配置 token 价格；工具调用 / storage 等非 token 费用未写入该表达式。
- OpenAI 价格会变化；后续执行前应重新抓取官方页生成。

Rollback:
- 执行前备份 options 表中 billing_setting.billing_mode 和 billing_setting.billing_expr 两行。
- 回滚时将这两行 value 恢复为备份值，或删除本次新增模型 key 后重启 / 等待配置同步。

SelfReview:
- 结论：通过。
- 事实依据 / 证据充分性检查：官方页、本地源码和生成 SQL 均已留痕；SQL 中两段 JSON 均可解析为 76 个模型配置。
- 变更边界完整性检查：仅新增 SQL 和计划文档，未连接数据库，未修改运行时代码。
- 负向关联影响检查：Batch / Flex / Priority、service_tier 透传开关、非 token 价格、模态缓存单价无法精确表达的模型均未强行写入。
- 上下游依赖检查：SQL 依赖 MySQL `JSON_MERGE_PATCH`；若目标库非 MySQL，需要改写 upsert / JSON merge 语法。
- 问题处置状态检查：已发现并移除 per-minute / per-character 行误纳入问题，当前无未闭环问题。
