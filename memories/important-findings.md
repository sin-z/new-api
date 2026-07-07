# Important Findings

- 日期：2026-07-07
  场景：Seedance 2.0 native create 连通性修复
  发现内容：Seedance 2.0 上游不接受显式 `service_tier=default`；本地 native canonical data 仍需要保存 `service_tier=default` 以便 get/list 渲染。
  依据来源：远端 `testnapi.token168.ai` 复测返回上游错误消息；本地 `convertToRequestPayload` 回归测试确认上游请求体省略 `service_tier`。
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
  发现内容：当前本地 `new-api` 修复后 create/get/list 已通过，但 `https://testnapi.token168.ai` 仍返回旧的 `service_tier` 上游拒绝错误；远端需部署本地修复后再复测成功。
  依据来源：本地 30169 HTTP 验证 create/get/list 返回 200；远端 2026-07-07 复测 `duration=1` 返回 400 `fail_to_fetch_task` 且消息仍指向 `service_tier`。
  适用范围：Seedance native API 发布验证与测试环境问题判断。
