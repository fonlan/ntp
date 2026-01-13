-- V004__add_search_indexes.sql
-- 添加搜索字段索引以提升查询性能

-- 为 title 字段添加索引（前缀搜索有效）
CREATE INDEX IF NOT EXISTS idx_bookmarks_title ON bookmarks(title);

-- 为 url 字段添加索引（用于 Import Merge 模式）
CREATE INDEX IF NOT EXISTS idx_bookmarks_url ON bookmarks(url);

-- 为 description 字段添加索引（用于搜索）
CREATE INDEX IF NOT EXISTS idx_bookmarks_description ON bookmarks(description);
