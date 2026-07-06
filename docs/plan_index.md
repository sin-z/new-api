# Plan Index

| Date | Task | Status | Plan | OverallPlan | ScopeSummary |
| --- | --- | --- | --- | --- | --- |
| 2026-06-28 | XRTokenArkVideo 薄 Task Adaptor | `completed` | `docs/plans/2026-06-28_XRTokenArkVideo薄TaskAdaptor.md` | `docs/plans/2026-06-28_XRTokenArkVideo薄TaskAdaptor.md` | 新增 XRTokenArkVideo 薄 task adaptor，保持 DoubaoVideo 既有 adaptor 行为不变，不新增公开 ARK 路由。 |
| 2026-07-02 | Seedance 2.0 原生接口 OpenAI 中转代码开发 | `completed` | `docs/plans/2026-07-02_Seedance2原生接口OpenAI中转代码开发.md` | `../docs/tech-design/token-gateway/seedance-2-native-openai-bridge-server-tech-design.md` | 新增 Seedance native `/api/v3/contents/generations/tasks` create/get/list，handler 内桥接到 OpenAI Video internal request 与现有 relay task；不新增转换 middleware，不修改 shared model 查询接口，保持账务、轮询、DB schema 和 OpenAI Video 既有行为不变。 |
| 2026-07-06 | Seedance 2.0 分辨率 USD 价格表调整 | `completed` | `docs/plans/2026-07-06_Seedance2分辨率USD价格表调整.md` | `docs/plans/2026-07-06_Seedance2分辨率USD价格表调整.md` | 将 Doubao Seedance 2.0 / fast 分辨率价格表调整为 BytePlus 海外官方 USD / M tokens 口径，并补充倍率单测。 |
