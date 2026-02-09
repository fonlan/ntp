package models

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// Bookmark 书签
type Bookmark struct {
	ID          int64     `json:"id"`
	Title       string    `json:"title"`
	URL         string    `json:"url"`
	IconURL     *string   `json:"icon_url"`
	IconPath    *string   `json:"icon_path"`
	IconChar    *string   `json:"icon_char"`
	IconBgColor *string   `json:"icon_bg_color"`
	Description *string   `json:"description"`
	GroupID     *int64    `json:"group_id"`
	SortOrder   int       `json:"sort_order"`
	IsNewWindow bool      `json:"is_new_window"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// BookmarkRepository 书签数据访问层
type BookmarkRepository struct {
	db *sql.DB
}

// NewBookmarkRepository 创建书签仓库
func NewBookmarkRepository(db *sql.DB) *BookmarkRepository {
	return &BookmarkRepository{db: db}
}

// GetDB 获取数据库连接（用于 Import 优化中的批量查询）
func (r *BookmarkRepository) GetDB() *sql.DB {
	return r.db
}

// Create 创建书签
func (r *BookmarkRepository) Create(bookmark *Bookmark) error {
	result, err := r.db.Exec(
		`INSERT INTO bookmarks (title, url, icon_url, icon_path, icon_char, icon_bg_color, description, group_id, sort_order, is_new_window)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		bookmark.Title, bookmark.URL, bookmark.IconURL, bookmark.IconPath, bookmark.IconChar, bookmark.IconBgColor,
		bookmark.Description, bookmark.GroupID, bookmark.SortOrder, bookmark.IsNewWindow,
	)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}

	bookmark.ID = id
	return nil
}

