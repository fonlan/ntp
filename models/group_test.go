package models

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func setupGroupRepositoryTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}

	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Fatalf("close test database: %v", err)
		}
	})

	if _, err := db.Exec(`
		PRAGMA foreign_keys=ON;

		CREATE TABLE groups (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE,
			sort_order INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE bookmarks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			title TEXT NOT NULL,
			url TEXT NOT NULL,
			group_id INTEGER,
			sort_order INTEGER NOT NULL DEFAULT 0,
			FOREIGN KEY (group_id) REFERENCES groups(id) ON DELETE SET NULL
		);
	`); err != nil {
		t.Fatalf("create test schema: %v", err)
	}

	return db
}

func seedGroupWithBookmark(t *testing.T, db *sql.DB) int64 {
	t.Helper()

	result, err := db.Exec("INSERT INTO groups (name, sort_order) VALUES (?, ?)", "Dev", 0)
	if err != nil {
		t.Fatalf("insert group: %v", err)
	}

	groupID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("read group id: %v", err)
	}

	if _, err := db.Exec("INSERT INTO bookmarks (title, url, group_id) VALUES (?, ?, ?)", "Go", "https://go.dev", groupID); err != nil {
		t.Fatalf("insert bookmark: %v", err)
	}

	return groupID
}

func TestGroupRepositoryDeleteWithBookmarks_keepsBookmarksUngrouped(t *testing.T) {
	db := setupGroupRepositoryTestDB(t)
	repo := NewGroupRepository(db)
	groupID := seedGroupWithBookmark(t, db)

	if err := repo.DeleteWithBookmarks(groupID, false); err != nil {
		t.Fatalf("delete group: %v", err)
	}

	var bookmarkCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM bookmarks WHERE group_id IS NULL").Scan(&bookmarkCount); err != nil {
		t.Fatalf("count ungrouped bookmarks: %v", err)
	}

	if bookmarkCount != 1 {
		t.Fatalf("expected 1 ungrouped bookmark, got %d", bookmarkCount)
	}
}

func TestGroupRepositoryDeleteWithBookmarks_deletesAssociatedBookmarks(t *testing.T) {
	db := setupGroupRepositoryTestDB(t)
	repo := NewGroupRepository(db)
	groupID := seedGroupWithBookmark(t, db)

	if err := repo.DeleteWithBookmarks(groupID, true); err != nil {
		t.Fatalf("delete group with bookmarks: %v", err)
	}

	var bookmarkCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM bookmarks").Scan(&bookmarkCount); err != nil {
		t.Fatalf("count bookmarks: %v", err)
	}

	if bookmarkCount != 0 {
		t.Fatalf("expected associated bookmarks to be deleted, got %d", bookmarkCount)
	}
}
