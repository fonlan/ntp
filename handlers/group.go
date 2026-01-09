package handlers

import (
	"encoding/json"
	"net/http"

	"ntp/models"
)

// GroupHandler 分组处理器
type GroupHandler struct {
	groupRepo *models.GroupRepository
}

// NewGroupHandler 创建分组处理器
func NewGroupHandler(gr *models.GroupRepository) *GroupHandler {
	return &GroupHandler{groupRepo: gr}
}

// GroupCreateRequest 创建请求
type GroupCreateRequest struct {
	Name      string `json:"name"`
	SortOrder int    `json:"sort_order,omitempty"`
}

// GroupUpdateRequest 更新请求
type GroupUpdateRequest struct {
	Name      *string `json:"name,omitempty"`
	SortOrder *int    `json:"sort_order,omitempty"`
}

// GroupReorderRequest 排序请求
type GroupReorderRequest []models.ReorderItem

// List 获取分组列表
func (h *GroupHandler) List(w http.ResponseWriter, r *http.Request) {
	groups, err := h.groupRepo.GetAll()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "获取分组列表失败: "+err.Error())
		return
	}

	respondJSON(w, http.StatusOK, groups)
}

// Create 创建分组
func (h *GroupHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req GroupCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "无效的请求格式")
		return
	}

	group := &models.Group{
		Name:      req.Name,
		SortOrder: req.SortOrder,
	}

	if err := h.groupRepo.Create(group); err != nil {
		respondError(w, http.StatusInternalServerError, "创建分组失败: "+err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, group)
}

// Update 更新分组
func (h *GroupHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := extractPathID(r, "/api/groups/")

	group, err := h.groupRepo.GetByID(id)
	if err != nil || group == nil {
		respondError(w, http.StatusNotFound, "分组不存在")
		return
	}

	var req GroupUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "无效的请求格式")
		return
	}

	if req.Name != nil {
		group.Name = *req.Name
	}
	if req.SortOrder != nil {
		group.SortOrder = *req.SortOrder
	}

	if err := h.groupRepo.Update(group); err != nil {
		respondError(w, http.StatusInternalServerError, "更新分组失败: "+err.Error())
		return
	}

	respondJSON(w, http.StatusOK, group)
}

// Delete 删除分组
func (h *GroupHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := extractPathID(r, "/api/groups/")

	if err := h.groupRepo.Delete(id); err != nil {
		respondError(w, http.StatusInternalServerError, "删除分组失败: "+err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "删除成功"})
}

// Reorder 批量排序
func (h *GroupHandler) Reorder(w http.ResponseWriter, r *http.Request) {
	var req GroupReorderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "无效的请求格式")
		return
	}

	items := make([]models.ReorderItem, len(req))

	for i, item := range req {
		items[i].ID = item.ID
		items[i].SortOrder = item.SortOrder
	}

	if err := h.groupRepo.BatchReorder(items); err != nil {
		respondError(w, http.StatusInternalServerError, "排序失败: "+err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "排序成功"})
}
