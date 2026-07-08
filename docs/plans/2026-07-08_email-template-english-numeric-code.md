# 2026-07-08 邮件英文化与纯数字验证码计划

Date: 2026-07-08

Task: 邮件英文化与纯数字验证码

Branch: `fix_codex/email-code-register-login`

Scope:
- 账户邮件模板：登录验证码、邮箱验证、密码重置。
- 用户手动输入的邮箱验证码生成逻辑。
- 相关单元测试、计划索引、变更记录和记忆沉淀。

Affected:
- `common/verification.go`
- `common/verification_email_login_test.go`
- `controller/user.go`
- `controller/misc.go`
- `controller/email_login_test.go`
- `controller/email_template_test.go`
- `docs/plan_index.md`
- `docs/changes.md`
- `memories/important-findings.md`

Risks:
- 验证码生成不能继续使用 UUID 前缀，否则可能出现英文字母。
- 密码重置链接 token 和重置后的随机密码不属于手动输入验证码，不能降级为 6 位数字。
- 邮件品牌名应继续来自 `SystemName`，避免在代码中硬编码 `Token168`。

Acceptance:
- 登录验证码和邮箱验证验证码均为 6 位纯数字。
- 登录验证码、邮箱验证、密码重置邮件主题和正文均为英文。
- 密码重置链接 token 仍保持现有长随机串语义。
- HTTP API、请求参数、响应结构、验证码有效期和校验逻辑不变。

Docs:
- 更新 `docs/changes.md`。
- 更新 `docs/plan_index.md`。
- 若产生稳定发现，更新 `memories/important-findings.md`。

Confirmation:
- 用户要求将验证码邮件内容改为英文。
- 用户确认覆盖全部账户邮件。
- 用户要求验证码改成纯数字。

Result:
- 新增 `common.GenerateNumericVerificationCode`，使用 `crypto/rand` 生成用户手动输入的纯数字验证码。
- `GET /api/user/email_login/code` 登录验证码改为 6 位纯数字，邮件主题和正文改为英文。
- `GET /api/verification` 邮箱验证验证码改为 6 位纯数字，邮件主题和正文改为英文。
- `GET /api/user/reset` 密码重置邮件主题和正文改为英文，重置链接 token 继续使用现有长随机串。
- 邮件品牌名继续使用 `common.SystemName`，不硬编码 `Token168`。

Verification:
- TDD 红灯：`go test ./common -run TestGenerateNumericVerificationCodeReturnsDigitsOnly -count=1` 失败于 `undefined: GenerateNumericVerificationCode`。
- TDD 红灯：`go test ./controller -run 'TestSignInCodeEmailUsesEnglishTemplateAndNumericCode|TestEmailVerificationUsesEnglishTemplateAndNumericCode|TestPasswordResetEmailUsesEnglishTemplateAndKeepsResetLink' -count=1` 失败于现有中文邮件主题。
- TDD 绿灯：`go test ./common -run 'TestGenerateNumericVerificationCodeReturnsDigitsOnly|TestEmailLoginPurposeIsIsolatedFromRegistrationPurpose' -count=1` 通过。
- TDD 绿灯：`go test ./controller -run 'TestSignInCodeEmailUsesEnglishTemplateAndNumericCode|TestEmailVerificationUsesEnglishTemplateAndNumericCode|TestPasswordResetEmailUsesEnglishTemplateAndKeepsResetLink' -count=1` 通过。
- 回归：`go test ./common -run 'Test.*Verification|TestSendEmail|TestNewSMTPClient|TestSMTPPlainAuth' -count=1` 通过。
- 回归：`go test ./controller -run 'Test.*Email.*|TestSendEmailLoginCode|TestSendPasswordResetEmail' -count=1` 通过。
- 静态检查：`git diff --check` 通过。

SelfReview:
- 结论：`通过`
- 事实依据 / 证据充分性检查：用户确认范围、源码 diff 和上述测试命令可验证。
- 变更边界完整性检查：改动限定在账户邮件模板、邮箱验证码生成、相关测试和过程文档。
- 负向关联影响检查：密码重置 token 与重置后随机密码仍使用 `GenerateVerificationCode`，未降级为纯数字。
- 上下游依赖检查：HTTP API、请求字段、响应结构、验证码有效期和校验逻辑未变；运行环境仍需将 `SystemName` 配置为 `Token168` 才会显示 Token168 品牌名。
- 新增风险 / 问题 / 矛盾点 / 模糊项 / 重叠项检查：6 位数字验证码空间较 UUID 前缀更小，但符合用户手动输入体验；现有接口限流策略未在本次范围内调整。
- 问题处置状态检查：未发现未闭环问题。

Rollback:
- 回滚 `common/verification.go`、`controller/user.go`、`controller/misc.go` 及新增/修改测试和文档即可恢复原中文模板与 UUID 前缀验证码行为。

Status: completed
