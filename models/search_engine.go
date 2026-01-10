package models

import (
	"database/sql"
)

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
	// 检查是否是最后一个搜索引擎
	var count int
	r.db.QueryRow("SELECT COUNT(*) FROM search_engines").Scan(&count)
	if count <= 1 {
		return sql.ErrTxDone // 使用错误表示不能删除最后一个
	}

	_, err := r.db.Exec("DELETE FROM search_engines WHERE id = ?", id)
	return err
}

// SetDefault 设置默认搜索引擎
func (r *SearchEngineRepository) SetDefault(id int64) error {
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
	if _, err := tx.Exec("UPDATE search_engines SET is_default = 1 WHERE id = ?", id); err != nil {
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
