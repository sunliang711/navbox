# Bug 修复交付：Docker 上传目录权限

## 问题背景

导入 zip 时服务返回 500，业务日志显示：

```text
write import icon file: open /app/data/uploads/<icon>.ico: permission denied
```

## 根因分析

Docker 镜像构建阶段虽然对 `/app/data/uploads` 做了 `chown`，但 `docker-compose.yml` 运行时使用 `./data/uploads:/app/data/uploads` bind mount 覆盖了镜像内目录。宿主机目录权限未必允许容器内 `navbox` 用户写入，导致导入新 icon 文件时失败。

## 修复方案

1. 新增 `docker-entrypoint.sh`。
2. 容器启动时先创建并 `chown` `NAVBOX_UPLOAD_DIR`。
3. 使用 `su-exec` 降权到 `navbox` 用户运行 `/app/navbox`。
4. 保持服务进程不以 root 长期运行。

## 文件变更

- `Dockerfile`
- `docker-entrypoint.sh`

## 验证结果

已执行：

```text
sh -n docker-entrypoint.sh
docker compose up -d --build navbox
docker compose exec -T --user navbox navbox sh -lc 'id && touch /app/data/uploads/.navbox-write-test && rm /app/data/uploads/.navbox-write-test'
docker compose exec -T navbox sh -lc 'ps -o user,pid,comm,args | grep /app/navbox | grep -v grep'
curl -sS http://localhost:8037/api/v1/health
```

验证结果：

1. entrypoint 脚本语法检查通过。
2. Docker 镜像重建并启动成功。
3. `navbox` 用户可以写入 `/app/data/uploads`。
4. 主服务进程用户为 `navbox`。
5. 健康检查返回 200。

## 风险与后续建议

该修复要求镜像启动入口使用 `navbox-entrypoint`。如果用户自定义 `entrypoint`，需要自行保证 `NAVBOX_UPLOAD_DIR` 对运行用户可写。
