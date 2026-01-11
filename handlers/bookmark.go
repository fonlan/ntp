package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"ntp/models"
	"ntp/services"
	"ntp/middleware"
)

// BookmarkHandler 书签处理器
type BookmarkHandler struct {
	bookmarkRepo *models.BookmarkRepository
	groupRepo    *models.GroupRepository
	iconService  *services.IconService
}

// NewBookmarkHandler 创建书签处理器
func NewBookmarkHandler(br *models.BookmarkRepository, gr *models.GroupRepository, iconService *services.IconService) *BookmarkHandler {
	return &BookmarkHandler{
		bookmarkRepo: br,
		groupRepo:    gr,
		iconService:  iconService,
	}
}

// ListRequest 列表请求参数
type ListRequest struct {
	GroupID *int64 `json:"group_id"`
}

// BookmarkCreateRequest 创建请求
type BookmarkCreateRequest struct {
	URL         string `json:"url"`
	GroupID     *int64 `json:"group_id"`
	Title       string `json:"title,omitempty"`
	IconURL     string `json:"icon_url,omitempty"`
	Description string `json:"description,omitempty"`
	IsNewWindow bool   `json:"is_new_window"`
}

// BookmarkUpdateRequest 更新请求
type BookmarkUpdateRequest struct {
	URL         *string `json:"url,omitempty"`
	Title       *string `json:"title,omitempty"`
	IconURL     *string `json:"icon_url,omitempty"`
	Description *string `json:"description,omitempty"`
	GroupID     *int64  `json:"group_id"`
	SortOrder   *int    `json:"sort_order,omitempty"`
	IsNewWindow *bool   `json:"is_new_window,omitempty"`
}

// BookmarkReorderRequest 排序请求
type BookmarkReorderRequest []models.ReorderItem

// List 获取书签列表
func (h *BookmarkHandler) List(w http.ResponseWriter, r *http.Request) {
	translator := middleware.TranslatorFromContext(r.Context())

	var req ListRequest
	groupIDStr := r.URL.Query().Get("group_id")

	if groupIDStr != "" {
		id, err := strconv.ParseInt(groupIDStr, 10, 64)
		if err == nil {
			req.GroupID = &id
		}
	}

	bookmarks, err := h.bookmarkRepo.GetAll(req.GroupID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, translator.T("bookmark.listFailed")+": "+err.Error())
		return
	}

	respondJSON(w, http.StatusOK, bookmarks)
}

// Create 创建书签
func (h *BookmarkHandler) Create(w http.ResponseWriter, r *http.Request) {
	translator := middleware.TranslatorFromContext(r.Context())

	var req BookmarkCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, translator.T("common.invalidRequest"))
		return
	}

	bookmark := &models.Bookmark{
		URL:         req.URL,
		Title:       req.Title,
		GroupID:     req.GroupID,
		SortOrder:   0,
		IsNewWindow: req.IsNewWindow,
	}

	// 处理可选字段
	if req.IconURL != "" {
		bookmark.IconURL = &req.IconURL
	}
	if req.Description != "" {
		bookmark.Description = &req.Description
	}

	// 如果没有提供标题，尝试从 URL 自动获取
	if bookmark.Title == "" {
		title, iconOptions, err := FetchMetadataFromURL(bookmark.URL)
		if err == nil {
			bookmark.Title = title
			// 如果没有提供图标且找到了图标选项，使用第一个图标
			if bookmark.IconURL == nil && len(iconOptions) > 0 {
				firstIconURL := iconOptions[0].URL
				bookmark.IconURL = &firstIconURL
			}
		} else {
			bookmark.Title = bookmark.URL
		}
	}

	// 处理图标
	if bookmark.IconURL != nil && *bookmark.IconURL != "" {
		iconURL := *bookmark.IconURL
		// 如果已经是本地图标路径，直接使用
		if strings.HasPrefix(iconURL, "/data/icons/") {
			bookmark.IconPath = &iconURL
			bookmark.IconURL = nil // 使用本地图标后清空URL字段
		} else {
			// 外部URL，尝试下载并保存到本地
			iconPath, err := h.iconService.DownloadIcon(iconURL)
			if err == nil {
				bookmark.IconPath = &iconPath
				bookmark.IconURL = nil // 使用本地图标后清空URL
			}
			// 如果下载失败，保留 IconURL 字段作为后备
		}
	} else {
		// 尝试自动下载 favicon
		iconPath, err := h.iconService.DownloadFavicon(bookmark.URL)
		if err == nil {
			bookmark.IconPath = &iconPath
			bookmark.IconURL = nil // 使用本地图标后清空URL
		}
	}

	if err := h.bookmarkRepo.Create(bookmark); err != nil {
		respondError(w, http.StatusInternalServerError, translator.T("bookmark.createFailed")+": "+err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, bookmark)
}

