package models

import "testing"

func TestGetBookmarksByURLs_emptyOrDuplicate(t *testing.T) {
	r := &BookmarkRepository{}

	got, err := r.GetBookmarksByURLs(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil map for nil input")
	}

	got, err = r.GetBookmarksByURLs([]string{"", "", ""})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil map for empty urls")
	}
}

