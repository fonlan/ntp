package handlers

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"ntp/middleware"
	"ntp/models"
	"ntp/services"
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
	URL         string  `json:"url"`
	GroupID     *int64  `json:"group_id"`
	Title       string  `json:"title,omitempty"`
	IconURL     string  `json:"icon_url,omitempty"`
	IconChar    *string `json:"icon_char,omitempty"`
	IconBgColor *string `json:"icon_bg_color,omitempty"`
	Description string  `json:"description,omitempty"`
	IsNewWindow bool    `json:"is_new_window"`
}

// BookmarkUpdateRequest 更新请求
type BookmarkUpdateRequest struct {
	URL         *string `json:"url,omitempty"`
	Title       *string `json:"title,omitempty"`
	IconURL     *string `json:"icon_url,omitempty"`
	IconChar    *string `json:"icon_char,omitempty"`
	IconBgColor *string `json:"icon_bg_color,omitempty"`
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

	// 获取分组中最大的 sort_order 值，新书签放在最后
	maxSortOrder, err := h.bookmarkRepo.GetMaxSortOrder(req.GroupID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, translator.T("bookmark.createFailed")+": "+err.Error())
		return
	}

	bookmark := &models.Bookmark{
		URL:         req.URL,
		Title:       req.Title,
		GroupID:     req.GroupID,
		SortOrder:   maxSortOrder + 1,
		IsNewWindow: req.IsNewWindow,
	}

	// 处理可选字段
	if req.IconURL != "" {
		bookmark.IconURL = &req.IconURL
	}
	if req.IconChar != nil && *req.IconChar != "" {
		bookmark.IconChar = req.IconChar
	}
	if req.IconBgColor != nil && *req.IconBgColor != "" {
		bookmark.IconBgColor = req.IconBgColor
	}
	if req.Description != "" {
		bookmark.Description = &req.Description
	}

	// 如果有字符图标，跳过图片图标处理
	if bookmark.IconChar != nil && *bookmark.IconChar != "" {
		// 清空图片图标相关字段
		bookmark.IconURL = nil
		bookmark.IconPath = nil
	} else {
		// 处理图片图标
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
				if !h.iconService.IsValidLocalIconPath(iconURL) {
					respondError(w, http.StatusBadRequest, translator.T("common.invalidIconPath"))
					return
				}
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

	// 处理图标类型切换
	// 如果提供了 icon_char，说明要使用字符图标
	if req.IconChar != nil {
		iconChar := *req.IconChar
		if iconChar != "" {
			// 设置字符图标，清空图片图标
			bookmark.IconChar = &iconChar
			// 删除旧图片图标
			if bookmark.IconPath != nil && *bookmark.IconPath != "" {
				h.iconService.DeleteIcon(*bookmark.IconPath)
			}
			bookmark.IconURL = nil
			bookmark.IconPath = nil
		} else {
			// icon_char 为空字符串，清空字符图标
			bookmark.IconChar = nil
		}
	}

	// 处理图片图标（只有在没有字符图标时才处理）
	if req.IconURL != nil && bookmark.IconChar == nil {
		iconURL := *req.IconURL
		// 忽略空字符串，保持原有的图片图标
		if iconURL == "" {
			// 不做任何更改，保持原有的 icon_path 和 icon_url
		} else if strings.HasPrefix(iconURL, "/data/icons/") {
			if !h.iconService.IsValidLocalIconPath(iconURL) {
				respondError(w, http.StatusBadRequest, translator.T("common.invalidIconPath"))
				return
			}
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
	if req.IconBgColor != nil {
		bookmark.IconBgColor = req.IconBgColor
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

	// 获取导入模式（追加、覆盖、合并）
	mode := r.FormValue("mode")
	if mode == "" {
		mode = "append" // 默认追加模式
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

	// 覆盖模式：先删除所有现有书签和分组
	if mode == "overwrite" {
		// 删除所有书签
		h.bookmarkRepo.DeleteAll()
		// 删除所有分组
		h.groupRepo.DeleteAll()
	}

	// 解析 HTML 书签格式，同时解析书签及其所属分组
	bookmarks, groupNames, groupNameToIndex, err := parseNetscapeBookmarksWithGroups(string(content))
	if err != nil {
		respondError(w, http.StatusBadRequest, translator.T("import.parseFailed")+": "+err.Error())
		return
	}

	// 创建或获取分组
	groupMap := make(map[string]int64) // 分组名 -> 分组ID
	existingGroups, _ := h.groupRepo.GetAll()

	// 先构建已存在分组的映射
	for _, g := range existingGroups {
		groupMap[g.Name] = g.ID
	}

	// 为新分组创建映射
	maxOrder := 0
	for _, g := range existingGroups {
		if g.SortOrder > maxOrder {
			maxOrder = g.SortOrder
		}
	}

	for _, groupName := range groupNames {
		if _, exists := groupMap[groupName]; !exists {
			maxOrder++
			newGroup := &models.Group{
				Name:      groupName,
				SortOrder: maxOrder,
			}
			if err := h.groupRepo.Create(newGroup); err == nil {
				groupMap[groupName] = newGroup.ID
			}
		}
	}

	// merge 模式下预加载现有书签，避免 N+1 查询
	existingByURL := make(map[string]*models.Bookmark)
	if mode == "merge" {
		urls := make([]string, 0, len(bookmarks))
		for i := range bookmarks {
			if bookmarks[i].URL != "" {
				urls = append(urls, bookmarks[i].URL)
			}
		}
		m, err := h.bookmarkRepo.GetBookmarksByURLs(urls)
		if err != nil {
			respondError(w, http.StatusInternalServerError, translator.T("bookmark.listFailed")+": "+err.Error())
			return
		}
		for u, b := range m {
			existingByURL[u] = b
		}
	}

	// 批量插入书签
	imported := 0
	failed := 0
	errors := []string{}

	// 创建一个从临时索引到实际分组 ID 的映射
	indexToGroupID := make(map[int64]*int64)
	for name, tempIdx := range groupNameToIndex {
		if actualGroupID, exists := groupMap[name]; exists {
			indexToGroupID[tempIdx] = &actualGroupID
		}
	}

	// sort_order 计数器：避免 N+1 查询问题
	groupSortOrders := make(map[int64]int)

	for i := range bookmarks {
		bookmark := &bookmarks[i]

		// 将临时的分组索引映射到实际的分组 ID
		if bookmark.GroupID != nil {
			tempID := *bookmark.GroupID
			if actualGroupID, ok := indexToGroupID[tempID]; ok {
				bookmark.GroupID = actualGroupID
			} else {
				bookmark.GroupID = nil // 分组不存在，设为未分组
			}
		}

		// 处理图标
		if bookmark.IconURL != nil && *bookmark.IconURL != "" {
			iconURL := *bookmark.IconURL

			// 检查是否是 base64 数据
			if strings.HasPrefix(iconURL, "data:image/") {
				// 解析 base64 数据
				parts := strings.SplitN(iconURL, ",", 2)
				if len(parts) == 2 {
					// 解码 base64
					decoded, err := base64.StdEncoding.DecodeString(parts[1])
					if err == nil {
						// 保存图标
						ext := ".png"
						if strings.HasPrefix(parts[0], "data:image/jpeg") {
							ext = ".jpg"
						} else if strings.HasPrefix(parts[0], "data:image/gif") {
							ext = ".gif"
						} else if strings.HasPrefix(parts[0], "data:image/svg+xml") {
							ext = ".svg"
						} else if strings.HasPrefix(parts[0], "data:image/webp") {
							ext = ".webp"
						}

						// 使用 base64 数据的 hash 作为文件名（和上传图标相同的规则）
						hash := sha256.Sum256(decoded)
						hashStr := hex.EncodeToString(hash[:])
						uniqueFilename := hashStr + ext

						iconPath, err := h.iconService.SaveUploadedIcon(uniqueFilename, decoded)
						if err == nil {
							bookmark.IconPath = &iconPath
							bookmark.IconURL = nil
						} else {
							// 保存失败，清空 IconURL 避免前端显示 base64 字符串
							bookmark.IconURL = nil
						}
					} else {
						// 解码失败，清空 IconURL
						bookmark.IconURL = nil
					}
				} else {
					// 格式错误，清空 IconURL
					bookmark.IconURL = nil
				}
			} else if !strings.HasPrefix(iconURL, "/data/icons/") {
				// 外部 URL，尝试下载并保存到本地
				iconPath, err := h.iconService.DownloadIcon(iconURL)
				if err == nil {
					bookmark.IconPath = &iconPath
					bookmark.IconURL = nil
				}
				// 如果下载失败，清空 IconURL
				if bookmark.IconURL != nil {
					bookmark.IconURL = nil
				}
			} else {
				// 已经是本地图标路径，直接使用
				if h.iconService.IsValidLocalIconPath(iconURL) {
					bookmark.IconPath = &iconURL
				}
				bookmark.IconURL = nil
			}
		}

		// 获取最大 sort_order（使用内存计数器避免 N+1 查询）
		groupID := bookmark.GroupID
		key := int64(-1)
		if groupID != nil {
			key = *groupID
		}
		currentMax := groupSortOrders[key]
		groupSortOrders[key] = currentMax + 1
		bookmark.SortOrder = groupSortOrders[key]

		// 合并模式：检查 URL 是否已存在
		if mode == "merge" {
			existingBookmark := existingByURL[bookmark.URL]
			if existingBookmark != nil {
				// URL 已存在，更新现有书签
				existingBookmark.Title = bookmark.Title
				existingBookmark.Description = bookmark.Description
				existingBookmark.GroupID = bookmark.GroupID
				existingBookmark.SortOrder = bookmark.SortOrder
				existingBookmark.IsNewWindow = bookmark.IsNewWindow
				// 更新图标（如果有新图标）
				if bookmark.IconPath != nil {
					// 删除旧图标
					if existingBookmark.IconPath != nil && *existingBookmark.IconPath != "" {
						h.iconService.DeleteIcon(*existingBookmark.IconPath)
					}
					existingBookmark.IconPath = bookmark.IconPath
				}
				if bookmark.IconURL != nil {
					existingBookmark.IconURL = bookmark.IconURL
				}
				if err := h.bookmarkRepo.Update(existingBookmark); err != nil {
					failed++
					errors = append(errors, fmt.Sprintf("%s: %s", bookmark.Title, err.Error()))
				} else {
					imported++
				}
				continue
			}
		}

		// 插入新书签（追加模式或合并模式中 URL 不存在的情况）
		if err := h.bookmarkRepo.Create(bookmark); err != nil {
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

// parseNetscapeBookmarksWithGroups 解析 Netscape 书签格式，同时解析分组关联
func parseNetscapeBookmarksWithGroups(html string) ([]models.Bookmark, []string, map[string]int64, error) {
	var bookmarks []models.Bookmark
	var groupNames []string                    // 有序的分组名列表
	groupNameToIndex := make(map[string]int64) // 分组名 -> 临时索引
	var currentGroupName string
	groupIndex := int64(1)

	lines := strings.Split(html, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// 解析分组标题
		if strings.HasPrefix(line, `<DT><H3`) {
			// 找到 <H3 标签的结束位置（最后一个 >）
			h3End := strings.Index(line, `<H3`)
			if h3End != -1 {
				// 从 <H3 后找第一个 >，这是属性标签的结束
				attrEnd := strings.Index(line[h3End:], `>`)
				if attrEnd != -1 {
					start := h3End + attrEnd + 1
					end := strings.Index(line[start:], `</H3>`)
					if end != -1 {
						currentGroupName = line[start : start+end]
						if _, exists := groupNameToIndex[currentGroupName]; !exists {
							groupNames = append(groupNames, currentGroupName)
							groupNameToIndex[currentGroupName] = groupIndex
							groupIndex++
						}
					}
				}
			}
		}

		// 分组结束
		if strings.HasPrefix(line, `</DL><p>`) {
			currentGroupName = ""
		}

		// 解析书签
		if strings.HasPrefix(line, `<DT><A HREF="`) {
			// 提取 URL
			urlStart := strings.Index(line, `HREF="`) + 6
			urlEnd := strings.Index(line[urlStart:], `"`)
			url := line[urlStart : urlStart+urlEnd]

			// 提取标题
			titleStart := strings.Index(line[urlStart+urlEnd:], `>`) + urlStart + urlEnd + 1
			titleEnd := strings.Index(line[titleStart:], `</A>`)
			title := line[titleStart : titleStart+titleEnd]

			bookmark := models.Bookmark{
				Title: title,
				URL:   url,
			}

			// 解析图标相关属性
			iconType := extractAttr(line, "ICON_TYPE")
			iconBgColor := extractAttr(line, "ICON_BG_COLOR")
			iconChar := extractAttr(line, "ICON_CHAR")

			if iconBgColor != "" {
				bookmark.IconBgColor = &iconBgColor
			}

			if iconType == "text" && iconChar != "" {
				bookmark.IconChar = &iconChar
			} else {
				// 图片图标：从 ICON 或 ICON_URI 获取
				var iconURL string
				iconTag := `ICON="`
				if iconStart := strings.Index(line, iconTag); iconStart != -1 {
					iconStart += len(iconTag)
					iconEnd := len(line)
					if nextPos := strings.Index(line[iconStart:], ` DESC="`); nextPos != -1 {
						iconEnd = iconStart + nextPos
					}
					if nextPos := strings.Index(line[iconStart:], ` NEW_WINDOW="`); nextPos != -1 && iconStart+nextPos < iconEnd {
						iconEnd = iconStart + nextPos
					}
					if nextPos := strings.Index(line[iconStart:], ` ICON_TYPE="`); nextPos != -1 && iconStart+nextPos < iconEnd {
						iconEnd = iconStart + nextPos
					}
					if nextPos := strings.Index(line[iconStart:], ` ICON_BG_COLOR="`); nextPos != -1 && iconStart+nextPos < iconEnd {
						iconEnd = iconStart + nextPos
					}
					if nextPos := strings.Index(line[iconStart:], `>`); nextPos != -1 && iconStart+nextPos < iconEnd {
						iconEnd = iconStart + nextPos
					}
					lastQuote := strings.LastIndex(line[iconStart:iconEnd], `"`)
					if lastQuote != -1 {
						iconURL = line[iconStart : iconStart+lastQuote]
					}
					if iconURL != "" {
						bookmark.IconURL = &iconURL
					}
				} else if iconStart := strings.Index(line, `ICON_URI="`); iconStart != -1 {
					iconStart += 10
					iconEnd := len(line)
					if nextPos := strings.Index(line[iconStart:], ` DESC="`); nextPos != -1 {
						iconEnd = iconStart + nextPos
					}
					if nextPos := strings.Index(line[iconStart:], ` NEW_WINDOW="`); nextPos != -1 && iconStart+nextPos < iconEnd {
						iconEnd = iconStart + nextPos
					}
					if nextPos := strings.Index(line[iconStart:], `>`); nextPos != -1 && iconStart+nextPos < iconEnd {
						iconEnd = iconStart + nextPos
					}
					lastQuote := strings.LastIndex(line[iconStart:iconEnd], `"`)
					if lastQuote != -1 {
						iconURL = line[iconStart : iconStart+lastQuote]
					}
					if iconURL != "" {
						bookmark.IconURL = &iconURL
					}
				}
			}

			// 提取描述
			if descStart := strings.Index(line, `DESC="`); descStart != -1 {
				descStart += 6
				descEnd := strings.Index(line[descStart:], `"`)
				if descEnd != -1 {
					desc := line[descStart : descStart+descEnd]
					if desc != "" {
						bookmark.Description = &desc
					}
				}
			}

			// 设置分组
			if currentGroupName != "" {
				// 临时使用 groupIndex 作为 GroupID，后面会替换为实际的分组 ID
				tempID := groupNameToIndex[currentGroupName]
				bookmark.GroupID = &tempID
			}

			// 提取是否在新窗口打开
			if strings.Contains(line, `NEW_WINDOW="1"`) {
				bookmark.IsNewWindow = true
			}

			bookmarks = append(bookmarks, bookmark)
		}
	}

	return bookmarks, groupNames, groupNameToIndex, nil
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
	html := generateNetscapeBookmarks(bookmarks, groups, h.iconService)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="bookmarks.html"`)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(html))
}

// 辅助函数：将图标转换为 base64
func iconToBase64(iconPath *string, iconURL *string, iconService *services.IconService) string {
	// 优先使用本地图标
	if iconPath != nil && *iconPath != "" {
		fullPath, err := iconService.LocalIconFilePath(*iconPath)
		if err == nil {
			data, err := os.ReadFile(fullPath)
			if err == nil {
				// 根据扩展名确定 MIME 类型
				ext := strings.ToLower(filepath.Ext(fullPath))
				mimeType := "image/png"
				switch ext {
				case ".jpg", ".jpeg":
					mimeType = "image/jpeg"
				case ".gif":
					mimeType = "image/gif"
				case ".svg":
					mimeType = "image/svg+xml"
				case ".webp":
					mimeType = "image/webp"
				case ".ico":
					mimeType = "image/x-icon"
				}
				return "data:" + mimeType + ";base64," + base64.StdEncoding.EncodeToString(data)
			}
		}
	}

	// 使用 URL
	if iconURL != nil && *iconURL != "" {
		return *iconURL
	}

	return ""
}

// 辅助函数：转义 HTML 属性
func escapeHTMLAttr(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

// extractAttr 从 HTML 标签行中提取指定属性的值
func extractAttr(line, attrName string) string {
	tag := attrName + `="`
	start := strings.Index(line, tag)
	if start == -1 {
		return ""
	}
	start += len(tag)
	end := strings.Index(line[start:], `"`)
	if end == -1 {
		return ""
	}
	return line[start : start+end]
}

// 辅助函数：转义 HTML 内容
func escapeHTMLContent(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

// 辅助函数：生成书签的图标属性字符串
// 包含 ICON_TYPE（image/text）、ICON（base64 或空）、ICON_CHAR（文字图标字符）、ICON_BG_COLOR（背景色）
func buildIconAttrs(b models.Bookmark, iconService *services.IconService) string {
	var attrs strings.Builder

	// 判断图标类型
	hasImageIcon := (b.IconPath != nil && *b.IconPath != "") || (b.IconURL != nil && *b.IconURL != "")
	hasTextIcon := b.IconChar != nil && *b.IconChar != ""

	if hasImageIcon {
		// 图片图标
		iconData := iconToBase64(b.IconPath, b.IconURL, iconService)
		attrs.WriteString(fmt.Sprintf(` ICON_TYPE="image" ICON="%s"`, escapeHTMLAttr(iconData)))
	} else if hasTextIcon {
		// 文字图标
		attrs.WriteString(fmt.Sprintf(` ICON_TYPE="text" ICON_CHAR="%s"`, escapeHTMLAttr(*b.IconChar)))
	}

	// 背景色（无论图标类型都输出）
	if b.IconBgColor != nil && *b.IconBgColor != "" {
		attrs.WriteString(fmt.Sprintf(` ICON_BG_COLOR="%s"`, escapeHTMLAttr(*b.IconBgColor)))
	}

	return attrs.String()
}

// generateNetscapeBookmarks 生成 Netscape 书签格式
func generateNetscapeBookmarks(bookmarks []models.Bookmark, groups []models.Group, iconService *services.IconService) string {
	var sb strings.Builder

	sb.WriteString(`<!DOCTYPE NETSCAPE-Bookmark-file-1>` + "\n")
	sb.WriteString(`<!-- This is an automatically generated file.` + "\n")
	sb.WriteString(`     It will be read and overwritten.` + "\n")
	sb.WriteString(`     DO NOT EDIT! -->` + "\n")
	sb.WriteString(`<META HTTP-EQUIV="Content-Type" CONTENT="text/html; charset=UTF-8">` + "\n")
	sb.WriteString(`<TITLE>Bookmarks</TITLE>` + "\n")
	sb.WriteString(`<H1>Bookmarks</H1>` + "\n")
	sb.WriteString(`<DL><p>` + "\n")

	// 分组书签
	groupMap := make(map[int64]models.Group)
	for _, g := range groups {
		groupMap[g.ID] = g
	}

	// 按分组索引书签（优化 O(n*m) 嵌套循环）
	groupBookmarks := make(map[int64][]models.Bookmark)
	for _, b := range bookmarks {
		if b.GroupID == nil {
			groupBookmarks[-1] = append(groupBookmarks[-1], b)
		} else {
			groupBookmarks[*b.GroupID] = append(groupBookmarks[*b.GroupID], b)
		}
	}

	// 未分组书签
	sb.WriteString(`<DT><H3 ADD_DATE="0">未分类</H3>` + "\n")
	sb.WriteString(`<DL><p>` + "\n")
	for _, b := range groupBookmarks[-1] {
		iconAttrs := buildIconAttrs(b, iconService)
		desc := ""
		if b.Description != nil {
			desc = fmt.Sprintf(` DESC="%s"`, escapeHTMLAttr(*b.Description))
		}
		newWindow := ""
		if b.IsNewWindow {
			newWindow = ` NEW_WINDOW="1"`
		}
		sb.WriteString(fmt.Sprintf(`<DT><A HREF="%s" ADD_DATE="%d"%s%s%s>%s</A>`+"\n",
			escapeHTMLAttr(b.URL), b.CreatedAt.Unix(), iconAttrs, desc, newWindow, escapeHTMLContent(b.Title)))
	}
	sb.WriteString(`</DL><p>` + "\n")

	// 分组书签
	for _, g := range groups {
		sb.WriteString(fmt.Sprintf(`<DT><H3 ADD_DATE="%d">%s</H3>`+"\n", g.CreatedAt.Unix(), escapeHTMLContent(g.Name)))
		sb.WriteString(`<DL><p>` + "\n")
		if bookmarks, ok := groupBookmarks[g.ID]; ok {
			for _, b := range bookmarks {
				iconAttrs := buildIconAttrs(b, iconService)
				desc := ""
				if b.Description != nil {
					desc = fmt.Sprintf(` DESC="%s"`, escapeHTMLAttr(*b.Description))
				}
				newWindow := ""
				if b.IsNewWindow {
					newWindow = ` NEW_WINDOW="1"`
				}
				sb.WriteString(fmt.Sprintf(`<DT><A HREF="%s" ADD_DATE="%d"%s%s%s>%s</A>`+"\n",
					escapeHTMLAttr(b.URL), b.CreatedAt.Unix(), iconAttrs, desc, newWindow, escapeHTMLContent(b.Title)))
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

	// 读取文件数据（限制 512KB + 1字节用于检测）
	data, err := io.ReadAll(io.LimitReader(file, 512*1024+1))
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
