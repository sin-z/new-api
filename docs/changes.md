# Changes

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
