-- 添加 icon_bg_color 字段到 bookmarks 表
-- 用于存储书签图标的自定义背景颜色（支持任意颜色包括透明）
ALTER TABLE bookmarks ADD COLUMN icon_bg_color TEXT DEFAULT '';
