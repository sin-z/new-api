# new-api 与 OpenAI Key 生成算法比对

## 任务范围

- 只读分析 `new-api` 用户 API key 的生成、展示和鉴权处理方式。
- 与 OpenAI 官方可验证信息做边界对比。
- 不修改业务源码、不运行服务、不访问生产 API。

## 事实依据

1. `common.GenerateKey()` 返回 `GenerateRandomCharsKey(48)`。
2. `GenerateRandomCharsKey` 使用 `crypto/rand.Int`，字符集为 `0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ`。
3. `controller.AddToken` 新增 token 时调用 `common.GenerateKey()`，并把结果写入 `model.Token.Key`。
4. `model.Token.Key` 是 `tokens.key` 字段，`GetFullKey()` 直接返回入库值，不添加 `sk-`。
5. `middleware.TokenAuth` 和 `TokenAuthReadOnly` 会从请求 key 中剥离 `sk-` 后再查库。
6. 默认前端展示、复制时给服务端返回的真实 key 加 `sk-` 前缀。
7. OpenAI 官方文档公开的是 API key 作为 Bearer 凭据使用方式，未公开 API key 生成算法。

## 结论

不能认为二者生成算法一致。

`new-api` 的可验证算法是：生成 48 位字母数字随机串，服务端存储不带 `sk-`，客户端显示和调用时补 `sk-`，服务端鉴权时再剥离。

OpenAI 的 API key 生成算法未公开；因此只能确认 `new-api` 在外观和使用方式上模拟了 OpenAI 的 `sk-...` 习惯，不能确认内部算法一致。

## 验证记录

- `git -C harness-engineering pull`：已是最新。
- `python3 harness-engineering/tools/harness_env.py doctor --profile bootstrap`：通过。
- `codegraph sync`、`codegraph status`：索引最新。
- `codegraph query "GenerateKey sk- Token"`：定位 `common/utils.go:254`。
- `codegraph callers "GenerateKey"`：确认 `controller.AddToken` 调用。
- 源码读取：`common/utils.go`、`controller/token.go`、`model/token.go`、`middleware/auth.go`、`web/default/src/features/keys/components/api-keys-provider.tsx`。

## 风险与回滚

- 本次仅新增分析留痕和重要发现记录，不改变运行逻辑。
- 回滚方式：删除本文件，并移除 `memories/important-findings.md` 中 2026-07-09 对应记录。
