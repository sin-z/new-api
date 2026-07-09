# Changes

## 2026-07-09

- 将 XRTokenArkVideo 任务查询接口从 `/v1/contents/generations/tasks/{task_id}` 切换到 `/v1/videos/generations/{task_id}`，创建接口保持不变。
- XRTokenArkVideo 任务查询成功时解析上游 `usage.completion_tokens` / `usage.total_tokens` 并写入 `TaskInfo`，用于任务完成后的 token 口径结算与使用日志。
- XRTokenArkVideo 结果 URL 兼容顶层 `video_url` 与 `content.video_url`，OpenAI Video wrapper 和 Seedance native 查询均可读取。
- Seedance native task 响应补充 XRToken 查询返回中的 `last_frame_url`、`framespersecond`、`draft`、`priority` 等字段映射；`draft=false` 与 `priority=0` 均按零值省略。
- 更新 Seedance native GET 契约示例和字段表，补齐 `framespersecond`、`usage` 与 XRToken 返回字段说明。

## 2026-07-08

- 修复 Seedance native 任务落库计费上下文中的 `PriceData.OtherRatios` 旧字段访问，改为调用 `OtherRatios()` 快照方法；`go test ./controller` 与根包 `go build` 不再因该类型错误失败。
- 将登录验证码、邮箱验证和密码重置三类账户邮件主题 / 正文改为英文；邮件品牌名继续来自 `SystemName`，不在代码中硬编码 `ZZ123`。
- 新增纯数字邮箱验证码生成能力，登录验证码和邮箱验证验证码改为 6 位数字；密码重置链接 token 和重置后随机密码继续使用原长随机串生成方式。
- 新增 SMTP 邮件内容测试，覆盖英文主题 / 正文、6 位数字验证码和密码重置链接保留。

## 2026-07-07

- 将 `GET /api/user/email_login/code` 扩展为邮箱登录 / 注册验证码发送：已存在启用用户沿用登录验证码；未注册邮箱在 `RegisterEnabled=true` 且未被现有或软删除账号占用时允许发码，发送阶段不创建用户。
- 将 `POST /api/user/email_login` 扩展为邮箱验证码登录 / 自动注册一体化：未注册邮箱验证码校验成功后自动创建普通启用用户，`username`、`email`、`display_name` 均使用完整邮箱，内部密码随机生成；无密码邮箱注册不受 `PasswordRegisterEnabled` 限制。
- 放宽 `UserNameMaxLength`、`User.Username`、`User.DisplayName`、`User.Email` 校验上限到 254，并在邮箱验证码登录地址校验中拒绝超过 254 字符的邮箱。
- 补齐 `controller/email_login_test.go` 测试夹具的 `LOG_DB`、`logs` 表、Redis 关闭和 fake SMTP，新增用例覆盖未注册发码不建用户、自动注册登录、loopback HTTP 登录、注册关闭、已存在用户不重复创建、禁用 / 软删除 / username 冲突、254/255 长度边界。
- 更新 `docs/api_contract.md` 邮箱验证码登录契约，移除“必须属于现有用户 / 不自动注册不存在邮箱”的旧口径。
- 补充邮箱登录运行配置诊断留痕：确认 `535 5.7.0 Invalid login or password` 来自 SMTP 认证失败链路，`REDIS_CONN_STRING` 缺失只会禁用 Redis 缓存，不是本次发码失败的直接原因。

## 2026-07-06

- 将 Doubao Seedance 2.0 / fast 分辨率价格表从人民币口径调整为 BytePlus 海外官方 USD / M tokens 口径：标准模型使用 7.0、4.3、7.7、4.7、4.0、2.4；fast 模型使用 5.6、3.3。
- 新增 `GetVideoInputRatio` 倍率单测，覆盖标准模型 480p/720p、1080p、4k、视频输入、fast 视频输入、fast 1080p 缺省回退和未知模型。

## 2026-07-03

