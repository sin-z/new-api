# Important Findings

- 日期：2026-07-09
  场景：XRTokenArkVideo 任务查询协议适配
  发现内容：XRTokenArkVideo 的任务查询接口已改为 `GET /v1/videos/generations/{task_id}`；该查询返回的 `usage.completion_tokens` / `usage.total_tokens` 会写入 `relaycommon.TaskInfo`，任务完成结算可按 token 重算，Seedance native GET 响应也会保留 `usage`。
  依据来源：源码 `relay/channel/task/xrtokenarkvideo/adaptor.go`、`controller/seedance_native.go`；测试 `TestFetchTaskUsesVideoGenerationsPath`、`TestParseTaskResultMapsUsageAndTopLevelVideoURL`、`TestSeedanceNativeTaskGetRendersXRTokenUsageAndNativeFields`；验证命令 `go test ./relay/channel/task/xrtokenarkvideo -count=1`、`go test ./controller -run 'SeedanceNative|XRToken' -count=1`、`go test ./service ./relay/channel/task/xrtokenarkvideo ./relay/channel/task/doubao ./relay -count=1`。
  适用范围：后续维护 XRToken ARK video 渠道轮询、Seedance native 查询响应、视频任务 token 结算与使用日志。

- 日期：2026-07-09
  场景：Seedance native task 响应字段渲染
  发现内容：Seedance native task 渲染层兼容顶层 `video_url` 与 `content.video_url`；XRToken 查询响应中的 `last_frame_url`、`framespersecond` 会在上游返回时保留；新增 `draft=false` 与 `priority=0` 按零值省略。返回 `id` 仍为 public task id，不暴露上游 task id。
  依据来源：源码 `controller/seedance_native.go`；测试 `TestSeedanceNativeTaskGetRendersXRTokenUsageAndNativeFields`、`TestSeedanceNativeTaskGetReadsXRTokenTopLevelVideoURL`；验证命令 `go test ./controller -run 'SeedanceNative|XRToken' -count=1`、`go test ./controller -count=1`。
  适用范围：后续维护 `/api/v3/contents/generations/tasks/{id}`、Seedance native list/get、XRToken 与 Doubao 存量任务数据兼容。

- 日期：2026-07-08
  场景：修复 Seedance native 编译错误
  发现内容：`types.PriceData.OtherRatios` 已在 `fc1259f5 refactor(price): improve handling of other ratios in PriceData` 中由导出字段改为私有 `otherRatios` 加 `OtherRatios()` 快照方法；`controller/relay.go` 已同步改为方法调用，`controller/seedance_native.go` 漏改会导致 `go test ./controller` 和根包 `go build` 编译失败。修复后 Go 代码中不再存在 `PriceData.OtherRatios` 非方法调用。
  依据来源：源码 `types/price_data.go`、`controller/relay.go`、`controller/seedance_native.go`；提交 `fc1259f5`；验证命令 `go test ./controller -count=1 -timeout=60s`、`go build -o /tmp/token168-new-api-build-check .`、`rg -n "PriceData\\.OtherRatios(?!\\()" --pcre2 --glob '*.go' .`。
  适用范围：后续维护 `types.PriceData` 附加倍率快照、Seedance native 任务落库计费上下文、异步 task billing 计费重算。

- 日期：2026-07-08
  场景：历史品牌替换 ZZ123
  发现内容：仓库内历史品牌大小写变体残留扫描无命中；账户邮件模板测试 fixture 已使用 `ZZ123` 和 `https://www.zz123.ai`，历史本地品牌目录路径已改写为 `<workspace>` 占位。
  依据来源：本轮残留扫描无输出；测试 `go test ./controller -run 'TestSignInCodeEmailUsesEnglishTemplateAndNumericCode|TestEmailVerificationUsesEnglishTemplateAndNumericCode|TestPasswordResetEmailUsesEnglishTemplateAndKeepsResetLink' -count=1 -timeout 60s` 通过；`git diff --check` 通过。
  适用范围：后续维护账户邮件模板测试、品牌示例域名、计划文档和技术方案历史执行记录。

- 日期：2026-07-08
  场景：账户邮件英文化与邮箱验证码数字化
  发现内容：登录验证码和邮箱验证验证码已改为 `crypto/rand` 生成的 6 位纯数字；密码重置链接 token 和重置后的随机密码仍沿用 `GenerateVerificationCode`，未降级为纯数字。登录验证码、邮箱验证和密码重置邮件主题 / 正文已改为英文，邮件品牌名继续来自 `SystemName`。
  依据来源：源码 `common/verification.go`、`controller/user.go`、`controller/misc.go`；测试 `common/verification_email_login_test.go`、`controller/email_template_test.go`；验证命令 `go test ./common -run 'Test.*Verification|TestSendEmail|TestNewSMTPClient|TestSMTPPlainAuth' -count=1`、`go test ./controller -run 'Test.*Email.*|TestSendEmailLoginCode|TestSendPasswordResetEmail' -count=1`。
  适用范围：后续维护邮箱验证码、账户邮件模板、SMTP 发信内容和 ZZ123 品牌名配置。

