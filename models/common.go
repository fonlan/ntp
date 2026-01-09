package models

// ReorderItem 排序项
type ReorderItem struct {
	ID        int64 `json:"id"`
	SortOrder int   `json:"sort_order"`
}
