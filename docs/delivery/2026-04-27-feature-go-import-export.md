# 任务交付：导入导出

## 任务背景

根据 `docs/project/navbox/tasks/task-06-import-export.md`，实现 admin 配置导入导出能力，支持 zip 包、`navbox.json`、icon 文件、按范围导出、导入冲突跳过和报告返回。

## 实现方案

1. 新增导入导出 DTO，定义 zip 内 `navbox.json` 的站点、Tag、icon 数据结构。
2. 新增 ImportExport Repository，负责导出数据查询、导入冲突查询和事务写入。
3. 新增 ImportExport Service，负责 zip 编解码、结构校验、icon 文件校验、冲突判断和导入报告生成。
4. 新增 ImportExport Handler，提供 admin 导出和导入接口。
5. 导出 zip 包含 `navbox.json` 和 `icons/` 目录。
6. 导入 zip 时拒绝路径穿越、未知文件、非法 JSON、非法 icon 和超出限制的文件。
7. 导入冲突按 Site ID、Tag ID/名称、Icon ID/SHA256 判断，冲突项跳过并写入报告。
8. icon 文件先写入上传目录，数据库导入失败时清理本次新写入文件。

## 文件变更

新增文件：

- `internal/dto/import_export.go`
- `internal/handler/import_export.go`
- `internal/repo/import_export.go`
- `internal/service/import_export.go`
- `internal/service/import_export_test.go`

修改文件：

- `internal/app/app.go`
- `internal/server/router.go`
- `internal/storage/icon.go`
- `docs/project/navbox/PROGRESS.md`

## 新增接口

admin 接口：

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| POST | `/api/v1/admin/export` | 导出配置 zip，Body 可传 `site_ids` 或 `tag_ids` |
| POST | `/api/v1/admin/import` | 上传并导入配置 zip，表单字段为 `file` |

导出请求示例：

```json
{
  "site_ids": ["<site-id>"],
  "tag_ids": ["<tag-id>"]
}
```

## 配置与依赖变更

1. 无新增 Go 依赖。
2. 无新增配置项。
3. 导入 zip 最大限制为 64MB。
4. 单个 icon 文件仍沿用上传配置 `upload.max_bytes`。

## 测试结果

已执行：

```text
go test ./...
go build ./...
npm --prefix web run build
docker compose build navbox
docker compose up -d postgres navbox
curl -sS -i -c /tmp/navbox-task06-cookies.txt -H 'Content-Type: application/json' -d '{"password":"<admin-password>"}' http://localhost:8080/api/v1/admin/login
curl -sS -b /tmp/navbox-task06-cookies.txt -F file=@/tmp/navbox-task06-icon.png http://localhost:8080/api/v1/admin/icons/upload
curl -sS -b /tmp/navbox-task06-cookies.txt -H 'Content-Type: application/json' -d '{"name":"Task06-1777282658","icon":"","color":"#2563eb","sort_order":606}' http://localhost:8080/api/v1/admin/tags
curl -sS -b /tmp/navbox-task06-cookies.txt -H 'Content-Type: application/json' -d '{"title":"Task06 Site 1777282658","default_url":"https://task06.example.com","icon_type":"image","icon_value":"/uploads/<icon-file>","tag_ids":["<tag-id>"]}' http://localhost:8080/api/v1/admin/sites
curl -sS -b /tmp/navbox-task06-cookies.txt -H 'Content-Type: application/json' -d '{"site_ids":["<site-id>"]}' -o /tmp/navbox-task06-export.zip -D /tmp/navbox-task06-export.headers http://localhost:8080/api/v1/admin/export
unzip -l /tmp/navbox-task06-export.zip
unzip -p /tmp/navbox-task06-export.zip navbox.json | jq '{version, sites: (.sites|length), tags: (.tags|length), icons: (.icons|length)}'
curl -sS -i -b /tmp/navbox-task06-cookies.txt -F file=@/tmp/navbox-task06-export.zip http://localhost:8080/api/v1/admin/import
curl -sS -i -b /tmp/navbox-task06-cookies.txt -X DELETE http://localhost:8080/api/v1/admin/sites/<site-id>
curl -sS -i -b /tmp/navbox-task06-cookies.txt -X DELETE http://localhost:8080/api/v1/admin/tags/<tag-id>
curl -sS -i -b /tmp/navbox-task06-cookies.txt -F file=@/tmp/navbox-task06-export.zip http://localhost:8080/api/v1/admin/import
curl -sS 'http://localhost:8080/api/v1/sites?tag_ids=<tag-id>'
curl -sS -i -H 'Content-Type: application/json' -d '{}' http://localhost:8080/api/v1/admin/export
curl -sS -i -b /tmp/navbox-task06-cookies.txt -F file=@/tmp/navbox-task06-icon.png http://localhost:8080/api/v1/admin/import
docker compose down
```

验证结果：

1. 未登录调用导出接口返回 401。
2. admin 可按指定 Site 导出 zip。
3. admin 可按指定 Tag 导出 zip。
4. 导出 zip 包含 `navbox.json` 和 `icons/<file>`。
5. 重复导入同一个 zip 时，Site、Tag、Icon、关联均跳过并返回冲突报告。
6. 删除测试 Site/Tag 后重新导入同一个 zip，Site、Tag、关联可重新创建。
7. 非 zip 文件导入返回 400。
8. 导入后的 Site 可通过公开 Site 查询接口按 Tag 查到。

## 风险与后续建议

1. 当前导入使用数据库事务保证数据写入一致性；文件系统无法参与数据库事务，已在数据库失败时清理本次新增 icon 文件。
2. 当前导入不会覆盖已有数据，冲突项只跳过并报告。
3. 后续前端任务需要提供导出范围选择、导入报告展示和失败提示。
