package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"ntp/models"
)

// SearchEngineHandler 搜索引擎处理器
type SearchEngineHandler struct {
	searchEngineRepo *models.SearchEngineRepository
}

// NewSearchEngineHandler 创建搜索引擎处理器
func NewSearchEngineHandler(ser *models.SearchEngineRepository) *SearchEngineHandler {
	return &SearchEngineHandler{searchEngineRepo: ser}
}

// SearchEngineCreateRequest 创建请求
type SearchEngineCreateRequest struct {
	Name        string `json:"name"`
	URL         string `json:"url"`
	Placeholder string `json:"placeholder"`
	IsDefault   bool   `json:"is_default"`
	SortOrder   int    `json:"sort_order,omitempty"`
}

// SearchEngineUpdateRequest 更新请求
type SearchEngineUpdateRequest struct {
	Name        *string `json:"name,omitempty"`
	URL         *string `json:"url,omitempty"`
	Placeholder *string `json:"placeholder,omitempty"`
	IsDefault   *bool   `json:"is_default,omitempty"`
	SortOrder   *int    `json:"sort_order,omitempty"`
}

// SearchEngineReorderRequest 排序请求
type SearchEngineReorderRequest []models.ReorderItem

// List 获取搜索引擎列表
func (h *SearchEngineHandler) List(w http.ResponseWriter, r *http.Request) {
	engines, err := h.searchEngineRepo.GetAll()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "获取搜索引擎列表失败: "+err.Error())
		return
	}

	respondJSON(w, http.StatusOK, engines)
}

// Create 创建搜索引擎
func (h *SearchEngineHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req SearchEngineCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "无效的请求格式")
		return
	}

	engine := &models.SearchEngine{
		Name:        req.Name,
		URL:         req.URL,
		Placeholder: req.Placeholder,
		IsDefault:   req.IsDefault,
		SortOrder:   req.SortOrder,
	}

	// 如果设置为默认，先清除其他默认标记
	if engine.IsDefault {
		h.searchEngineRepo.SetDefault(0) // 清除所有默认
	}

	if err := h.searchEngineRepo.Create(engine); err != nil {
		respondError(w, http.StatusInternalServerError, "创建搜索引擎失败: "+err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, engine)
}

// Update 更新搜索引擎
func (h *SearchEngineHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := extractPathID(r, "/api/search-engines/")

	engine, err := h.searchEngineRepo.GetByID(id)
	if err != nil || engine == nil {
		respondError(w, http.StatusNotFound, "搜索引擎不存在")
		return
	}

	var req SearchEngineUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "无效的请求格式")
		return
	}

	if req.Name != nil {
		engine.Name = *req.Name
	}
	if req.URL != nil {
		engine.URL = *req.URL
	}
	if req.Placeholder != nil {
		engine.Placeholder = *req.Placeholder
	}
	if req.IsDefault != nil {
		engine.IsDefault = *req.IsDefault
		if engine.IsDefault {
			h.searchEngineRepo.SetDefault(0) // 清除其他默认
		}
	}
	if req.SortOrder != nil {
		engine.SortOrder = *req.SortOrder
	}

	if err := h.searchEngineRepo.Update(engine); err != nil {
		respondError(w, http.StatusInternalServerError, "更新搜索引擎失败: "+err.Error())
		return
	}

	respondJSON(w, http.StatusOK, engine)
}

// Delete 删除搜索引擎
func (h *SearchEngineHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := extractPathID(r, "/api/search-engines/")

	if err := h.searchEngineRepo.Delete(id); err != nil {
		if err.Error() == "sql: transaction is already closed" || err.Error() == "sql: Tx is closed" {
			respondError(w, http.StatusBadRequest, "不能删除最后一个搜索引擎")
		} else {
			respondError(w, http.StatusInternalServerError, "删除搜索引擎失败: "+err.Error())
		}
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "删除成功"})
}

// SetDefault 设置默认引擎
func (h *SearchEngineHandler) SetDefault(w http.ResponseWriter, r *http.Request) {
	// 路径格式: /api/search-engines/:id/set-default
	path := strings.TrimPrefix(r.URL.Path, "/api/search-engines/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		respondError(w, http.StatusBadRequest, "无效的路径")
		return
	}

	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "无效的 ID")
		return
	}

	if err := h.searchEngineRepo.SetDefault(id); err != nil {
		respondError(w, http.StatusInternalServerError, "设置默认引擎失败: "+err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "设置成功"})
}

// Reorder 批量排序
func (h *SearchEngineHandler) Reorder(w http.ResponseWriter, r *http.Request) {
	var req SearchEngineReorderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "无效的请求格式")
		return
	}

	items := make([]models.ReorderItem, len(req))

	for i, item := range req {
		items[i].ID = item.ID
		items[i].SortOrder = item.SortOrder
	}

	if err := h.searchEngineRepo.BatchReorder(items); err != nil {
		respondError(w, http.StatusInternalServerError, "排序失败: "+err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "排序成功"})
}
