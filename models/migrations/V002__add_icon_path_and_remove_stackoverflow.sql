-- 添加搜索引擎图标字段
ALTER TABLE search_engines ADD COLUMN icon_path TEXT;

-- 删除 Stack Overflow 搜索引擎
DELETE FROM search_engines WHERE name = 'Stack Overflow';