- 日期：2026-07-07
  场景：邮箱验证码登录改为注册 / 登录一体化
  发现内容：`new-api` 的 `GET /api/user/email_login/code` 已支持未注册邮箱在 `RegisterEnabled=true` 且邮箱 / 同名 username 未被现有或软删除账号占用时发码，但发码阶段不创建用户；`POST /api/user/email_login` 在邮箱登录验证码校验成功后才自动创建普通启用用户并登录，自动创建用户的 `username`、`email`、`display_name` 均为完整邮箱，内部密码随机生成且不受 `PasswordRegisterEnabled` 约束。邮箱验证码 purpose 继续与注册验证码 purpose 隔离。
  依据来源：源码 `controller/user.go`、`model/user.go`、`docs/api_contract.md`；测试 `controller/email_login_test.go`、`common/verification_email_login_test.go`；验证命令 `go test ./controller -run 'Test.*EmailLogin' -count=1`、`go test ./common -run TestEmailLoginPurposeIsIsolatedFromRegistrationPurpose -count=1`、`go test ./model -run User -count=1`。
  适用范围：后续维护邮箱验证码登录、注册开关、无密码邮箱注册、用户模型长度、Console 登录文案和接口契约。

- 日期：2026-07-07
  场景：邮箱验证码登录发码运行配置诊断
  发现内容：`535 5.7.0 Invalid login or password` 是 `common.SendEmail` 调用 SMTP 认证时返回并由 `common.ApiError` 原样透出的错误；`REDIS_CONN_STRING` 未配置只会让 `InitRedisClient` 设置 `RedisEnabled=false` 并返回 nil，不是该 SMTP 认证失败的直接原因。当前邮箱验证码存储使用进程内 `verificationMap`，不读写 Redis；单实例可工作，多实例或无状态部署存在验证码不共享风险。
  依据来源：源码 `controller/user.go`、`common/email.go`、`common/redis.go`、`common/verification.go`、`model/option.go`；验证命令 `go test ./common -run 'TestSendEmail|TestNewSMTPClient|TestSMTPPlainAuth' -count=1`、`go test ./controller -run 'TestSendEmailLoginCode' -count=1`。
  适用范围：后续排查邮箱验证码发码失败、SMTP 配置、Redis 配置、多实例验证码一致性问题。

- 日期：2026-07-07
  场景：Seedance 2.0 native create 连通性修复
  发现内容：Seedance 2.0 上游不接受显式 `service_tier=default`；本地 native canonical data 仍需要保存 `service_tier=default` 以便 get/list 渲染。
  依据来源：远端 `testnapi.zz123.ai` 复测返回上游错误消息；本地 `convertToRequestPayload` 回归测试确认上游请求体省略 `service_tier`。
  适用范围：Seedance native create 转 Doubao / VolcEngine / XRToken ARK video 上游请求。

- 日期：2026-07-07
  场景：Seedance 2.0 native create 连通性修复
  发现内容：Seedance 2.0 `duration=1` 会被上游拒绝；本地 create handler 应先校验 `duration` 为缺省、`-1` 或 `4..15`。
  依据来源：本地 HTTP 验证 `duration=1` 返回 400 `InvalidParameter.InvalidValue`；`TestSeedanceNativeBuildOpenAIRequestRejectsInvalidDuration` 覆盖该规则。
  适用范围：`/api/v3/contents/generations/tasks` Seedance native create。

- 日期：2026-07-07
  场景：Seedance 2.0 task 响应解析
  发现内容：Doubao / XRToken ARK video 上游任务响应的 `created_at` / `updated_at` 可能返回 RFC3339 字符串，而不是 Unix 秒数；解析层需兼容 number、数字字符串和 RFC3339 字符串。
  依据来源：本地服务错误复现为 `json: cannot unmarshal string into Go struct field responseTask.created_at of type int64`；`TestParseTaskResultAcceptsStringTimestamps` 和 `TestDoResponseAcceptsStringTimestamps` 红绿验证。
  适用范围：Doubao task adaptor、XRToken ARK video task adaptor、Seedance native task renderer。

- 日期：2026-07-07
  场景：本地与远端环境差异
  发现内容：当前本地 `new-api` 修复后 create/get/list 已通过，但 `https://testnapi.zz123.ai` 仍返回旧的 `service_tier` 上游拒绝错误；远端需部署本地修复后再复测成功。
  依据来源：本地 30169 HTTP 验证 create/get/list 返回 200；远端 2026-07-07 复测 `duration=1` 返回 400 `fail_to_fetch_task` 且消息仍指向 `service_tier`。
  适用范围：Seedance native API 发布验证与测试环境问题判断。
