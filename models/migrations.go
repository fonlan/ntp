package models

import (
	"database/sql"
	"embed"
	"fmt"
	"log"
	"sort"
	"strings"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

// Migration 代表一个数据库迁移
type Migration struct {
	Version string
	Name    string
	SQL     string
}

// loadMigrations 从嵌入的文件系统加载所有迁移文件
func loadMigrations() ([]Migration, error) {
	entries, err := migrationFS.ReadDir("migrations")
	if err != nil {
		return nil, fmt.Errorf("读取迁移目录失败: %w", err)
	}

	var migrations []Migration
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		// 解析文件名: V001__description.sql
		filename := entry.Name()
		parts := strings.SplitN(strings.TrimSuffix(filename, ".sql"), "__", 2)
		if len(parts) != 2 || !strings.HasPrefix(parts[0], "V") {
			log.Printf("警告: 跳过无效的迁移文件名: %s", filename)
			continue
		}

		version := parts[0][1:] // 去掉 'V' 前缀
		name := parts[1]

		content, err := migrationFS.ReadFile("migrations/" + filename)
		if err != nil {
			return nil, fmt.Errorf("读取迁移文件 %s 失败: %w", filename, err)
		}

		migrations = append(migrations, Migration{
			Version: version,
			Name:    name,
			SQL:     string(content),
		})
	}

	// 按版本号排序
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	return migrations, nil
}

// RunMigrations 执行所有待执行的数据库迁移
func RunMigrations(db *sql.DB) error {
	// 创建迁移记录表
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version TEXT PRIMARY KEY,
			applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		return fmt.Errorf("创建迁移表失败: %w", err)
	}

	// 获取已执行的迁移
	appliedVersions := make(map[string]bool)
	rows, err := db.Query("SELECT version FROM schema_migrations")
	if err != nil {
		return fmt.Errorf("查询迁移记录失败: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var version string
		if err := rows.Scan(&version); err != nil {
			return fmt.Errorf("扫描迁移记录失败: %w", err)
		}
		appliedVersions[version] = true
	}

	// 加载所有迁移文件
	migrations, err := loadMigrations()
	if err != nil {
		return err
	}

	// 执行未执行的迁移
	for _, migration := range migrations {
		if appliedVersions[migration.Version] {
			log.Printf("迁移 V%s 已执行，跳过", migration.Version)
			continue
		}

		log.Printf("执行迁移 V%s: %s", migration.Version, migration.Name)

		// 在事务中执行迁移
		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("开始事务失败: %w", err)
		}

		// 执行迁移 SQL
		if _, err := tx.Exec(migration.SQL); err != nil {
			tx.Rollback()
			return fmt.Errorf("执行迁移 V%s 失败: %w", migration.Version, err)
		}

		// 记录迁移
		if _, err := tx.Exec("INSERT INTO schema_migrations (version) VALUES (?)", migration.Version); err != nil {
			tx.Rollback()
			return fmt.Errorf("记录迁移 V%s 失败: %w", migration.Version, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("提交迁移 V%s 失败: %w", migration.Version, err)
		}

		log.Printf("迁移 V%s 执行成功", migration.Version)
	}

	log.Println("数据库迁移完成")
	return nil
}
