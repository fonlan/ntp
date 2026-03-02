package handlers

import (
	"net/http"

	"ntp/middleware"
	"ntp/models"
)

// InitialDataHandler 首屏聚合数据处理器
type InitialDataHandler struct {
	bookmarkRepo     *models.BookmarkRepository
	groupRepo        *models.GroupRepository
	searchEngineRepo *models.SearchEngineRepository
}

// InitialDataResponse 首屏聚合数据响应
type InitialDataResponse struct {
	Bookmarks     []models.Bookmark     `json:"bookmarks"`
	Groups        []models.Group        `json:"groups"`
	SearchEngines []models.SearchEngine `json:"search_engines"`
}

// NewInitialDataHandler 创建首屏聚合数据处理器
func NewInitialDataHandler(br *models.BookmarkRepository, gr *models.GroupRepository, sr *models.SearchEngineRepository) *InitialDataHandler {
	return &InitialDataHandler{
		bookmarkRepo:     br,
		groupRepo:        gr,
		searchEngineRepo: sr,
	}
}

// Get 获取首屏聚合数据
func (h *InitialDataHandler) Get(w http.ResponseWriter, r *http.Request) {
	translator := middleware.TranslatorFromContext(r.Context())

	bookmarks, err := h.bookmarkRepo.GetAll(nil)
	if err != nil {
		respondError(w, http.StatusInternalServerError, translator.T("common.internalError"))
		return
	}

	groups, err := h.groupRepo.GetAll()
	if err != nil {
		respondError(w, http.StatusInternalServerError, translator.T("common.internalError"))
		return
	}

	searchEngines, err := h.searchEngineRepo.GetAll()
	if err != nil {
		respondError(w, http.StatusInternalServerError, translator.T("common.internalError"))
		return
	}

	respondJSON(w, http.StatusOK, InitialDataResponse{
		Bookmarks:     bookmarks,
		Groups:        groups,
		SearchEngines: searchEngines,
	})
}
