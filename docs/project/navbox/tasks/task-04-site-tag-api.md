# 任务 04：Site / Tag 后端 API

## 目标

实现网站和 Tag 的核心后端 API，包括游客查询和 admin 管理。

## 技术方案

1. 公开接口提供 Site 列表、Tag 列表、公开配置。
2. admin 接口提供 Site CRUD、Tag CRUD、默认 Tag、常用、排序。
3. Site 与 Tag 使用多对多关系。
4. 多 Tag 筛选使用 AND 语义。
5. DTO 与 Model 分离。

## 验收标准

1. 游客可以查询网站和 Tag。
2. 游客不能调用写接口。
3. admin 可以新增、编辑、删除网站。
4. admin 可以新增、编辑、删除 Tag。
5. 多 Tag 筛选结果必须同时包含所有选中 Tag。
6. 删除 Tag 不删除网站。
7. 无 Tag 网站进入 `未分类` 视图。

## 实现步骤

1. 定义 Site / Tag DTO。
2. 实现 Site Service 和 Repository。
3. 实现 Tag Service 和 Repository。
4. 实现公开查询 Handler。
5. 实现 admin 管理 Handler。
6. 实现批量添加 Tag、批量移除 Tag、批量删除网站。
7. 实现排序接口。

## 前置依赖

任务 03。

## 风险

1. 多 Tag AND 查询需要保证性能和语义正确。
2. 批量删除需要事务保护。
