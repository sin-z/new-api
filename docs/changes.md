# Changes

## 2026-06-29

- Phase 2 TDD 实现 Seedance native create / get / list 中转：新增 `/api/v3/contents/generations/tasks` POST / GET by id / GET list 路由，不新增 DELETE / cancel。
- 新增 Seedance native create request 转换与校验：native-only 字段写入 OpenAI Video internal request metadata；拒绝 `callback_url`、非 `default` service tier、`frames`、`seed`、`camera_fixed`。
- 新增 Seedance native get/list renderer：基于当前用户维度 task 查询事实源，使用 public `Task.TaskID` 渲染 native response，过滤非 Seedance native 平台任务，非本人 / 不存在 / 不可渲染统一 native 404。
- 回改 Review 问题：业务代码 JSON 解析改走 `common.*` wrapper；list 在 DAO 层先过滤 native platforms 并分批扫描，修正分页 / total 语义；分页非法参数返回 native 400；create/list 错误码与 HTTP status 按 contract 精确映射；GET/list 不再挂 `Distribute()`；补齐 handler 中文职责注释。
- 二次回改 Review 问题：`filter.status` 非法值返回 native 400 `InvalidParameter.InvalidValue` / `BadRequest`，不再落入 500 `InternalServiceError`。
- 调整 Doubao native create response 分支：客户端只返回 public id；`Task.Data` 写入 canonical Seedance task data，保留 `content.video_url`、状态、model、service tier、request snapshot，不把 upstream task id 写入响应 `id`。
- 补充 Go 测试覆盖 native request metadata 保真、unsupported 字段校验、`TokenAuth()` / `Distribute()` 路由注册、middleware relay mode、public id / upstream id 隔离、OpenAI/native create-get 互查 renderer、OpenAI wrapper 消费 canonical `Task.Data`、list task id / platforms 过滤、分页非法参数、错误码映射和用户维度查询。

## 2026-06-28

- 新增 XRTokenArkVideo 薄 task adaptor 计划与技术方案，并已进入 Phase 2/3 实现回改。
- 回改 Phase 1 Review 问题：补齐技术方案前置矩阵、自 review 结论和计划文件 Git 可追踪例外。
- Phase 2 TDD 实现 XRTokenArkVideo 内部 task adaptor：最终 ChannelType 为 `101`，`ChannelBaseURLs[59..100]` 为预留空字符串占位，`ChannelBaseURLs[101]` 为 `https://api.xrtoken.net`；新增 task adaptor 分发和 `relay/channel/task/xrtokenarkvideo` 包，覆盖 XRToken `/v1/contents/generations/tasks` create / fetch 路径和顶层 `video_url` 解析。
- 同步 default / classic 两套管理后台渠道类型下拉和图标映射为 `101`，未加入普通模型拉取类型集合。
- 新增单元测试覆盖 XRToken adaptor 分发、101 默认 BaseURL / `GetBaseURL()`、create URL、fetch URL、create response、顶层 `video_url` 到 `TaskInfo.Url` / OpenAI Video `metadata.url` 的转换、OpenAI Video `Task.GetResultURL()` fallback，并补 DoubaoVideo `/api/v3` 路径回归测试。
- 完成自 review：确认不新增公开视频路由、不实现 delete/cancel、不新增通用 `APIType` / `api_profile`；相关 Go 测试、`go vet`、`git diff --check`、SQL lint、环境 doctor 和 CodeGraph 同步通过。前端脚本因当前环境缺 `bun` / `oxfmt` / `prettier` 未执行成功，已记录为环境缺工具。
