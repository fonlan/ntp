package models

import (
	"database/sql"
	"embed"
	"fmt"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

//go:embed schema.sql
var schemaFS embed.FS

var DB *sql.DB

// InitDB 初始化数据库连接和表结构
func InitDB(dataSourceName string) error {
	var err error
	DB, err = sql.Open("sqlite3", dataSourceName)
	if err != nil {
		return fmt.Errorf("打开数据库失败: %w", err)
	}

	// 设置 SQLite 为 WAL 模式以提升并发性能
	if _, err := DB.Exec("PRAGMA journal_mode=WAL; PRAGMA foreign_keys=ON;"); err != nil {
		return fmt.Errorf("设置 SQLite 模式失败: %w", err)
	}

	// 读取并执行 schema.sql
	schema, err := schemaFS.ReadFile("schema.sql")
	if err != nil {
		return fmt.Errorf("读取 schema 失败: %w", err)
	}

	if _, err := DB.Exec(string(schema)); err != nil {
		return fmt.Errorf("执行 schema 失败: %w", err)
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