// Update 更新书签
func (h *BookmarkHandler) Update(w http.ResponseWriter, r *http.Request) {
	translator := middleware.TranslatorFromContext(r.Context())
	id := extractPathID(r, "/api/bookmark/")

	bookmark, err := h.bookmarkRepo.GetByID(id)
	if err != nil || bookmark == nil {
		respondError(w, http.StatusNotFound, translator.T("bookmark.notFound"))
		return
	}

	var req BookmarkUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, translator.T("common.invalidRequest"))
		return
	}

	if req.Title != nil {
		bookmark.Title = *req.Title
	}
	if req.URL != nil {
		bookmark.URL = *req.URL
	}
	if req.IconURL != nil {
		iconURL := *req.IconURL
		// 如果已经是本地图标路径，直接使用
		if strings.HasPrefix(iconURL, "/data/icons/") {
			// 删除旧图标
			if bookmark.IconPath != nil && *bookmark.IconPath != "" && *bookmark.IconPath != iconURL {
				h.iconService.DeleteIcon(*bookmark.IconPath)
			}
			bookmark.IconPath = &iconURL
			bookmark.IconURL = nil // 使用本地图标后清空URL字段
		} else {
			// 外部URL，尝试下载并保存到本地
			bookmark.IconURL = req.IconURL
			iconPath, err := h.iconService.DownloadIcon(iconURL)
			if err == nil {
				// 删除旧图标
				if bookmark.IconPath != nil && *bookmark.IconPath != "" {
					h.iconService.DeleteIcon(*bookmark.IconPath)
				}
				bookmark.IconPath = &iconPath
				bookmark.IconURL = nil // 使用本地图标后清空URL
			}
		}
	}
	if req.Description != nil {
		bookmark.Description = req.Description
	}
	// GroupID 总是更新（因为移除了 omitempty）
	bookmark.GroupID = req.GroupID
	if req.SortOrder != nil {
		bookmark.SortOrder = *req.SortOrder
	}
	if req.IsNewWindow != nil {
		bookmark.IsNewWindow = *req.IsNewWindow
	}

	if err := h.bookmarkRepo.Update(bookmark); err != nil {
		respondError(w, http.StatusInternalServerError, translator.T("bookmark.updateFailed")+": "+err.Error())
		return
	}

	respondJSON(w, http.StatusOK, bookmark)
}

// Delete 删除书签
func (h *BookmarkHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := extractPathID(r, "/api/bookmark/")
	translator := middleware.TranslatorFromContext(r.Context())

	if err := h.bookmarkRepo.Delete(id); err != nil {
		respondError(w, http.StatusInternalServerError, translator.T("bookmark.deleteFailed")+": "+err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "删除成功"})
}