// BatchCreate 批量创建书签（用于 Import 优化）
func (r *BookmarkRepository) BatchCreate(bookmarks []*Bookmark) error {
	if len(bookmarks) == 0 {
		return nil
	}

	tx, err := r.db.Begin()
	if err != nil {
		return err
	}

	stmt, err := tx.Prepare(
		`INSERT INTO bookmarks (title, url, icon_url, icon_path, icon_char, icon_bg_color, description, group_id, sort_order, is_new_window, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		tx.Rollback()
		return err
	}
	defer stmt.Close()

	for _, b := range bookmarks {
		if _, err := stmt.Exec(
			b.Title, b.URL, b.IconURL, b.IconPath, b.IconChar, b.IconBgColor,
			b.Description, b.GroupID, b.SortOrder, b.IsNewWindow,
			b.CreatedAt, b.UpdatedAt,
		); err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}

// GetByID 根据 ID 获取书签
func (r *BookmarkRepository) GetByID(id int64) (*Bookmark, error) {
	bookmark := &Bookmark{}
	err := r.db.QueryRow(
		`SELECT id, title, url, icon_url, icon_path, icon_char, icon_bg_color, description, group_id, sort_order, is_new_window, created_at, updated_at
		 FROM bookmarks WHERE id = ?`,
		id,
	).Scan(
		&bookmark.ID, &bookmark.Title, &bookmark.URL, &bookmark.IconURL, &bookmark.IconPath, &bookmark.IconChar, &bookmark.IconBgColor,
		&bookmark.Description, &bookmark.GroupID, &bookmark.SortOrder, &bookmark.IsNewWindow,
		&bookmark.CreatedAt, &bookmark.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	return bookmark, err
}

// GetAll 获取所有书签
func (r *BookmarkRepository) GetAll(groupID *int64) ([]Bookmark, error) {
	var rows *sql.Rows
	var err error

	if groupID == nil {
		rows, err = r.db.Query(
			`SELECT id, title, url, icon_url, icon_path, icon_char, icon_bg_color, description, group_id, sort_order, is_new_window, created_at, updated_at
			 FROM bookmarks ORDER BY sort_order ASC`,
		)
	} else {
		rows, err = r.db.Query(
			`SELECT id, title, url, icon_url, icon_path, icon_char, icon_bg_color, description, group_id, sort_order, is_new_window, created_at, updated_at
			 FROM bookmarks WHERE group_id = ? ORDER BY sort_order ASC`,
			*groupID,
		)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// 预分配 slice 容量（基于典型数据大小优化）
	bookmarks := make([]Bookmark, 0, 100)
	for rows.Next() {
		var b Bookmark
		if err := rows.Scan(
			&b.ID, &b.Title, &b.URL, &b.IconURL, &b.IconPath, &b.IconChar, &b.IconBgColor,
			&b.Description, &b.GroupID, &b.SortOrder, &b.IsNewWindow,
			&b.CreatedAt, &b.UpdatedAt,
		); err != nil {
			return nil, err
		}
		bookmarks = append(bookmarks, b)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return bookmarks, nil
}

// Update 更新书签
func (r *BookmarkRepository) Update(bookmark *Bookmark) error {
	_, err := r.db.Exec(
		`UPDATE bookmarks
		 SET title = ?, url = ?, icon_url = ?, icon_path = ?, icon_char = ?, icon_bg_color = ?, description = ?, group_id = ?, sort_order = ?, is_new_window = ?, updated_at = CURRENT_TIMESTAMP
		 WHERE id = ?`,
		bookmark.Title, bookmark.URL, bookmark.IconURL, bookmark.IconPath, bookmark.IconChar, bookmark.IconBgColor,
		bookmark.Description, bookmark.GroupID, bookmark.SortOrder, bookmark.IsNewWindow,
		bookmark.ID,
	)
	return err
}

// Delete 删除书签
func (r *BookmarkRepository) Delete(id int64) error {
	_, err := r.db.Exec("DELETE FROM bookmarks WHERE id = ?", id)
	return err
}

// BatchReorder 批量排序
func (r *BookmarkRepository) BatchReorder(items []ReorderItem) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}

	stmt, err := tx.Prepare("UPDATE bookmarks SET sort_order = ? WHERE id = ?")
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

// GetMaxSortOrder 获取分组中最大的 sort_order 值
func (r *BookmarkRepository) GetMaxSortOrder(groupID *int64) (int, error) {
	var maxSortOrder int
	var err error

	if groupID == nil {
		err = r.db.QueryRow(
			"SELECT COALESCE(MAX(sort_order), -1) FROM bookmarks WHERE group_id IS NULL",
		).Scan(&maxSortOrder)
	} else {
		err = r.db.QueryRow(
			"SELECT COALESCE(MAX(sort_order), -1) FROM bookmarks WHERE group_id = ?",
			*groupID,
		).Scan(&maxSortOrder)
	}

	if err != nil {
		return 0, err
	}

	return maxSortOrder, nil
}

// Search 搜索书签
func (r *BookmarkRepository) Search(query string) ([]Bookmark, error) {
	rows, err := r.db.Query(
		`SELECT id, title, url, icon_url, icon_path, icon_char, icon_bg_color, description, group_id, sort_order, is_new_window, created_at, updated_at
		 FROM bookmarks
		 WHERE title LIKE ? OR url LIKE ? OR description LIKE ?
		 ORDER BY sort_order ASC`,
		"%"+query+"%", "%"+query+"%", "%"+query+"%",
	)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bookmarks []Bookmark
	for rows.Next() {
		var b Bookmark
		if err := rows.Scan(
			&b.ID, &b.Title, &b.URL, &b.IconURL, &b.IconPath, &b.IconChar, &b.IconBgColor,
			&b.Description, &b.GroupID, &b.SortOrder, &b.IsNewWindow,
			&b.CreatedAt, &b.UpdatedAt,
		); err != nil {
			return nil, err
		}
		bookmarks = append(bookmarks, b)
	}

	// 检查迭代过程中的错误
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return bookmarks, nil
}

// GetByURL 根据 URL 获取书签
func (r *BookmarkRepository) GetByURL(url string) (*Bookmark, error) {
	bookmark := &Bookmark{}
	err := r.db.QueryRow(
		`SELECT id, title, url, icon_url, icon_path, icon_char, icon_bg_color, description, group_id, sort_order, is_new_window, created_at, updated_at
		 FROM bookmarks WHERE url = ?`,
		url,
	).Scan(
		&bookmark.ID, &bookmark.Title, &bookmark.URL, &bookmark.IconURL, &bookmark.IconPath, &bookmark.IconChar, &bookmark.IconBgColor,
		&bookmark.Description, &bookmark.GroupID, &bookmark.SortOrder, &bookmark.IsNewWindow,
		&bookmark.CreatedAt, &bookmark.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	return bookmark, err
}

// GetBookmarksByURLs 根据 URL 列表批量查询书签（用于 Import 优化）
func (r *BookmarkRepository) GetBookmarksByURLs(urls []string) (map[string]*Bookmark, error) {
	if len(urls) == 0 {
		return nil, nil
	}

	// 去重后再查询，降低 SQL 变量数量，避免触发 SQLite 变量上限。
	seen := make(map[string]struct{}, len(urls))
	unique := make([]string, 0, len(urls))
	for _, u := range urls {
		if u == "" {
			continue
		}
		if _, ok := seen[u]; ok {
			continue
		}
		seen[u] = struct{}{}
		unique = append(unique, u)
	}
	if len(unique) == 0 {
		return nil, nil
	}

	bookmarkMap := make(map[string]*Bookmark)

	// SQLite 默认变量上限通常为 999，这里留出余量。
	const maxVarsPerQuery = 500

	for start := 0; start < len(unique); start += maxVarsPerQuery {
		end := start + maxVarsPerQuery
		if end > len(unique) {
			end = len(unique)
		}
		chunk := unique[start:end]

		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(chunk)), ",")
		query := fmt.Sprintf(
			"SELECT id, title, url, icon_url, icon_path, icon_char, icon_bg_color, description, group_id, sort_order, is_new_window, created_at, updated_at FROM bookmarks WHERE url IN (%s)",
			placeholders,
		)

		args := make([]interface{}, len(chunk))
		for i, u := range chunk {
			args[i] = u
		}

		rows, err := r.db.Query(query, args...)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			b := &Bookmark{}
			err := rows.Scan(
				&b.ID, &b.Title, &b.URL, &b.IconURL, &b.IconPath, &b.IconChar, &b.IconBgColor,
				&b.Description, &b.GroupID, &b.SortOrder, &b.IsNewWindow,
				&b.CreatedAt, &b.UpdatedAt,
			)
			if err != nil {
				rows.Close()
				return nil, err
			}
			bookmarkMap[b.URL] = b
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return nil, err
		}
		rows.Close()
	}

	return bookmarkMap, nil
}

// DeleteAll 删除所有书签
func (r *BookmarkRepository) DeleteAll() error {
	_, err := r.db.Exec("DELETE FROM bookmarks")
	return err
}
