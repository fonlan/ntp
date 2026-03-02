package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWithConditionalETagReturns304(t *testing.T) {
	handler := withConditionalETag(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[{"id":1}]`))
	}))

	firstReq := httptest.NewRequest(http.MethodGet, "/api/bookmarks", nil)
	firstResp := httptest.NewRecorder()
	handler.ServeHTTP(firstResp, firstReq)

	if firstResp.Code != http.StatusOK {
		t.Fatalf("expected first status %d, got %d", http.StatusOK, firstResp.Code)
	}

	etag := firstResp.Header().Get("ETag")
	if etag == "" {
		t.Fatal("expected ETag header on first response")
	}

	secondReq := httptest.NewRequest(http.MethodGet, "/api/bookmarks", nil)
	secondReq.Header.Set("If-None-Match", etag)
	secondResp := httptest.NewRecorder()
	handler.ServeHTTP(secondResp, secondReq)

	if secondResp.Code != http.StatusNotModified {
		t.Fatalf("expected second status %d, got %d", http.StatusNotModified, secondResp.Code)
	}
	if secondResp.Body.Len() != 0 {
		t.Fatalf("expected empty body for 304 response, got %q", secondResp.Body.String())
	}
}

func TestWithConditionalETagSkipsNonGET(t *testing.T) {
	handler := withConditionalETag(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/bookmarks", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, resp.Code)
	}
	if resp.Header().Get("ETag") != "" {
		t.Fatalf("expected no ETag for non-GET response, got %q", resp.Header().Get("ETag"))
	}
}

func TestIsIfNoneMatchHandlesWeakAndStrongTag(t *testing.T) {
	etag := `W/"abc123"`
	if !isIfNoneMatch(`"abc123"`, etag) {
		t.Fatal("expected strong tag to match weak ETag")
	}
	if !isIfNoneMatch(`W/"abc123"`, etag) {
		t.Fatal("expected weak tag to match weak ETag")
	}
	if isIfNoneMatch(`"zzz999"`, etag) {
		t.Fatal("expected non-matching tag to return false")
	}
}
