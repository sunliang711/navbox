# Bug 修复交付：导入失败日志与提示

## 问题背景

后台选择 zip 导入时，接口 `/api/v1/admin/import` 返回 500，但容器日志只有 request logger，无法看到导入失败的底层原因。

## 根因分析

导入导出 Handler 在 Service 返回错误时直接调用统一错误响应，未记录业务错误。当前运行容器也加载了旧构建产物，导致用户看到 500 时缺少可排查日志。

## 修复方案

1. `ImportExportHandler` 注入 `zerolog.Logger`。
2. 导入失败时记录 `Import config failed` 错误日志。
3. 导出失败时记录 `Export config failed` 错误日志。
4. 前端导入失败时显示失败提示，并清空文件选择框，允许再次选择同一个 zip。
5. 同步更新嵌入式前端产物。

## 文件变更

- `internal/handler/import_export.go`
- `web/src/AdminApp.tsx`
- `web/src/preferences.tsx`
- `internal/web/dist/index.html`
- `internal/web/dist/assets/index-B7G7xxF2.js`
- `internal/web/dist/assets/index-BBdTjZRt.js`

## 验证结果

已执行：

```text
go test ./...
npm --prefix web run build
go build ./...
docker compose up -d --build navbox
curl -sS -i -b <test-session> -F file=@/Users/eagle/Downloads/navbox-export-20260428070730.zip http://localhost:8037/api/v1/admin/import
```

验证结果：

1. Go 测试通过。
2. 前端构建通过。
3. Go 构建通过。
4. Docker 镜像重建并启动成功。
5. 使用 2026-04-28 07:07:30 导出的 zip 调用导入接口返回 200。
6. 导入报告显示 6 个网站、5 个 Tag、8 个 icon 均因已存在而跳过，符合重复导入预期。

## 风险与后续建议

当前修复不会改变导入冲突策略。若后续仍出现 500，日志中会出现 `Import config failed` 并带有底层错误，可直接据此定位具体数据或文件问题。
