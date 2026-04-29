# Session 滑动续期

## 任务背景

管理后台原 Session 为固定有效期，登录后只在创建时写入 `expires_at` 和 Cookie Max-Age，后续请求不会续期。当前实现改为滑动续期，避免活跃使用管理后台时 Session 到点失效。

## 实现方案

1. 保持现有 Session token 和 Cookie 结构不变，不新增接口、不改表结构。
2. 在 Admin Session 中间件校验时调用带续期的 Service 方法。
3. 当 Session 剩余有效期低于 `auth.session_ttl / 2` 时，将数据库 `expires_at` 更新为 `now + auth.session_ttl`。
4. 仅在发生续期时重新下发同名 HttpOnly Cookie，Cookie Max-Age 仍使用 `auth.session_ttl`。

## 文件变更

- `internal/service/auth.go`：新增 `ValidateSessionWithRenewal` 和续期判断逻辑。
- `internal/repo/auth.go`：新增 `UpdateSessionExpiresAt`，按 token hash 更新未过期 Session 的过期时间。
- `internal/middleware/auth.go`：Admin Session 校验改为支持滑动续期，并在续期时重发 Cookie。
- `internal/service/auth_test.go`：覆盖低于半个 TTL 时续期、剩余时间充足时不续期。
- `internal/middleware/auth_test.go`：覆盖续期时写 `Set-Cookie`、未续期时不写 Cookie。

## 配置与依赖

- 未新增配置项。
- 未新增依赖。
- 仍使用现有 `auth.session_ttl` 控制 Session 总有效期和滑动续期期限。

## 测试结果

```bash
go test ./...
```

结果：通过。

## 风险与后续建议

- 当前方案是纯滑动续期，只要持续活跃访问受保护的 admin API，就可以持续延长 Session。
- 如果后续需要强制最长登录周期，可增加绝对生命周期配置，例如 `auth.session_max_lifetime`。
