# 任务 02：数据模型、数据库连接与迁移

## 目标

完成 PostgreSQL 连接、GORM Model 和基础迁移，使业务数据可以持久化。

## 技术方案

1. 使用 GORM 连接 PostgreSQL。
2. 配置连接池参数。
3. 定义 `Site`、`Tag`、`SiteTag`、`AdminSetting`、`AdminSession`、`Icon` Model。
4. 使用 AutoMigrate 完成 MVP 阶段迁移。
5. Repository 层封装基础查询和写入。

## 验收标准

1. 应用启动时可以连接 PostgreSQL。
2. 数据库表能自动创建。
3. 数据库连接失败时应用启动失败。
4. Repository 基础单元测试覆盖关键查询。

## 实现步骤

1. 定义数据库配置结构。
2. 实现 GORM 初始化和连接池配置。
3. 实现 Model。
4. 实现 AutoMigrate。
5. 实现 Site、Tag、Auth、Icon Repository。

## 前置依赖

任务 01。

## 风险

1. AutoMigrate 适合 MVP，后续复杂变更需要迁移工具。
2. UUID 生成策略需要统一。
