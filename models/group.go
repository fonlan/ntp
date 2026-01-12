package models

import (
	"database/sql"
	"time"
)

// Group 书签分组
type Group struct {
	ID          int64          `json:"id"`
	Name        string         `json:"name"`
	SortOrder   int            `json:"sort_order"`
	CreatedAt   time.Time      `json:"created_at"`
	BookmarkCount int          `json:"bookmark_count"`
}

// GroupRepository 分组数据访问层
type GroupRepository struct {
	db *sql.DB
}

// NewGroupRepository 创建分组仓库
func NewGroupRepository(db *sql.DB) *GroupRepository {
	return &GroupRepository{db: db}
}

// Create 创建分组
func (r *GroupRepository) Create(group *Group) error {
	result, err := r.db.Exec(
		"INSERT INTO groups (name, sort_order) VALUES (?, ?)",
		group.Name, group.SortOrder,
	)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}

	group.ID = id
	return nil
}

// GetByID 根据 ID 获取分组
func (r *GroupRepository) GetByID(id int64) (*Group, error) {
	group := &Group{}
	err := r.db.QueryRow(
		"SELECT id, name, sort_order, created_at FROM groups WHERE id = ?",
		id,
	).Scan(&group.ID, &group.Name, &group.SortOrder, &group.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	return group, err
}

// GetAll 获取所有分组
func (r *GroupRepository) GetAll() ([]Group, error) {
	rows, err := r.db.Query(`
		SELECT g.id, g.name, g.sort_order, g.created_at,
		       (
			SELECT COUNT(*)
			FROM bookmarks b
			WHERE b.group_id = g.id
		       ) as bookmark_count
		FROM groups g
		ORDER BY g.sort_order ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	groups := []Group{}
	for rows.Next() {
		var g Group
		if err := rows.Scan(&g.ID, &g.Name, &g.SortOrder, &g.CreatedAt, &g.BookmarkCount); err != nil {
			return nil, err
		}
		groups = append(groups, g)
	}

	return groups, nil
}

// Update 更新分组
func (r *GroupRepository) Update(group *Group) error {
	_, err := r.db.Exec(
		"UPDATE groups SET name = ?, sort_order = ? WHERE id = ?",
		group.Name, group.SortOrder, group.ID,
	)
	return err
}

// Delete 删除分组
func (r *GroupRepository) Delete(id int64) error {
	_, err := r.db.Exec("DELETE FROM groups WHERE id = ?", id)
	return err
}

// BatchReorder 批量排序
func (r *GroupRepository) BatchReorder(items []ReorderItem) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}

	stmt, err := tx.Prepare("UPDATE groups SET sort_order = ? WHERE id = ?")
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

// DeleteAll 删除所有分组
func (r *GroupRepository) DeleteAll() error {
	_, err := r.db.Exec("DELETE FROM groups")
	return err
}
