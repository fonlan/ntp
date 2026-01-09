package models

import (
	"database/sql"
	"time"
)

// Bookmark 书签
type Bookmark struct {
	ID          int64      `json:"id"`
	Title       string     `json:"title"`
	URL         string     `json:"url"`
	IconURL     *string    `json:"icon_url"`
	IconPath    *string    `json:"icon_path"`
	Description *string    `json:"description"`
	GroupID     *int64     `json:"group_id"`
	SortOrder   int        `json:"sort_order"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// BookmarkRepository 书签数据访问层
type BookmarkRepository struct {
	db *sql.DB
}

// NewBookmarkRepository 创建书签仓库
func NewBookmarkRepository(db *sql.DB) *BookmarkRepository {
	return &BookmarkRepository{db: db}
}

// Create 创建书签
func (r *BookmarkRepository) Create(bookmark *Bookmark) error {
	result, err := r.db.Exec(
		`INSERT INTO bookmarks (title, url, icon_url, icon_path, description, group_id, sort_order)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		bookmark.Title, bookmark.URL, bookmark.IconURL, bookmark.IconPath,
		bookmark.Description, bookmark.GroupID, bookmark.SortOrder,
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

// GetByID 根据 ID 获取书签
func (r *BookmarkRepository) GetByID(id int64) (*Bookmark, error) {
	bookmark := &Bookmark{}
	err := r.db.QueryRow(
		`SELECT id, title, url, icon_url, icon_path, description, group_id, sort_order, created_at, updated_at
		 FROM bookmarks WHERE id = ?`,
		id,
	).Scan(
		&bookmark.ID, &bookmark.Title, &bookmark.URL, &bookmark.IconURL, &bookmark.IconPath,
		&bookmark.Description, &bookmark.GroupID, &bookmark.SortOrder,
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
			`SELECT id, title, url, icon_url, icon_path, description, group_id, sort_order, created_at, updated_at
			 FROM bookmarks ORDER BY sort_order ASC`,
		)
	} else {
		rows, err = r.db.Query(
			`SELECT id, title, url, icon_url, icon_path, description, group_id, sort_order, created_at, updated_at
			 FROM bookmarks WHERE group_id = ? ORDER BY sort_order ASC`,
			*groupID,
		)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bookmarks []Bookmark
	for rows.Next() {
		var b Bookmark
		if err := rows.Scan(
			&b.ID, &b.Title, &b.URL, &b.IconURL, &b.IconPath,
			&b.Description, &b.GroupID, &b.SortOrder,
			&b.CreatedAt, &b.UpdatedAt,
		); err != nil {
			return nil, err
		}
		bookmarks = append(bookmarks, b)
	}

	return bookmarks, nil
}

// Update 更新书签
func (r *BookmarkRepository) Update(bookmark *Bookmark) error {
	_, err := r.db.Exec(
		`UPDATE bookmarks
		 SET title = ?, url = ?, icon_url = ?, icon_path = ?, description = ?, group_id = ?, sort_order = ?, updated_at = CURRENT_TIMESTAMP
		 WHERE id = ?`,
		bookmark.Title, bookmark.URL, bookmark.IconURL, bookmark.IconPath,
		bookmark.Description, bookmark.GroupID, bookmark.SortOrder,
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

// Search 搜索书签
func (r *BookmarkRepository) Search(query string) ([]Bookmark, error) {
	rows, err := r.db.Query(
		`SELECT id, title, url, icon_url, icon_path, description, group_id, sort_order, created_at, updated_at
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
			&b.ID, &b.Title, &b.URL, &b.IconURL, &b.IconPath,
			&b.Description, &b.GroupID, &b.SortOrder,
			&b.CreatedAt, &b.UpdatedAt,
		); err != nil {
			return nil, err
		}
		bookmarks = append(bookmarks, b)
	}

	return bookmarks, nil
}
