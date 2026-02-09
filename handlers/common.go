package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"ntp/models"
)

// ReorderItem 排序项 (导出 models 中的类型)
type ReorderItem = models.ReorderItem

// respondJSON 返回 JSON 响应
func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

// respondError 返回错误响应
func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}

// extractPathID 从路径中提取 ID (格式: /prefix/ID 或 /prefix/ID/action)
func extractPathID(r *http.Request, prefix string) int64 {
	path := strings.TrimPrefix(r.URL.Path, prefix)
	parts := strings.Split(path, "/")
	if len(parts) > 0 && parts[0] != "" {
		id, _ := strconv.ParseInt(parts[0], 10, 64)
		return id
	}
	return 0
}
