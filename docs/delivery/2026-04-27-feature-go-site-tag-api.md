# 任务交付：Site / Tag 后端 API

## 任务背景

根据 `docs/project/navbox/tasks/task-04-site-tag-api.md`，实现网站和 Tag 的核心后端 API，包括游客查询和 admin 管理。

## 实现方案

1. 新增 Site / Tag DTO。
2. 扩展 Site / Tag Repository，支持列表、CRUD、多对多关系、默认 Tag、排序和批量操作。
3. 新增 Site / Tag Service，处理 ID 校验、Tag 绑定校验、多 Tag AND 筛选和 DTO 转换。
4. 新增 Site / Tag Handler，提供公开查询接口和 admin 管理接口。
5. admin 管理接口统一挂载 Session Middleware。
6. 多 Tag 筛选在数据库层使用 `GROUP BY + HAVING COUNT(DISTINCT tag_id)` 实现 AND 语义。

## 文件变更

新增文件：

- `internal/dto/site.go`
- `internal/dto/tag.go`
- `internal/handler/site.go`
- `internal/handler/tag.go`
- `internal/service/errors.go`
- `internal/service/site.go`
- `internal/service/tag.go`
- `internal/service/uuid.go`

修改文件：

- `internal/app/app.go`
- `internal/repo/site.go`
- `internal/repo/tag.go`
- `internal/server/router.go`
- `docs/project/navbox/PROGRESS.md`

## 新增接口

公开接口：

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/api/v1/sites` | 查询网站列表，支持搜索、Tag 筛选、视图筛选 |
| GET | `/api/v1/tags` | 查询 Tag 列表 |
| GET | `/api/v1/config/public` | 查询公开配置 |

admin 接口：

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| POST | `/api/v1/admin/sites` | 新增网站 |
| PUT | `/api/v1/admin/sites/:id` | 更新网站 |
| DELETE | `/api/v1/admin/sites/:id` | 删除网站 |
| POST | `/api/v1/admin/sites/batch-delete` | 批量删除网站 |
| POST | `/api/v1/admin/sites/batch-tags` | 批量添加或移除 Tag |
| PUT | `/api/v1/admin/sites/order` | 调整网站顺序 |
| POST | `/api/v1/admin/tags` | 新增 Tag |
| PUT | `/api/v1/admin/tags/:id` | 更新 Tag |
| DELETE | `/api/v1/admin/tags/:id` | 删除 Tag |
| PUT | `/api/v1/admin/tags/:id/default` | 设置默认 Tag |
| PUT | `/api/v1/admin/tags/order` | 调整 Tag 顺序 |

## 测试结果

已执行：

```text
go test ./...
go build ./...
npm --prefix web run build
docker compose build navbox
docker compose up -d postgres navbox
curl -sS http://127.0.0.1:8080/api/v1/health
curl -sS -c /tmp/navbox-task04-cookies.txt -H 'Content-Type: application/json' -d '{"password":"<admin-password>"}' http://127.0.0.1:8080/api/v1/admin/login
curl -sS -b /tmp/navbox-task04-cookies.txt -H 'Content-Type: application/json' -d '{"name":"Dev"}' http://127.0.0.1:8080/api/v1/admin/tags
curl -sS -b /tmp/navbox-task04-cookies.txt -H 'Content-Type: application/json' -d '{"name":"Monitor"}' http://127.0.0.1:8080/api/v1/admin/tags
curl -sS -b /tmp/navbox-task04-cookies.txt -H 'Content-Type: application/json' -d '{"title":"Grafana","default_url":"https://grafana.example.com","tag_ids":["<dev-tag-id>","<monitor-tag-id>"]}' http://127.0.0.1:8080/api/v1/admin/sites
curl -sS 'http://127.0.0.1:8080/api/v1/sites?tag_ids=<dev-tag-id>,<monitor-tag-id>'
curl -sS 'http://127.0.0.1:8080/api/v1/sites?view=uncategorized'
curl -sS -b /tmp/navbox-task04-cookies.txt -X PUT http://127.0.0.1:8080/api/v1/admin/tags/<dev-tag-id>/default
curl -sS -b /tmp/navbox-task04-cookies.txt -X DELETE http://127.0.0.1:8080/api/v1/admin/tags/<monitor-tag-id>
curl -sS http://127.0.0.1:8080/api/v1/config/public
curl -sS -i -H 'Content-Type: application/json' -d '{"name":"Blocked"}' http://127.0.0.1:8080/api/v1/admin/tags
docker compose down
```

验证结果：

1. 游客可以查询 Site 和 Tag。
2. 未登录游客调用 admin 写接口返回 401。
3. admin 可以新增 Tag。
4. admin 可以新增 Site 并绑定多个 Tag。
5. 多 Tag 筛选返回同时包含所有选中 Tag 的 Site。
6. `view=uncategorized` 可以返回无 Tag 网站。
7. 设置默认 Tag 后，公开配置返回默认 Tag ID。
8. 删除 Tag 不会删除 Site，只会移除关联关系。

## 风险与后续建议

1. 当前接口已覆盖后端能力，前端管理界面将在后续任务实现。
2. 当前批量操作已提供接口，前端二次确认和结果展示需要在任务 08 处理。
3. 后续 icon 字段会在任务 05 接入实际上传与访问能力。
