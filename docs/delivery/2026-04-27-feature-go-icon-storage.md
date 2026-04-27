# 任务交付：icon 存储与访问

## 任务背景

根据 `docs/project/navbox/tasks/task-05-icon-storage.md`，实现 icon 上传、校验、保存、公开访问和元数据记录能力。

## 实现方案

1. 新增 icon DTO、Repository、Service、Handler。
2. icon 文件保存到 `./data/uploads/`，文件名使用内容 SHA256，避免覆盖和路径穿越。
3. 上传接口仅允许 admin 通过 Session 访问。
4. 文件校验限制 MIME、扩展名、大小和空文件。
5. 上传成功后写入 `icons` 表，重复内容复用已有元数据。
6. 通过 `/uploads/:file` 提供只读公开访问，并限制只能读取文件名。

## 文件变更

新增文件：

- `internal/dto/icon.go`
- `internal/handler/icon.go`
- `internal/service/icon.go`
- `internal/storage/icon.go`
- `internal/storage/icon_test.go`
- `internal/storage/provider.go`

修改文件：

- `internal/app/app.go`
- `internal/repo/icon.go`
- `internal/server/router.go`
- `docs/project/navbox/PROGRESS.md`

## 新增接口

admin 接口：

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| POST | `/api/v1/admin/icons/upload` | 上传 icon 文件 |

公开访问：

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/uploads/:file` | 读取已上传 icon 文件 |

## 配置与依赖变更

1. 使用既有上传配置 `upload.dir`、`upload.max_icon_size`、`upload.allowed_icon_mimes`。
2. 默认上传目录为 `./data/uploads/`。
3. 无新增 Go 或前端依赖。

## 测试结果

已执行：

```text
go test ./...
go build ./...
npm --prefix web run build
docker compose build navbox
docker compose up -d postgres navbox
curl -sS -i -c /tmp/navbox-icon-cookies.txt -H 'Content-Type: application/json' -d '{"password":"<admin-password>"}' http://127.0.0.1:8037/api/v1/admin/login
curl -sS -i -F file=@/tmp/navbox-icon.png http://127.0.0.1:8037/api/v1/admin/icons/upload
curl -sS -i -b /tmp/navbox-icon-cookies.txt -F file=@/tmp/navbox-icon.png http://127.0.0.1:8037/api/v1/admin/icons/upload
curl -sS -i http://127.0.0.1:8037/uploads/<icon-file-name>
curl -sS -i -b /tmp/navbox-icon-cookies.txt -F file=@/tmp/navbox-not-image.txt http://127.0.0.1:8037/api/v1/admin/icons/upload
docker compose down
```

验证结果：

1. 未登录游客调用上传接口返回 401。
2. admin 上传合法 PNG 返回 200，并返回 icon 元数据和公开 URL。
3. 上传后的 icon 文件保存在 `./data/uploads/`。
4. `/uploads/:file` 可以公开读取 icon，响应 `Content-Type` 为 `image/png`。
5. 非图片文件上传返回 400。

## 风险与后续建议

1. 当前只实现上传和公开访问；网站选择、替换 icon 的前端体验将在后续 admin 前端任务中实现。
2. icon 删除和孤儿文件清理尚未实现，后续可在删除网站或导入覆盖策略中统一处理。
3. 导入导出任务需要把 `icons` 元数据和 `./data/uploads/` 文件一起纳入 zip。
