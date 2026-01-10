-- 书签分组表
CREATE TABLE IF NOT EXISTS groups (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- 书签表
CREATE TABLE IF NOT EXISTS bookmarks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    title TEXT NOT NULL,
    url TEXT NOT NULL,
    icon_url TEXT,
    icon_path TEXT,
    description TEXT,
    group_id INTEGER,
    sort_order INTEGER NOT NULL DEFAULT 0,
    is_new_window BOOLEAN DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (group_id) REFERENCES groups(id) ON DELETE SET NULL
);

-- 搜索引擎配置表
CREATE TABLE IF NOT EXISTS search_engines (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    url TEXT NOT NULL,
    placeholder TEXT NOT NULL,
    is_default BOOLEAN DEFAULT 0,
    sort_order INTEGER DEFAULT 0
);

-- 索引优化
CREATE INDEX IF NOT EXISTS idx_bookmarks_group ON bookmarks(group_id);
CREATE INDEX IF NOT EXISTS idx_bookmarks_sort ON bookmarks(sort_order);

-- 插入默认搜索引擎
INSERT OR IGNORE INTO search_engines (name, url, placeholder, is_default, sort_order) VALUES
    ('百度', 'https://www.baidu.com/s?wd={q}', '百度一下', 1, 0),
    ('Google', 'https://www.google.com/search?q={q}', 'Google Search', 0, 1),
    ('必应', 'https://www.bing.com/search?q={q}', '微软必应', 0, 2),
    ('GitHub', 'https://github.com/search?q={q}', '搜索代码库', 0, 3),
    ('Stack Overflow', 'https://stackoverflow.com/search?q={q}', '搜索技术问题', 0, 4);