// Reorder 批量排序
func (h *BookmarkHandler) Reorder(w http.ResponseWriter, r *http.Request) {
	var req BookmarkReorderRequest
	translator := middleware.TranslatorFromContext(r.Context())
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, translator.T("common.invalidRequest"))
		return
	}

	items := make([]models.ReorderItem, len(req))

	for i, item := range req {
		items[i].ID = item.ID
		items[i].SortOrder = item.SortOrder
	}

	if err := h.bookmarkRepo.BatchReorder(items); err != nil {
		respondError(w, http.StatusInternalServerError, translator.T("bookmark.reorderFailed")+": "+err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "排序成功"})
}

// Search 搜索书签
func (h *BookmarkHandler) Search(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	translator := middleware.TranslatorFromContext(r.Context())
	if query == "" {
		respondError(w, http.StatusBadRequest, "Missing search keyword")
		return
	}

	bookmarks, err := h.bookmarkRepo.Search(query)
	if err != nil {
		respondError(w, http.StatusInternalServerError, translator.T("bookmark.searchFailed")+": "+err.Error())
		return
	}

	respondJSON(w, http.StatusOK, bookmarks)
}

// Import 导入书签
func (h *BookmarkHandler) Import(w http.ResponseWriter, r *http.Request) {
	translator := middleware.TranslatorFromContext(r.Context())

	// 解析 multipart 表单
	if err := r.ParseMultipartForm(32 << 20); err != nil { // 32MB
		respondError(w, http.StatusBadRequest, translator.T("common.invalidRequest"))
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		respondError(w, http.StatusBadRequest, translator.T("import.fileRequired"))
		return
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		respondError(w, http.StatusInternalServerError, translator.T("upload.uploadFailed"))
		return
	}

	// 解析 HTML 书签格式
	bookmarks, err := parseNetscapeBookmarks(string(content))
	if err != nil {
		respondError(w, http.StatusBadRequest, translator.T("import.parseFailed")+": "+err.Error())
		return
	}

	// 批量插入
	imported := 0
	failed := 0
	errors := []string{}

	for _, bookmark := range bookmarks {
		if err := h.bookmarkRepo.Create(&bookmark); err != nil {
			failed++
			errors = append(errors, fmt.Sprintf("%s: %s", bookmark.Title, err.Error()))
		} else {
			imported++
		}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"imported": imported,
		"failed":   failed,
		"errors":   errors,
	})
}

// Export 导出书签
func (h *BookmarkHandler) Export(w http.ResponseWriter, r *http.Request) {
	translator := middleware.TranslatorFromContext(r.Context())

	bookmarks, err := h.bookmarkRepo.GetAll(nil)
	if err != nil {
		respondError(w, http.StatusInternalServerError, translator.T("bookmark.listFailed"))
		return
	}

	groups, err := h.groupRepo.GetAll()
	if err != nil {
		respondError(w, http.StatusInternalServerError, translator.T("group.listFailed"))
		return
	}

	// 生成 Netscape 书签 HTML 格式
	html := generateNetscapeBookmarks(bookmarks, groups)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="bookmarks.html"`)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(html))
}

// parseNetscapeBookmarks 解析 Netscape 书签格式
func parseNetscapeBookmarks(html string) ([]models.Bookmark, error) {
	var bookmarks []models.Bookmark

	// 简单的 HTML 解析（实际项目中应使用 HTML 解析器）
	lines := strings.Split(html, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, `<DT><A HREF="`) {
			// 提取 URL
			urlStart := strings.Index(line, `HREF="`) + 6
			urlEnd := strings.Index(line[urlStart:], `"`)
			url := line[urlStart : urlStart+urlEnd]

			// 提取标题
			titleStart := strings.Index(line[urlStart+urlEnd:], `>`) + urlStart + urlEnd + 1
			titleEnd := strings.Index(line[titleStart:], `</A>`)
			title := line[titleStart : titleStart+titleEnd]

			// 提取图标
			iconURL := ""
			if iconStart := strings.Index(line, `ICON_URI="`); iconStart != -1 {
				iconStart += 9
				iconEnd := strings.Index(line[iconStart:], `"`)
				iconURL = line[iconStart : iconStart+iconEnd]
			}

			bookmark := models.Bookmark{
				Title: title,
				URL:   url,
			}
			if iconURL != "" {
				bookmark.IconURL = &iconURL
			}
			bookmarks = append(bookmarks, bookmark)
		}
	}

	return bookmarks, nil
}

