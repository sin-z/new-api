# 2026-07-07 邮箱验证码注册登录一体化计划

Date: 2026-07-07

Task: 邮箱验证码注册 / 登录一体化

Branch: `fix_codex/email-code-register-login`

Scope:
- `new-api` 邮箱验证码登录接口契约和实现。
- 用户模型 `username` / `display_name` 长度上限调整。
- 邮箱验证码登录控制器测试。

Affected:
- `controller/user.go`
- `controller/email_login_test.go`
- `model/user.go`
- `docs/api_contract.md`
- `docs/plan_index.md`
- `docs/changes.md`

Risks:
- 自动注册会扩大匿名邮箱验证码接口的账号创建能力，必须受 `RegisterEnabled` 控制。
- `username` / `display_name` 长度上限扩大到 254，需通过 Go 测试和现有用户更新路径验证。
- 登录审计依赖 `model.LOG_DB`，测试夹具需覆盖日志表，避免隐藏真实登录路径问题。

Acceptance:
- 已注册启用邮箱维持原登录行为。
- 未注册邮箱在 `RegisterEnabled=true` 时可通过邮箱验证码自动创建普通用户并登录。
- 自动创建用户使用完整邮箱作为 `username`、`email`、`display_name`，随机内部密码不可由用户直接获知。
- `RegisterEnabled=false` 时，新邮箱请求验证码和验证码登录均失败。
- 禁用用户、软删除 / 已占用邮箱、非法邮箱和超长邮箱均失败。
- `EmailLoginPurpose` 与 `EmailVerificationPurpose` 继续隔离。

Docs:
- 更新 `docs/api_contract.md` 中邮箱验证码登录契约。
- 完成后更新 `docs/changes.md`。

Confirmation:
- 用户已选择方案 2：邮箱验证码注册 / 登录一体化。
- 用户确认新邮箱自动注册遵守 `RegisterEnabled`。
- 用户确认自动注册使用完整邮箱作为 `username`，并将 username 长度放宽到 254。
- 用户确认无密码邮箱注册不受 `PasswordRegisterEnabled` 约束。

Result:
- `GET /api/user/email_login/code` 已支持已注册启用用户发登录码、未注册邮箱在 `RegisterEnabled=true` 时发登录 / 注册码，发送阶段不创建用户。
- `POST /api/user/email_login` 已支持未注册邮箱验证码校验成功后自动创建普通启用用户并登录。
- 自动创建用户使用完整邮箱作为 `username`、`email`、`display_name`，内部密码随机生成，不受 `PasswordRegisterEnabled` 约束。
- 禁用用户、软删除用户、同名 username 占用、注册关闭和超长邮箱均返回失败。
- `UserNameMaxLength`、`User.Username`、`User.DisplayName`、`User.Email` 上限同步为 254。

Verification:
- `go test ./controller -run 'Test.*EmailLogin' -count=1`：通过，覆盖 handler 与 loopback HTTP 邮箱验证码自动注册登录链路。
- `go test ./common -run TestEmailLoginPurposeIsIsolatedFromRegistrationPurpose -count=1`：通过。
- `go test ./model -run User -count=1`：通过。
- `git diff --check`：通过。
- `go test ./... -run '^$'`：根包因仓内缺 `web/classic/dist` 和 `web/default/dist` 编译失败；其他已枚举包空跑通过。该阻塞与本次改动无关，已用 `controller` loopback HTTP 测试覆盖本次接口行为。

SelfReview:
- 结论：`通过`
- 事实依据 / 证据充分性检查：用户确认项、代码 diff、契约 diff 和上述命令结果可验证。
- 变更边界完整性检查：改动限定在邮箱验证码登录控制器、用户模型长度、接口契约、测试、计划索引和变更记录。
- 负向关联影响检查：已覆盖已存在启用用户登录、注册关闭、禁用用户、软删除 / username 冲突、验证码 purpose 隔离和 254/255 边界。
- 上下游依赖检查：SMTP 发送路径通过 fake SMTP 测试；登录审计通过 `LOG_DB` 测试夹具覆盖；本地完整服务启动受缺前端 dist 阻塞。
- 新增风险 / 问题 / 矛盾点 / 模糊项 / 重叠项检查：剩余风险为生产数据库字段物理长度如被外部迁移限制为小于 254，需要部署前核对；当前 GORM 模型未声明较短 varchar 长度。
- 问题处置状态检查：无未闭环代码问题；根包 dist 缺失为既有环境阻塞，已记录验证证据。

Status: completed
