# 任务交付：admin 认证与 Session

## 任务背景

根据 `docs/project/navbox/tasks/task-03-admin-auth-session.md`，实现 admin 初始密码、登录、Session、退出和修改密码能力。

## 实现方案

1. 新增 admin 认证配置。
2. 首次启动且未配置密码时生成 16 位随机初始密码。
3. 初始密码通过英文日志打印。
4. admin 密码使用 bcrypt 存储。
5. 登录成功后创建 Session，并通过 HttpOnly Cookie 返回 token。
6. 数据库只保存 Session token hash。
7. 管理接口通过 Session Middleware 校验。
8. 支持退出登录和修改密码。

## 文件变更

新增文件：

- `internal/dto/auth.go`
- `internal/handler/auth.go`
- `internal/middleware/auth.go`
- `internal/service/auth.go`
- `internal/service/auth_test.go`

修改文件：

- `internal/config/config.go`
- `internal/repo/auth.go`
- `internal/server/router.go`
- `internal/app/app.go`
- `internal/response/response.go`
- `config/config.toml`
- `docker-compose.yml`
- `go.mod`
- `docs/project/navbox/PROGRESS.md`

## 配置与依赖变更

新增配置：

```toml
[auth]
session_ttl = "24h"
cookie_name = "navbox_admin_session"
cookie_secure = false
initial_password_length = 16
```

新增环境变量：

```text
NAVBOX_AUTH_SESSION_TTL=24h
```

新增 Go 依赖：

- `golang.org/x/crypto/bcrypt`

## 新增接口

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| POST | `/api/v1/admin/login` | admin 登录 |
| POST | `/api/v1/admin/logout` | admin 退出 |
| GET | `/api/v1/admin/session` | 检查当前 Session |
| POST | `/api/v1/admin/password` | 修改 admin 密码 |

## 测试结果

已执行：

```text
go test ./...
go build ./...
npm --prefix web run build
docker compose build navbox
docker compose up -d postgres navbox
docker compose logs --no-color navbox
curl -sS -i http://127.0.0.1:8037/api/v1/admin/session
curl -sS -i -c /tmp/navbox-cookies.txt -H 'Content-Type: application/json' -d '{"password":"<initial-password>"}' http://127.0.0.1:8037/api/v1/admin/login
curl -sS -i -b /tmp/navbox-cookies.txt http://127.0.0.1:8037/api/v1/admin/session
curl -sS -i -b /tmp/navbox-cookies.txt -H 'Content-Type: application/json' -d '{"current_password":"<initial-password>","new_password":"<new-password>"}' http://127.0.0.1:8037/api/v1/admin/password
curl -sS -i -H 'Content-Type: application/json' -d '{"password":"<initial-password>"}' http://127.0.0.1:8037/api/v1/admin/login
curl -sS -i -c /tmp/navbox-cookies-new.txt -H 'Content-Type: application/json' -d '{"password":"<new-password>"}' http://127.0.0.1:8037/api/v1/admin/login
curl -sS -i -b /tmp/navbox-cookies-new.txt -X POST http://127.0.0.1:8037/api/v1/admin/logout
curl -sS -i -b /tmp/navbox-cookies-new.txt http://127.0.0.1:8037/api/v1/admin/session
docker compose down
```

验证结果：

1. 未登录访问 `/api/v1/admin/session` 返回 401。
2. 使用初始密码登录成功，并返回 HttpOnly Cookie。
3. 登录后访问 `/api/v1/admin/session` 返回成功。
4. 修改密码成功。
5. 旧密码登录失败。
6. 新密码登录成功。
7. 退出后 Session 失效。
8. `admin_settings.password_hash` 已加密存储，长度为 60。

## 风险与后续建议

1. 当前 Cookie `secure` 默认关闭，生产 HTTPS 部署时建议开启 `NAVBOX_AUTH_COOKIE_SECURE=true`。
2. 当前未实现 CSRF token，后续管理界面成型后可评估是否补充。
3. 后续 Site / Tag 管理接口需要挂载同一个 Session Middleware。
