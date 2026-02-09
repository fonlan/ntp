package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"

	"ntp/middleware"
	"ntp/models"
	"ntp/services"
)

// SearchEngineHandler 搜索引擎处理器
type SearchEngineHandler struct {
	searchEngineRepo *models.SearchEngineRepository
	iconService      *services.IconService
}

// NewSearchEngineHandler 创建搜索引擎处理器
func NewSearchEngineHandler(ser *models.SearchEngineRepository, is *services.IconService) *SearchEngineHandler {
	return &SearchEngineHandler{searchEngineRepo: ser, iconService: is}
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
	translator := middleware.TranslatorFromContext(r.Context())

	engines, err := h.searchEngineRepo.GetAll()
	if err != nil {
		respondError(w, http.StatusInternalServerError, translator.T("searchEngine.listFailed")+": "+err.Error())
		return
	}

	respondJSON(w, http.StatusOK, engines)
}

// Create 创建搜索引擎
func (h *SearchEngineHandler) Create(w http.ResponseWriter, r *http.Request) {
	translator := middleware.TranslatorFromContext(r.Context())

	var req SearchEngineCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, translator.T("common.invalidRequest"))
		return
	}

	// 自动下载图标
	iconPath, _ := h.iconService.DownloadFavicon(req.URL)
	var iconPathPtr *string
	if iconPath != "" {
		iconPathPtr = &iconPath
	}

	// 默认引擎由 SetDefault 统一维护，避免出现多个默认引擎。
	engine := &models.SearchEngine{
		Name:        req.Name,
		URL:         req.URL,
		Placeholder: req.Placeholder,
		IconPath:    iconPathPtr,
		IsDefault:   false,
		SortOrder:   req.SortOrder,
	}

	if err := h.searchEngineRepo.Create(engine); err != nil {
		respondError(w, http.StatusInternalServerError, translator.T("searchEngine.createFailed")+": "+err.Error())
		return
	}

	if req.IsDefault {
		if err := h.searchEngineRepo.SetDefault(engine.ID); err != nil {
			respondError(w, http.StatusInternalServerError, translator.T("searchEngine.setDefaultFailed")+": "+err.Error())
			return
		}
		engine.IsDefault = true
	}

	respondJSON(w, http.StatusCreated, engine)
}

// Update 更新搜索引擎
func (h *SearchEngineHandler) Update(w http.ResponseWriter, r *http.Request) {
	translator := middleware.TranslatorFromContext(r.Context())
	id := extractPathID(r, "/api/search-engine/")

	engine, err := h.searchEngineRepo.GetByID(id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, translator.T("searchEngine.updateFailed")+": "+err.Error())
		return
	}
	if engine == nil {
		respondError(w, http.StatusNotFound, translator.T("searchEngine.notFound"))
		return
	}

	var req SearchEngineUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, translator.T("common.invalidRequest"))
		return
	}

	// 仅当用户显式选择“设为默认”时才执行切换；不允许通过 false 清空默认。
	setDefault := req.IsDefault != nil && *req.IsDefault

	if req.Name != nil {
		engine.Name = *req.Name
	}
	if req.URL != nil {
		engine.URL = *req.URL
		// 如果 URL 改变，重新下载图标
		iconPath, _ := h.iconService.DownloadFavicon(engine.URL)
		if iconPath != "" {
			engine.IconPath = &iconPath
		}
	}
	if req.Placeholder != nil {
		engine.Placeholder = *req.Placeholder
	}
	if req.SortOrder != nil {
		engine.SortOrder = *req.SortOrder
	}

	if err := h.searchEngineRepo.Update(engine); err != nil {
		respondError(w, http.StatusInternalServerError, translator.T("searchEngine.updateFailed")+": "+err.Error())
		return
	}

	if setDefault {
		if err := h.searchEngineRepo.SetDefault(engine.ID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				respondError(w, http.StatusNotFound, translator.T("searchEngine.notFound"))
				return
			}
			respondError(w, http.StatusInternalServerError, translator.T("searchEngine.setDefaultFailed")+": "+err.Error())
			return
		}
		engine.IsDefault = true
	}

	respondJSON(w, http.StatusOK, engine)
}

// Delete 删除搜索引擎
func (h *SearchEngineHandler) Delete(w http.ResponseWriter, r *http.Request) {
	translator := middleware.TranslatorFromContext(r.Context())
	id := extractPathID(r, "/api/search-engine/")

	if err := h.searchEngineRepo.Delete(id); err != nil {
		switch {
		case errors.Is(err, models.ErrCannotDeleteLastSearchEngine):
			respondError(w, http.StatusBadRequest, translator.T("searchEngine.cannotDeleteLast"))
		case errors.Is(err, sql.ErrNoRows):
			respondError(w, http.StatusNotFound, translator.T("searchEngine.notFound"))
		default:
			respondError(w, http.StatusInternalServerError, translator.T("searchEngine.deleteFailed")+": "+err.Error())
		}
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": translator.T("searchEngine.deleteSuccess")})
}

// SetDefault 设置默认引擎（POST /api/search-engine/{id}）
func (h *SearchEngineHandler) SetDefault(w http.ResponseWriter, r *http.Request) {
	translator := middleware.TranslatorFromContext(r.Context())
	id := extractPathID(r, "/api/search-engine/")
	if id <= 0 {
		respondError(w, http.StatusBadRequest, translator.T("common.invalidID"))
		return
	}

	if err := h.searchEngineRepo.SetDefault(id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			respondError(w, http.StatusNotFound, translator.T("searchEngine.notFound"))
			return
		}
		respondError(w, http.StatusInternalServerError, translator.T("searchEngine.setDefaultFailed")+": "+err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": translator.T("searchEngine.setDefaultSuccess")})
}

// Reorder 批量排序
func (h *SearchEngineHandler) Reorder(w http.ResponseWriter, r *http.Request) {
	translator := middleware.TranslatorFromContext(r.Context())

	var req SearchEngineReorderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, translator.T("common.invalidRequest"))
		return
	}

	items := make([]models.ReorderItem, len(req))

	for i, item := range req {
		items[i].ID = item.ID
		items[i].SortOrder = item.SortOrder
	}

	if err := h.searchEngineRepo.BatchReorder(items); err != nil {
		respondError(w, http.StatusInternalServerError, translator.T("searchEngine.reorderFailed")+": "+err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": translator.T("searchEngine.reorderSuccess")})
}