- 回改 Seedance native -> OpenAI bridge 的上游请求体转换：native 路径不再把 `duration` 写入内部 `TaskSubmitReq.Duration/Seconds`，Doubao 上游调用只以 `metadata.duration` 为准。
- Doubao Seedance 上游 `requestPayload` 补充 `priority` 字段，metadata 中的 `priority` 会进入实际上游 JSON body。
- Doubao adaptor 检测到 metadata 原生 `content` 时保序透传全部 content，不再删除多个 text 后只追加第一个 prompt；旧 OpenAI Video 路径无原生 `content` 时继续按 `images + prompt` 组装。
- 按“便于后续跟进 new-api 上游更新”的要求收窄 Doubao adaptor diff：`DoResponse` 仅在 native response 分支前置新增处理并提前返回，默认 OpenAI Video response 原代码路径保持原样；`convertToRequestPayload` 仅在 `hasNativeContent` 时提前返回，其余原逻辑保持原样。

## 2026-07-02

- 实现 Seedance 2.0 BytePlus / ModelArk native `/api/v3/contents/generations/tasks` create / get / list：create 在 native API handler 内转为内部 OpenAI Video task request 并复用现有 relay task 提交、计费、落库和轮询链路；get/list 使用 public task id 和当前 user 渲染 native task object。
- Doubao task adaptor create 响应按 native mode 返回 `{ "id": "<public task id>" }`，默认 OpenAI Video wrapper 行为保持不变；`Task.Data` 写入可被 native renderer 与 OpenAI Video converter 复用的 canonical request snapshot。
- 新增 native error shell、native renderable channel guard（当前仅 DoubaoVideo / VolcEngine）、最近 7 天 list、分页、status、task_ids、model、service_tier 过滤；过滤在 native handler 内完成，未改 `model/task.go` 查询接口。
- 新增单元和路由测试覆盖 handler 内 native->OpenAI 请求转换、参数错误、鉴权错误外壳、route 注册、get/list 用户隔离和过滤、Doubao native/OpenAI 响应拆分、canonical taskData 和 OpenAI Video 兼容转换。
- 验证：相关包 `go test`、`go vet`、`git diff --check`、CodeGraph status 均通过；根包空跑因缺 `web/classic/dist` 阻塞，已在计划中记录。

## 2026-06-28

- 新增 XRTokenArkVideo 薄 task adaptor 计划与技术方案，并已进入 Phase 2/3 实现回改。
- 回改 Phase 1 Review 问题：补齐技术方案前置矩阵、自 review 结论和计划文件 Git 可追踪例外。
- Phase 2 TDD 实现 XRTokenArkVideo 内部 task adaptor：最终 ChannelType 为 `101`，`ChannelBaseURLs[59..100]` 为预留空字符串占位，`ChannelBaseURLs[101]` 为 `https://api.xrtoken.net`；新增 task adaptor 分发和 `relay/channel/task/xrtokenarkvideo` 包，覆盖 XRToken `/v1/contents/generations/tasks` create / fetch 路径和顶层 `video_url` 解析。
- 同步 default / classic 两套管理后台渠道类型下拉和图标映射为 `101`，未加入普通模型拉取类型集合。
- 新增单元测试覆盖 XRToken adaptor 分发、101 默认 BaseURL / `GetBaseURL()`、create URL、fetch URL、create response、顶层 `video_url` 到 `TaskInfo.Url` / OpenAI Video `metadata.url` 的转换、OpenAI Video `Task.GetResultURL()` fallback，并补 DoubaoVideo `/api/v3` 路径回归测试。
- 完成自 review：确认不新增公开视频路由、不实现 delete/cancel、不新增通用 `APIType` / `api_profile`；相关 Go 测试、`go vet`、`git diff --check`、SQL lint、环境 doctor 和 CodeGraph 同步通过。前端脚本因当前环境缺 `bun` / `oxfmt` / `prettier` 未执行成功，已记录为环境缺工具。
