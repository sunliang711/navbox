# 任务交付：admin 密码恢复模式

## 任务背景

参照 `~/Sync/traffic-monitor` 的 restore mode，为 Navbox 增加忘记 admin 密码时的临时恢复机制。

## 实现方案

1. 新增 `[restore]` 配置段，通过 `mode=admin-password` 启用恢复模式。
2. 恢复模式必须配置长度不少于 32 字符的 `token`。
3. 恢复 Token 启动后 5 分钟内有效，成功使用后一次性失效。
4. 恢复模式下 `/admin` 显示密码恢复表单。
5. 恢复成功后重置唯一 admin 密码，并清理所有 admin Session。
6. 恢复成功后需要删除恢复配置并重启服务。

## 文件变更

新增文件：

- `internal/handler/restore.go`
- `internal/handler/restore_test.go`
- `internal/config/config_test.go`

修改文件：

- `internal/config/config.go`
- `internal/dto/auth.go`
- `internal/repo/auth.go`
- `internal/service/auth.go`
- `internal/service/auth_test.go`
- `internal/server/router.go`
- `internal/app/app.go`
- `internal/response/response.go`
- `web/src/AdminApp.tsx`
- `web/src/api.ts`
- `web/src/types.ts`
- `web/src/styles.css`
- `config/config.toml`
- `docker-compose.yml`
- `README.md`
- `docs/deploy/2026-04-27-deploy-go-navbox.md`

## 配置与环境变量

新增配置：

```toml
[restore]
mode = ""
token = ""
```

新增环境变量：

```text
NAVBOX_RESTORE_MODE=admin-password
NAVBOX_RESTORE_TOKEN=<openssl rand -hex 32>
```

## 新增接口

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/api/v1/restore/status` | 查询恢复模式状态 |
| POST | `/api/v1/restore/admin-password` | 使用 Restore Token 重置 admin 密码 |

## 测试结果

已执行：

```text
go test ./...
go build ./...
npm --prefix web run build
```

验证结果：

1. 恢复配置校验覆盖合法模式、非法模式、短 Token。
2. Restore Token 错误、过期、重复使用均会被拒绝。
3. Restore Token 正确时可重置 admin 密码。
4. 重置密码后旧密码失效，新密码可登录。
5. 重置密码后所有旧 admin Session 被清理。
6. 前端构建通过，恢复模式下 `/admin` 可展示恢复表单。

## 风险与后续建议

1. `NAVBOX_RESTORE_TOKEN` 属于敏感配置，禁止写入公开配置仓库。
2. 恢复模式仅用于临时恢复密码，完成后必须删除 `NAVBOX_RESTORE_MODE` 和 `NAVBOX_RESTORE_TOKEN` 并重启服务。
3. Restore Token 有效期从服务启动时开始计算，超过 5 分钟需要重新生成 Token 并重启服务。
