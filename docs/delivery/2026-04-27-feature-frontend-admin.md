# 任务交付：admin 前端管理能力

## 任务背景

根据 `docs/project/navbox/tasks/task-08-frontend-admin.md`，实现 admin 登录、网站管理、Tag 管理、批量操作、导入导出和密码修改。

## 实现方案

1. 新增隐藏管理路径 `/admin`，游客首页不展示 admin 管理入口。
2. admin 进入 `/admin` 后先检查 Session，未登录时展示登录界面。
3. 登录成功后通过 Session Cookie 调用现有 admin API。
4. 新增 admin API Client，覆盖认证、网站、Tag、icon、导入导出、密码修改接口。
5. 实现网站管理表格、侧边编辑面板、icon 上传、排序保存、删除。
6. 实现 Tag 管理表格、侧边编辑面板、默认 Tag、排序保存、删除。
7. 实现网站批量选择、批量添加 Tag、批量移除 Tag、批量删除。
8. 实现导出全部、导出选中网站、导出选中 Tag，以及 zip 导入和导入报告展示。
9. 实现修改 admin 密码和退出登录。

## 文件变更

新增文件：

- `web/src/AdminApp.tsx`

修改文件：

- `web/src/App.tsx`
- `web/src/api.ts`
- `web/src/types.ts`
- `web/src/styles.css`
- `internal/web/dist/index.html`
- `internal/web/dist/assets/index-DC3E-Fnw.js`
- `internal/web/dist/assets/index-Cx-i8HeA.css`
- `docs/project/navbox/PROGRESS.md`

## admin 能力

1. 登录、退出、Session 状态检测。
2. 修改 admin 密码。
3. 新增、编辑、删除网站。
4. 上传网站 icon。
5. 批量删除网站。
6. 批量添加和移除 Tag。
7. 调整网站排序。
8. 新增、编辑、删除 Tag。
9. 设置默认 Tag。
10. 调整 Tag 排序。
11. 导入和导出配置。

## 配置与依赖变更

1. 无新增前端依赖。
2. 无新增后端依赖。
3. 无新增配置项。

## 测试结果

已执行：

```text
npm --prefix web run build
go test ./...
go build ./...
docker compose build navbox
docker compose up -d navbox
curl -sS -i http://localhost:8037/admin
curl -sS -i http://localhost:8037/
curl -sS -i http://localhost:8037/api/v1/admin/session
curl -sS -i -c /tmp/navbox-admin-ui-cookies.txt -H 'Content-Type: application/json' -d '{"password":"<admin-password>"}' http://127.0.0.1:8037/api/v1/admin/login
curl -sS -i -b /tmp/navbox-admin-ui-cookies.txt http://127.0.0.1:8037/api/v1/admin/session
curl -sS -b /tmp/navbox-admin-ui-cookies.txt -H 'Content-Type: application/json' -d '{"name":"UI-1777284202","icon":"","color":"#2f7d6d","sort_order":808,"is_enabled":true}' http://127.0.0.1:8037/api/v1/admin/tags
curl -sS -b /tmp/navbox-admin-ui-cookies.txt -H 'Content-Type: application/json' -d '{"title":"UI Site 1777284202","default_url":"https://ui-smoke.example.com","icon_type":"text","icon_value":"UI","tag_ids":["<tag-id>"]}' http://127.0.0.1:8037/api/v1/admin/sites
curl -sS -b /tmp/navbox-admin-ui-cookies.txt -H 'Content-Type: application/json' -d '{"site_ids":["<site-id>"]}' -o /tmp/navbox-admin-ui-export.zip -D /tmp/navbox-admin-ui-export.headers http://127.0.0.1:8037/api/v1/admin/export
curl -sS -i -b /tmp/navbox-admin-ui-cookies.txt -X DELETE http://127.0.0.1:8037/api/v1/admin/sites/<site-id>
curl -sS -i -b /tmp/navbox-admin-ui-cookies.txt -X DELETE http://127.0.0.1:8037/api/v1/admin/tags/<tag-id>
```

验证结果：

1. `/admin` 返回嵌入式前端资源。
2. 游客首页 `/` 返回嵌入式前端资源。
3. 未登录调用 admin Session 返回 401。
4. admin 登录后 Session 返回 200。
5. admin Tag 创建接口可用。
6. admin Site 创建接口可用。
7. admin 导出接口可用。
8. 测试创建的 Site 和 Tag 已删除。
9. Docker 服务当前保持运行，可访问 `http://localhost:8037/` 和 `http://localhost:8037/admin`。

## 风险与后续建议

1. 当前 admin 前端已覆盖核心管理能力，复杂表单校验仍以后端响应为准。
2. 导入导出和批量删除属于高风险操作，当前已使用确认弹窗；后续可增加更详细的预览和二次确认文案。
3. 任务 09 需要补充最终验收清单、部署文档和端到端操作说明。
