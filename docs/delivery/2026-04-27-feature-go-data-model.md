# 任务交付：数据模型、数据库连接与迁移

## 任务背景

根据 `docs/project/navbox/tasks/task-02-data-model.md`，完成 PostgreSQL 连接、GORM Model、基础迁移和 Repository 基础查询能力。

## 实现方案

1. 新增数据库配置项。
2. 使用 GORM PostgreSQL Driver 初始化数据库连接。
3. 配置数据库连接池。
4. 启动阶段执行数据库 Ping 和 AutoMigrate。
5. 定义 Site、Tag、SiteTag、AdminSetting、AdminSession、Icon Model。
6. 新增 Site、Tag、Auth、Icon Repository 基础接口和实现。
7. 使用 sqlmock 覆盖基础 Count 查询测试。

## 文件变更

新增文件：

- `internal/database/database.go`
- `internal/database/migrate.go`
- `internal/model/base.go`
- `internal/model/site.go`
- `internal/model/tag.go`
- `internal/model/site_tag.go`
- `internal/model/admin_setting.go`
- `internal/model/admin_session.go`
- `internal/model/icon.go`
- `internal/repo/site.go`
- `internal/repo/tag.go`
- `internal/repo/auth.go`
- `internal/repo/icon.go`
- `internal/repo/repo_test.go`

修改文件：

- `internal/config/config.go`
- `internal/app/app.go`
- `config/config.toml`
- `go.mod`
- `go.sum`
- `docs/project/navbox/PROGRESS.md`

## 配置与依赖变更

新增配置：

```toml
[database]
dsn = "host=localhost user=navbox password=navbox dbname=navbox port=5432 sslmode=disable TimeZone=UTC"
max_open_conns = 20
max_idle_conns = 5
conn_max_lifetime = "1h"
conn_max_idle_time = "30m"
connect_timeout = "5s"
```

新增 Go 依赖：

- `gorm.io/gorm`
- `gorm.io/driver/postgres`
- `github.com/google/uuid`
- `github.com/DATA-DOG/go-sqlmock`

## 数据变更

AutoMigrate 创建以下表：

- `sites`
- `tags`
- `site_tags`
- `admin_settings`
- `admin_sessions`
- `icons`

启动时会创建 PostgreSQL 扩展：

- `pgcrypto`

## 测试结果

已执行：

```text
go test ./...
go build ./...
npm --prefix web run build
docker compose build navbox
docker compose up -d postgres navbox
curl -sS http://127.0.0.1:8037/api/v1/health
docker compose exec -T postgres psql -U navbox -d navbox -c "\dt"
docker compose down
```

健康检查返回：

```json
{"code":200,"data":{"status":"ok"},"message":"success"}
```

数据库表验证结果包含：

```text
admin_sessions
admin_settings
icons
site_tags
sites
tags
```

## 风险与后续建议

1. 当前迁移使用 GORM AutoMigrate，后续复杂 schema 变更建议引入版本化迁移工具。
2. 当前 Repository 仅提供基础查询，完整 CRUD 将在任务 04 实现。
3. admin 密码初始化和 Session 逻辑将在任务 03 实现。
