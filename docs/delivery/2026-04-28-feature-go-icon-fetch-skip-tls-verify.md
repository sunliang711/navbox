# 功能开发：icon 获取支持跳过 TLS 证书校验

## 任务背景

`https://10.1.1.2:8006/` 使用 Proxmox 自签名证书，Go 默认 HTTPS 校验会因本地不信任该 CA 导致 icon 获取失败。页面实际存在可用 icon：`/pve2/images/logo-128.png`。

## 实现方案

新增 `icon_fetch.skip_tls_verify` 配置项，默认关闭。开启后仅影响后端“从网站获取 icon”的 HTTP client，不改变用户访问网站、站点列表展示、上传 icon 等其他链路。

## 文件变更

- `internal/config/config.go`
- `internal/service/icon.go`
- `internal/service/favicon.go`
- `internal/service/favicon_test.go`
- `config/config.toml`
- `docker-compose.yml`
- `README.md`
- `docs/deploy/2026-04-27-deploy-go-navbox.md`

## 配置与依赖变更

新增配置项：

```toml
[icon_fetch]
skip_tls_verify = false
```

新增环境变量：

```bash
NAVBOX_ICON_FETCH_SKIP_TLS_VERIFY=true
```

未新增 Go 依赖。

## 测试结果

已执行：

```bash
go test ./...
```

结果：全部通过。

## 风险与后续建议

1. `skip_tls_verify=true` 会跳过 icon 抓取请求的 HTTPS 证书校验，只建议在内网自签名证书站点场景开启。
2. 生产公网部署建议保持默认 `false`。
3. 更严格的长期方案是支持配置自定义 CA 文件，并信任 Proxmox CA。