// generateNetscapeBookmarks 生成 Netscape 书签格式
func generateNetscapeBookmarks(bookmarks []models.Bookmark, groups []models.Group) string {
	var sb strings.Builder

	sb.WriteString(`<!DOCTYPE NETSCAPE-Bookmark-file-1>` + "\n")
	sb.WriteString(`<!-- This is an automatically generated file.` + "\n")
	sb.WriteString(`     It will be read and overwritten.` + "\n")
	sb.WriteString(`     DO NOT EDIT! -->` + "\n")
	sb.WriteString(`<META HTTP-EQUIV="Content-Type" CONTENT="text/html; charset=UTF-8">` + "\n")
	sb.WriteString(`<TITLE>Bookmarks</TITLE>` + "\n")
	sb.WriteString(`<H1>Bookmarks</H1>` + "\n")
	sb.WriteString(`<DL><p>` + "\n")

	// 按分组组织书签
	groupMap := make(map[int64]models.Group)
	for _, g := range groups {
		groupMap[g.ID] = g
	}

	// 未分组书签
	sb.WriteString(`<DT><H3>未分类</H3>` + "\n")
	sb.WriteString(`<DL><p>` + "\n")
	for _, b := range bookmarks {
		if b.GroupID == nil {
			iconURL := ""
			if b.IconURL != nil {
				iconURL = *b.IconURL
			}
			sb.WriteString(fmt.Sprintf(`<DT><A HREF="%s" ADD_DATE="%d" ICON_URI="%s">%s</A>`+"\n",
				b.URL, b.CreatedAt.Unix(), iconURL, b.Title))
		}
	}
	sb.WriteString(`</DL><p>` + "\n")

	// 分组书签
	for _, g := range groups {
		sb.WriteString(fmt.Sprintf(`<DT><H3>%s</H3>`+"\n", g.Name))
		sb.WriteString(`<DL><p>` + "\n")
		for _, b := range bookmarks {
			if b.GroupID != nil && *b.GroupID == g.ID {
				iconURL := ""
				if b.IconURL != nil {
					iconURL = *b.IconURL
				}
				sb.WriteString(fmt.Sprintf(`<DT><A HREF="%s" ADD_DATE="%d" ICON_URI="%s">%s</A>`+"\n",
					b.URL, b.CreatedAt.Unix(), iconURL, b.Title))
			}
		}
		sb.WriteString(`</DL><p>` + "\n")
	}

	sb.WriteString(`</DL><p>` + "\n")

	return sb.String()
}

// UploadIcon 上传图标
func (h *BookmarkHandler) UploadIcon(w http.ResponseWriter, r *http.Request) {
	translator := middleware.TranslatorFromContext(r.Context())

	// 解析 multipart 表单
	if err := r.ParseMultipartForm(5 << 20); err != nil { // 5MB
		respondError(w, http.StatusBadRequest, translator.T("common.invalidRequest"))
		return
	}

	file, header, err := r.FormFile("icon")
	if err != nil {
		respondError(w, http.StatusBadRequest, translator.T("import.fileRequired"))
		return
	}
	defer file.Close()

	// 读取文件数据
	data, err := io.ReadAll(file)
	if err != nil {
		respondError(w, http.StatusInternalServerError, translator.T("upload.uploadFailed"))
		return
	}

	// 验证文件大小（最大 512KB）
	if len(data) > 512*1024 {
		respondError(w, http.StatusBadRequest, "文件大小不能超过 512KB")
		return
	}

	// 保存图标
	iconPath, err := h.iconService.SaveUploadedIcon(header.Filename, data)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "保存图标失败: "+err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"icon_url":  "",
		"icon_path": iconPath,
	})
}
