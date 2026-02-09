package models

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

var DB *sql.DB

// InitDB 初始化数据库连接并自动执行迁移
func InitDB(dataSourceName string) error {
	var err error
	DB, err = sql.Open("sqlite3", dataSourceName)
	if err != nil {
		return fmt.Errorf("打开数据库失败: %w", err)
	}

	// 验证数据库连接
	if err := DB.Ping(); err != nil {
		return fmt.Errorf("数据库连接失败: %w", err)
	}

	// 配置数据库连接池
	// SQLite 的部分 PRAGMA 是连接级别的（例如 foreign_keys、busy_timeout），为了确保配置生效并减少锁竞争，这里使用单连接。
	DB.SetMaxOpenConns(1)
	DB.SetMaxIdleConns(1)
	DB.SetConnMaxLifetime(0)
	DB.SetConnMaxIdleTime(0)

	// 设置 SQLite 为 WAL 模式以提升并发性能，并添加其他性能优化 PRAGMA
	if _, err := DB.Exec(`
		PRAGMA journal_mode=WAL;
		PRAGMA foreign_keys=ON;
		PRAGMA synchronous=NORMAL;
		PRAGMA cache_size=-64000;
		PRAGMA temp_store=MEMORY;
		PRAGMA mmap_size=268435456;
		PRAGMA page_size=4096;
		PRAGMA busy_timeout=5000;
	`); err != nil {
		return fmt.Errorf("设置 SQLite 模式失败: %w", err)
	}

	// 自动执行数据库迁移
	if err := RunMigrations(DB); err != nil {
		return fmt.Errorf("数据库迁移失败: %w", err)
	}

	log.Println("数据库初始化成功")
	return nil
}

// CloseDB 关闭数据库连接
func CloseDB() error {
	if DB != nil {
		return DB.Close()
	}
	return nil
}
