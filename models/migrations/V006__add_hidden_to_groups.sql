-- 分组表增加 hidden 字段，用于支持隐藏分组
ALTER TABLE groups ADD COLUMN hidden BOOLEAN NOT NULL DEFAULT 0;
