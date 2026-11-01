# 开发与评审交付：浏览器侧网站 icon 获取兜底

## 任务背景

部分目标网站位于后端无法访问的其他内网，但 admin 使用的浏览器可以访问该网络。原有“从网站获取”只由后端请求目标网站，此时即使浏览器能够打开网站，也无法自动保存其 icon。

本次在用户主动点击“从网站获取”时增加浏览器侧兜底，并增强后端 icon 候选发现能力。

## 需求边界

1. 仅在用户点击“从网站获取”且后端获取失败时，自动尝试浏览器侧获取。
2. 浏览器成功读取图标后，复用现有 `POST /api/v1/admin/icons/upload` 接口保存到后端。
3. 新增网站保存时，后端自动获取 icon 的触发条件和失败处理保持不变。
4. 不新增 API、配置项、数据库结构或第三方依赖。
5. 浏览器侧获取属于尽力而为，仍受目标网站 CORS 策略和浏览器 Mixed Content 策略限制。

## 编排与实现方案

本次按“主实现、独立评审、问题修复、独立复审、交付文档”的顺序完成。

### 后端获取增强

1. 保留页面 `<link rel="...icon">` 候选解析。
2. 解析页面 `<link rel="manifest">`，读取 Web App Manifest 中的 `icons[].src`。
3. 页面和 Manifest 请求在默认 `Navbox/1.0` 请求失败或返回非 2xx 时，使用浏览器 User-Agent 与 `Accept-Language` 再尝试一次。
4. 图标下载失败或内容无效时，使用浏览器请求头重试。
5. 增加以下常见路径候选：
   - `/favicon.ico`
   - `/favicon.png`
   - `/apple-touch-icon.png`
   - `/apple-touch-icon-precomposed.png`
6. 后端整体获取超时仍为 5 秒；HTML 最大读取 256 KiB，Manifest 最大读取 64 KiB，候选数量设置上限并去重。
7. 成功读取 2xx 页面但未声明图标时，不重复请求页面，直接进入常见路径回退，避免耗尽 5 秒预算。

### 浏览器侧兜底

1. 前端先调用现有后端获取接口；仅在非鉴权错误失败后启动浏览器兜底。
2. 浏览器尝试读取目标页面中的 icon 与 Manifest 声明，再追加常见路径候选。
3. 所有跨域请求使用 `mode: "cors"`、`credentials: "omit"`，不向目标内网站点发送用户凭据。
4. 仅接受 `http` / `https` URL，并拒绝带 UserInfo 的 URL。
5. 前端整体尝试时间上限为 10 秒，单次请求上限为 3 秒；超时覆盖响应头和完整响应体读取。
6. HTML 最大读取 256 KiB，Manifest 最大读取 64 KiB，图标最大读取 5 MiB；读取采用流式限长，超限或非 2xx 响应会主动取消 Body。
7. 最多尝试 12 个图标候选，其中页面声明最多 6 个、Manifest 候选最多 2 个，并为 4 个常见路径预留空间。
8. 图标 Blob 获取成功后封装为 `File`，复用现有上传接口；上传成功后继续写入表单的 `icon_type` 和 `icon_value`。

## 文件变更

新增文件：

- `web/src/browserIcon.ts`
- `docs/delivery/2026-11-01-dev-review-go-orchestrated-browser-icon-fallback.md`

修改文件：

- `web/src/AdminApp.tsx`
- `internal/service/favicon.go`
- `internal/service/favicon_test.go`

## 配置、依赖与数据变更

- 配置项：无新增或修改。
- 环境变量：无新增或修改。
- Go / Web 依赖：无新增或升级。
- 数据库表与数据迁移：无。
- HTTP API 契约：无变更，复用现有 icon 获取与上传接口。

## 安全与稳定性

1. 后端继续使用受限 HTTP Client，保留 URL Scheme、UserInfo、DNS 解析、私网 CIDR 白名单、特殊用途地址和重定向校验，未放宽 SSRF 防护。
2. 后端下载继续复用现有 icon MIME、内容安全、大小限制和 SHA256 去重逻辑。
3. 浏览器请求不携带目标站点 Cookie；CORS 不允许时无法读取响应体，兜底会失败并显示原有失败提示。
4. HTTPS 管理页面访问 HTTP 内网站点时可能被 Mixed Content 策略阻止，此限制无法由页面脚本绕过。
5. 前端响应体读取有超时和大小上限，避免慢速响应或异常大文件导致管理页面长期等待或无界占用内存。
6. admin Session 失效时不会继续浏览器兜底或吞掉 401，而是沿用现有未授权处理流程。

## 测试结果

已执行并通过：

```bash
go test ./...
go vet ./...
tsc --noEmit
vite build
git diff --check
```

后端新增测试覆盖：

1. 页面 icon 与 Manifest URL 解析。
2. Manifest icon 相对路径和绝对路径解析。
3. 浏览器请求头构造。
4. 页面声明、Manifest 候选与常见路径的候选顺序。
5. 默认请求被拒绝后使用浏览器请求头重试。
6. 2xx 页面没有声明 icon 时只请求一次，并进入 `/favicon.ico` 等路径回退。

## 独立评审与处理结果

### 首轮评审

发现以下问题：

1. 阻断：前端定时器只覆盖响应头，未覆盖响应体读取；图标使用无大小限制的 `response.blob()`，慢速或超大响应可能导致长时间等待和内存占用。
2. 警告：后端成功读取但没有声明 icon 的页面仍使用浏览器 User-Agent 重复请求，可能耗尽 5 秒总预算，导致 `/favicon.ico` 回退无法执行。
3. 建议：Manifest 元素只有 TypeScript 类型断言，异常数组元素可能导致整份候选解析失败。

### 修复措施

1. 将 `AbortController` 生命周期扩展到完整响应体消费，并改为流式限长读取；非 2xx 响应主动取消 Body。
2. 图标增加 5 MiB 浏览器侧读取上限。
3. 后端在首次 2xx 页面成功读取后立即返回解析结果，不因没有声明 icon 而重复请求，并补充回归测试。
4. Manifest 元素增加对象类型和 `src` 字符串运行时检查，跳过单个异常元素。

### 复审结论

独立 Agent 复审未发现新的阻断或警告问题，首轮问题均已关闭。

## 残余风险与后续建议

1. 前端浏览器抓取当前没有自动化测试，慢速流、Abort、超限、CORS 失败和上传 401 主要由实现审查、TypeScript 检查与生产构建保障。
2. 浏览器侧固定 5 MiB 安全上限不与后端可配置的 `upload.max_bytes` 联动；当后端允许超过 5 MiB 时，浏览器兜底仍会拒绝更大的图标。
3. CORS 和 Mixed Content 属于浏览器安全边界。若目标网站不允许跨域读取，可继续使用手动上传，或在目标网站侧配置合适的 CORS 响应头。
