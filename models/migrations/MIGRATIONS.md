# 数据库迁移系统使用指南

## 概述

项目现在支持自动数据库迁移。每次程序启动时，系统会自动检测并执行所有待执行的数据库迁移。

## 迁移文件命名规则

迁移文件存放在 `models/migrations/` 目录下，文件名格式：

```
V{版本号}__{描述}.sql
```

例如：
- `V001__initial_schema.sql` - 初始数据库结构
- `V002__add_user_table.sql` - 添加用户表
- `V003__add_bookmark_tags.sql` - 为书签添加标签字段

**重要规则：**
- 版本号必须是数字，从 001 开始递增
- 描述使用下划线分隔单词
- 文件扩展名必须是 `.sql`
- 版本号和描述之间使用两个下划线 `__` 分隔

## 创建新迁移

当你需要修改数据库结构时（添加表、修改列、添加索引等）：

1. **创建迁移文件**
   ```bash
   # 在 models/migrations/ 目录下创建新文件
   # 文件名：V{下一个版本号}__{描述}.sql
   ```

2. **编写 SQL 语句**
   ```sql
   -- 示例：添加新列
   ALTER TABLE bookmarks ADD COLUMN tags TEXT DEFAULT '';

   -- 示例：创建新表
   CREATE TABLE IF NOT EXISTS tags (
       id INTEGER PRIMARY KEY AUTOINCREMENT,
       name TEXT NOT NULL UNIQUE,
       created_at DATETIME DEFAULT CURRENT_TIMESTAMP
   );

   -- 示例：添加索引
   CREATE INDEX IF NOT EXISTS idx_bookmarks_tags ON bookmarks(tags);
   ```

3. **测试迁移**
   ```bash
   # 删除现有数据库（注意：这会清空所有数据）
   rm data/bookmarks.db

   # 重新编译并运行
   go build -o ntp .
   ./ntp
   ```

4. **验证迁移**
   - 查看日志，确认迁移成功执行
   - 检查数据库结构是否符合预期

## 迁移执行机制

- **自动执行**：程序启动时自动执行所有未执行的迁移
- **幂等性**：已执行的迁移会被自动跳过
- **事务支持**：每个迁移在独立事务中执行，失败会自动回滚
- **版本跟踪**：迁移记录存储在 `schema_migrations` 表中

## 查看迁移状态

```bash
# 使用 sqlite3 查看已执行的迁移
sqlite3 data/bookmarks.db "SELECT * FROM schema_migrations ORDER BY version;"
```

## 迁移最佳实践

1. **向后兼容**：尽量避免破坏性更改（如删除列）
2. **数据迁移**：如果需要迁移数据，在同一迁移文件中完成
3. **测试完整流程**：
   - 在空数据库上测试（新用户场景）
   - 在旧数据库上测试（升级场景）
4. **版本号递增**：始终使用下一个可用的版本号
5. **清晰描述**：文件描述应清楚说明迁移的目的

## 示例：添加新字段

假设你要为书签添加"访问次数"字段：

1. 创建文件 `models/migrations/V002__add_visit_count.sql`：
   ```sql
   -- 添加访问次数字段
   ALTER TABLE bookmarks ADD COLUMN visit_count INTEGER DEFAULT 0;
   ```

2. 重新编译并运行，系统会自动执行迁移

## Docker 部署

迁移系统在 Docker 环境中同样有效：

```bash
# 构建镜像（迁移文件已嵌入二进制）
docker build -t ntp .

# 运行容器（首次启动自动执行所有迁移）
docker-compose up -d
```

数据库会在容器首次启动时自动初始化，后续升级只需重新构建镜像即可。

## 故障排查

如果迁移失败：

1. 查看错误日志，定位问题 SQL 语句
2. 修复迁移文件
3. 删除数据库重新测试（开发环境）
4. 或手动修复数据库（生产环境需谨慎）

```sql
-- 手动标记迁移为已执行（仅用于故障恢复）
INSERT INTO schema_migrations (version) VALUES ('002');
```
