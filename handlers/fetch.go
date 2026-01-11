package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"ntp/middleware"
)

// FetchHandler 自动获取网站元数据
type FetchHandler struct{}

// NewFetchHandler 创建 FetchHandler
func NewFetchHandler() *FetchHandler {
	return &FetchHandler{}
}

// FetchRequest 获取元数据请求
type FetchRequest struct {
	URL string `json:"url"`
}

// FetchResponse 获取元数据响应
type FetchResponse struct {
	Title        string        `json:"title"`
	IconURL      string        `json:"icon_url,omitempty"`
	IconOptions  []IconOption  `json:"icon_options,omitempty"`
}

// IconOption 图标选项
type IconOption struct {
	URL     string `json:"url"`
	Type    string `json:"type,omitempty"`    // image/png, image/svg+xml 等
	Sizes   string `json:"sizes,omitempty"`   // 任何尺寸，如 "64x64", "32x32 64x64"
	Rel     string `json:"rel,omitempty"`     // icon, apple-touch-icon 等
	IsFavicon bool `json:"is_favicon"`        // 是否为 /favicon.ico
}

// FetchMetadata 获取网站元数据
func (h *FetchHandler) FetchMetadata(w http.ResponseWriter, r *http.Request) {
	translator := middleware.TranslatorFromContext(r.Context())

	var req FetchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, translator.T("common.invalidRequest"))
		return
	}

	title, iconOptions, err := FetchMetadataFromURL(req.URL)
	if err != nil {
		respondError(w, http.StatusInternalServerError, translator.T("fetch.fetchFailed")+": "+err.Error())
		return
	}

	respondJSON(w, http.StatusOK, FetchResponse{
		Title:       title,
		IconOptions: iconOptions,
	})
}

// FetchMetadataFromURL 从 URL 获取标题和所有图标
func FetchMetadataFromURL(inputURL string) (string, []IconOption, error) {
	// 确保URL有协议前缀
	url := inputURL
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "https://" + url
	}

	// 创建 HTTP 客户端，设置超时
	client := &http.Client{
		Timeout: 8 * time.Second,
	}

	// 发送 GET 请求
	resp, err := client.Get(url)
	if err != nil {
		return "", nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态码
	if resp.StatusCode != http.StatusOK {
		return "", nil, fmt.Errorf("HTTP 状态码: %d", resp.StatusCode)
	}

	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, fmt.Errorf("读取响应失败: %w", err)
	}

	// 使用 goquery 解析 HTML
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		return "", nil, fmt.Errorf("解析 HTML 失败: %w", err)
	}

	// 提取标题
	title := doc.Find("title").First().Text()
	if title == "" {
		title = inputURL
	}

	// 提取所有图标链接
	var iconOptions []IconOption
	baseURL := url
	start := 8 // https://
	if strings.HasPrefix(url, "http://") {
		start = 7 // http://
	}
	if idx := strings.Index(url[start:], "/"); idx != -1 {
		baseURL = url[:start+idx]
	}

	// 收集所有图标相关的 link 标签
	iconRels := []string{
		"icon", "shortcut icon", "apple-touch-icon", "apple-touch-icon-precomposed",
		"fluid-icon", "mask-icon", "icon shortcut-icon",
	}

	doc.Find("link").Each(func(i int, s *goquery.Selection) {
		rel, _ := s.Attr("rel")
		href, exists := s.Attr("href")
		if !exists || href == "" {
			return
		}

		// 检查是否是图标相关的 link
		relLower := strings.ToLower(rel)
		isIcon := false
		for _, iconRel := range iconRels {
			if relLower == iconRel || strings.Contains(relLower, iconRel) {
				isIcon = true
				break
			}
		}
		if !isIcon {
			return
		}

		// 构建完整的 URL
		iconURL := href
		if strings.HasPrefix(iconURL, "//") {
			protocol := "https:"
			if strings.HasPrefix(url, "http://") {
				protocol = "http:"
			}
			iconURL = protocol + iconURL
		} else if !strings.HasPrefix(iconURL, "http://") && !strings.HasPrefix(iconURL, "https://") {
			// 处理相对路径
			if strings.HasPrefix(iconURL, "/") {
				// 绝对路径: /assets/favicon.png
				iconURL = baseURL + iconURL
			} else {
				// 相对路径: assets/favicon.png
				iconURL = baseURL + "/" + iconURL
			}
		}

		// 获取图标属性
		iconType, _ := s.Attr("type")
		sizes, _ := s.Attr("sizes")

		iconOptions = append(iconOptions, IconOption{
			URL:        iconURL,
			Type:       iconType,
			Sizes:      sizes,
			Rel:        rel,
			IsFavicon:  false,
		})
	})

	// 如果没有找到任何图标，添加 /favicon.ico
	if len(iconOptions) == 0 {
		iconOptions = append(iconOptions, IconOption{
			URL:        baseURL + "/favicon.ico",
			IsFavicon:  true,
		})
	}

	return strings.TrimSpace(title), iconOptions, nil
}
