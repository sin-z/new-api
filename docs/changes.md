# Changes

## 2026-06-28

- 新增 XRTokenArkVideo 薄 task adaptor 计划与技术方案，并已进入 Phase 2/3 实现回改。
- 回改 Phase 1 Review 问题：补齐技术方案前置矩阵、自 review 结论和计划文件 Git 可追踪例外。
- Phase 2 TDD 实现 XRTokenArkVideo 内部 task adaptor：最终 ChannelType 为 `101`，`ChannelBaseURLs[59..100]` 为预留空字符串占位，`ChannelBaseURLs[101]` 为 `https://api.xrtoken.net`；新增 task adaptor 分发和 `relay/channel/task/xrtokenarkvideo` 包，覆盖 XRToken `/v1/contents/generations/tasks` create / fetch 路径和顶层 `video_url` 解析。
- 同步 default / classic 两套管理后台渠道类型下拉和图标映射为 `101`，未加入普通模型拉取类型集合。
- 新增单元测试覆盖 XRToken adaptor 分发、101 默认 BaseURL / `GetBaseURL()`、create URL、fetch URL、create response、顶层 `video_url` 到 `TaskInfo.Url` / OpenAI Video `metadata.url` 的转换、OpenAI Video `Task.GetResultURL()` fallback，并补 DoubaoVideo `/api/v3` 路径回归测试。
- 完成自 review：确认不新增公开视频路由、不实现 delete/cancel、不新增通用 `APIType` / `api_profile`；相关 Go 测试、`go vet`、`git diff --check`、SQL lint、环境 doctor 和 CodeGraph 同步通过。前端脚本因当前环境缺 `bun` / `oxfmt` / `prettier` 未执行成功，已记录为环境缺工具。
