# 任务交付：前端首页与游客体验

## 任务背景

根据 `docs/project/navbox/tasks/task-07-frontend-public.md`，实现游客可用首页，包括 Tag 筛选、搜索、网站卡片、只读卡片操作和最近访问本地记录。

## 实现方案

1. 新增前端 API Client，调用公开只读接口获取 Site、Tag 和公开配置。
2. 首页初始化读取默认 Tag，并默认进入默认 Tag 视图。
3. 前端维护 Tag 多选状态，筛选请求交给后端公开 Site 接口处理。
4. 实现系统视图：全部、常用、最近、未分类。
5. 最近访问仅写入浏览器 `localStorage`，不调用服务端写接口。
6. 网站卡片支持文本 icon、上传图片 icon、常用标识、Tag 摘要、打开默认 URL、打开 LAN URL、复制链接。
7. 游客界面不展示 admin 管理入口。
8. 修正嵌入式首页服务逻辑，避免 `/` 访问出现 `Location: ./` 重定向。

## 文件变更

新增文件：

- `web/src/api.ts`
- `web/src/recent.ts`
- `web/src/types.ts`
- `internal/web/dist/assets/index-Dj2Vlwdt.js`
- `internal/web/dist/assets/index-D2cxMogA.css`

修改文件：

- `web/src/App.tsx`
- `web/src/styles.css`
- `internal/web/dist/index.html`
- `internal/server/router.go`
- `docs/project/navbox/PROGRESS.md`

## 游客能力

1. 查看网站卡片。
2. 查看 Tag 列表和网站数量。
3. 多选 Tag 筛选网站。
4. 使用关键字搜索网站。
5. 切换全部、常用、最近、未分类视图。
6. 打开默认 URL。
7. 打开 LAN URL。
8. 复制默认 URL。
9. 最近访问保存在本地。

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
docker compose up -d postgres navbox
docker compose exec -T navbox wget -qO- http://127.0.0.1:8080/api/v1/health
docker compose exec -T navbox wget -S -qO- http://127.0.0.1:8080/
docker compose exec -T navbox wget -qO- http://127.0.0.1:8080/api/v1/tags
docker compose exec -T navbox wget -qO- 'http://127.0.0.1:8080/api/v1/sites?view=default'
```

验证结果：

1. 前端 TypeScript 和 Vite 构建通过。
2. Go 测试和构建通过。
3. Docker 镜像构建通过。
4. Docker 服务已启动。
5. 嵌入式首页 `/` 返回 200，并正确引用构建后的 JS/CSS。
6. 公开 Tag 接口返回 200。
7. 默认视图 Site 查询返回 200。

## 风险与后续建议

1. 当前游客首页已具备只读能力；admin 管理界面将在任务 08 实现。
2. 最近访问依赖浏览器 `localStorage`，不同浏览器和设备之间不共享。
3. 复制链接依赖浏览器 Clipboard API，非安全上下文或浏览器限制时可能失败。
