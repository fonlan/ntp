package models

import (
	"database/sql"
	"errors"
	"fmt"
)

// ErrCannotDeleteLastSearchEngine 表示不能删除最后一个搜索引擎
var ErrCannotDeleteLastSearchEngine = errors.New("cannot delete the last search engine")

// SearchEngine 搜索引擎
type SearchEngine struct {
	ID          int64   `json:"id"`
	Name        string  `json:"name"`
	URL         string  `json:"url"`
	Placeholder string  `json:"placeholder"`
	IconPath    *string `json:"icon_path"`
	IsDefault   bool    `json:"is_default"`
	SortOrder   int     `json:"sort_order"`
}

// SearchEngineRepository 搜索引擎数据访问层
type SearchEngineRepository struct {
	db *sql.DB
}

// NewSearchEngineRepository 创建搜索引擎仓库
func NewSearchEngineRepository(db *sql.DB) *SearchEngineRepository {
	return &SearchEngineRepository{db: db}
}

// Create 创建搜索引擎
func (r *SearchEngineRepository) Create(engine *SearchEngine) error {
	result, err := r.db.Exec(
		"INSERT INTO search_engines (name, url, placeholder, icon_path, is_default, sort_order) VALUES (?, ?, ?, ?, ?, ?)",
		engine.Name, engine.URL, engine.Placeholder, engine.IconPath, engine.IsDefault, engine.SortOrder,
	)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}

	engine.ID = id
	return nil
}

// GetByID 根据 ID 获取搜索引擎
func (r *SearchEngineRepository) GetByID(id int64) (*SearchEngine, error) {
	engine := &SearchEngine{}
	err := r.db.QueryRow(
		"SELECT id, name, url, placeholder, icon_path, is_default, sort_order FROM search_engines WHERE id = ?",
		id,
	).Scan(&engine.ID, &engine.Name, &engine.URL, &engine.Placeholder, &engine.IconPath, &engine.IsDefault, &engine.SortOrder)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	return engine, err
}

// GetDefault 获取默认搜索引擎
func (r *SearchEngineRepository) GetDefault() (*SearchEngine, error) {
	engine := &SearchEngine{}
	err := r.db.QueryRow(
		"SELECT id, name, url, placeholder, icon_path, is_default, sort_order FROM search_engines WHERE is_default = 1 LIMIT 1",
	).Scan(&engine.ID, &engine.Name, &engine.URL, &engine.Placeholder, &engine.IconPath, &engine.IsDefault, &engine.SortOrder)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	return engine, err
}

// GetAll 获取所有搜索引擎
func (r *SearchEngineRepository) GetAll() ([]SearchEngine, error) {
	rows, err := r.db.Query(
		"SELECT id, name, url, placeholder, icon_path, is_default, sort_order FROM search_engines ORDER BY sort_order ASC",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	engines := []SearchEngine{}
	for rows.Next() {
		var e SearchEngine
		if err := rows.Scan(&e.ID, &e.Name, &e.URL, &e.Placeholder, &e.IconPath, &e.IsDefault, &e.SortOrder); err != nil {
			return nil, err
		}
		engines = append(engines, e)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return engines, nil
}

// Update 更新搜索引擎
func (r *SearchEngineRepository) Update(engine *SearchEngine) error {
	_, err := r.db.Exec(
		"UPDATE search_engines SET name = ?, url = ?, placeholder = ?, icon_path = ?, is_default = ?, sort_order = ? WHERE id = ?",
		engine.Name, engine.URL, engine.Placeholder, engine.IconPath, engine.IsDefault, engine.SortOrder, engine.ID,
	)
	return err
}

// Delete 删除搜索引擎
func (r *SearchEngineRepository) Delete(id int64) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}

	var count int
	if err := tx.QueryRow("SELECT COUNT(*) FROM search_engines").Scan(&count); err != nil {
		tx.Rollback()
		return err
	}
	if count <= 1 {
		tx.Rollback()
		return ErrCannotDeleteLastSearchEngine
	}

	var isDefault bool
	if err := tx.QueryRow("SELECT is_default FROM search_engines WHERE id = ?", id).Scan(&isDefault); err != nil {
		tx.Rollback()
		return err
	}

	// 如果删除的是默认引擎，需要先指定新的默认引擎，避免出现没有默认引擎的状态。
	if isDefault {
		var newDefaultID int64
		if err := tx.QueryRow(
			"SELECT id FROM search_engines WHERE id != ? ORDER BY sort_order ASC, id ASC LIMIT 1",
			id,
		).Scan(&newDefaultID); err != nil {
			tx.Rollback()
			return err
		}

		if _, err := tx.Exec("UPDATE search_engines SET is_default = 0"); err != nil {
			tx.Rollback()
			return err
		}
		if _, err := tx.Exec("UPDATE search_engines SET is_default = 1 WHERE id = ?", newDefaultID); err != nil {
			tx.Rollback()
			return err
		}
	}

	if _, err := tx.Exec("DELETE FROM search_engines WHERE id = ?", id); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

// SetDefault 设置默认搜索引擎
func (r *SearchEngineRepository) SetDefault(id int64) error {
	if id <= 0 {
		return fmt.Errorf("无效的搜索引擎 ID")
	}

	tx, err := r.db.Begin()
	if err != nil {
		return err
	}

	// 先清除所有默认标记
	if _, err := tx.Exec("UPDATE search_engines SET is_default = 0"); err != nil {
		tx.Rollback()
		return err
	}

	// 设置新的默认引擎
	result, err := tx.Exec("UPDATE search_engines SET is_default = 1 WHERE id = ?", id)
	if err != nil {
		tx.Rollback()
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		tx.Rollback()
		return err
	}
	if rows == 0 {
		tx.Rollback()
		return sql.ErrNoRows
	}

	return tx.Commit()
}

// NormalizeDefault 规范化默认引擎：保证存在且仅存在一个默认引擎
func (r *SearchEngineRepository) NormalizeDefault() error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}

	var id int64
	err = tx.QueryRow(
		"SELECT id FROM search_engines ORDER BY (is_default = 1) DESC, sort_order ASC, id ASC LIMIT 1",
	).Scan(&id)
	if err == sql.ErrNoRows {
		return tx.Commit()
	}
	if err != nil {
		tx.Rollback()
		return err
	}

	if _, err := tx.Exec("UPDATE search_engines SET is_default = CASE WHEN id = ? THEN 1 ELSE 0 END", id); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

// BatchReorder 批量排序
func (r *SearchEngineRepository) BatchReorder(items []ReorderItem) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}

	stmt, err := tx.Prepare("UPDATE search_engines SET sort_order = ? WHERE id = ?")
	if err != nil {
		tx.Rollback()
		return err
	}
	defer stmt.Close()

	for _, item := range items {
		if _, err := stmt.Exec(item.SortOrder, item.ID); err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}
