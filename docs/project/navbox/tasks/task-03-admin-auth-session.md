# 任务 03：admin 认证与 Session

## 目标

实现 admin 初始密码、登录、Session、退出和修改密码能力。

## 技术方案

1. 首次启动且未配置密码时，生成 16 位随机初始密码。
2. 初始密码打印英文日志。
3. admin 密码使用 bcrypt 存储。
4. 登录成功后生成 Session token，Cookie 返回 token，数据库只保存 token hash。
5. Session 默认有效期 24 小时。
6. 管理接口统一使用 Session Middleware 校验。

## 验收标准

1. 首次启动无密码时生成初始密码并打印英文日志。
2. admin 可登录并获得 Session Cookie。
3. 未登录访问管理接口返回未认证。
4. admin 可修改密码，旧密码失效。
5. admin 可退出登录，当前 Session 失效。

## 实现步骤

1. 实现密码生成和初始化逻辑。
2. 实现登录接口。
3. 实现 Session 创建、查询、删除。
4. 实现 Session Middleware。
5. 实现修改密码接口。
6. 补充认证相关测试。

## 前置依赖

任务 02。

## 风险

1. 不能在日志中输出密码哈希或 Session token。
2. Cookie 安全属性需要适配本地 HTTP 和生产 HTTPS。
