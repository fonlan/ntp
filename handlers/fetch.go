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
	Title       string       `json:"title"`
	IconURL     string       `json:"icon_url,omitempty"`
	IconOptions []IconOption `json:"icon_options,omitempty"`
}

// IconOption 图标选项
type IconOption struct {
	URL       string `json:"url"`
	Type      string `json:"type,omitempty"`  // image/png, image/svg+xml 等
	Sizes     string `json:"sizes,omitempty"` // 任何尺寸，如 "64x64", "32x32 64x64"
	Rel       string `json:"rel,omitempty"`   // icon, apple-touch-icon 等
	IsFavicon bool   `json:"is_favicon"`      // 是否为 /favicon.ico
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

// iconRelTypes 图标相关的 rel 类型
var iconRelTypes = map[string]bool{
	"icon":                         true,
	"shortcut icon":                true,
	"apple-touch-icon":             true,
	"apple-touch-icon-precomposed": true,
	"fluid-icon":                   true,
	"mask-icon":                    true,
}

// buildBaseURL 从完整 URL 构建基础 URL
func buildBaseURL(fullURL string) string {
	start := 8 // https://
	if strings.HasPrefix(fullURL, "http://") {
		start = 7 // http://
	}
	if idx := strings.Index(fullURL[start:], "/"); idx != -1 {
		return fullURL[:start+idx]
	}
	return fullURL
}

// normalizeIconURL 将图标 URL 转换为完整的绝对 URL
func normalizeIconURL(iconURL, baseURL, pageURL string) string {
	if strings.HasPrefix(iconURL, "//") {
		protocol := "https:"
		if strings.HasPrefix(pageURL, "http://") {
			protocol = "http:"
		}
		return protocol + iconURL
	}
	if strings.HasPrefix(iconURL, "http://") || strings.HasPrefix(iconURL, "https://") {
		return iconURL
	}
	if strings.HasPrefix(iconURL, "/") {
		return baseURL + iconURL
	}
	return baseURL + "/" + iconURL
}

// isIconLink 检查 link 标签是否是图标相关的
func isIconLink(rel string) bool {
	relLower := strings.ToLower(rel)
	for iconRel := range iconRelTypes {
		if relLower == iconRel || strings.Contains(relLower, iconRel) {
			return true
		}
	}
	return false
}

// FetchMetadataFromURL 从 URL 获取标题和所有图标
func FetchMetadataFromURL(inputURL string) (string, []IconOption, error) {
	// 确保URL有协议前缀
	url := inputURL
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "https://" + url
	}

	// 创建 HTTP 客户端，设置超时
	client := &http.Client{Timeout: 8 * time.Second}

	// 提取域名用于备用方案
	domain := extractDomain(url)

	// 发送 GET 请求
	resp, err := client.Get(url)

	var title string
	var iconOptions []IconOption
	var fetchErr error

	// 如果直接访问成功，尝试解析
	if err == nil && resp.StatusCode == http.StatusOK {
		defer resp.Body.Close()

		// 读取响应体（限制 5MB，防止内存溢出）
		body, err := io.ReadAll(io.LimitReader(resp.Body, 5*1024*1024))
		if err == nil {
			// 使用 goquery 解析 HTML
			doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
			if err == nil {
				// 提取标题
				title = doc.Find("title").First().Text()
				if title == "" {
					title = inputURL
				}

				baseURL := buildBaseURL(url)
				iconOptions = extractIconOptions(doc, url, baseURL)
			} else {
				fetchErr = fmt.Errorf("解析 HTML 失败: %w", err)
			}
		} else {
			fetchErr = fmt.Errorf("读取响应失败: %w", err)
		}
	} else {
		if err != nil {
			fetchErr = fmt.Errorf("请求失败: %w", err)
		} else {
			resp.Body.Close()
			fetchErr = fmt.Errorf("HTTP 状态码: %d", resp.StatusCode)
		}
	}

	// 使用默认标题
	if title == "" {
		title = inputURL
	}

	// 如果没有找到任何图标或解析失败，使用备用方案
	if len(iconOptions) == 0 {
		iconOptions = getFallbackIcons(domain)
	}

	// 如果完全失败但有备用图标，仍然返回成功
	if fetchErr != nil && len(iconOptions) > 0 {
		return strings.TrimSpace(title), iconOptions, nil
	}

	if fetchErr != nil {
		return "", nil, fetchErr
	}

	return strings.TrimSpace(title), iconOptions, nil
}

// extractDomain 从 URL 中提取域名
func extractDomain(url string) string {
	url = strings.TrimPrefix(url, "https://")
	url = strings.TrimPrefix(url, "http://")
	if idx := strings.Index(url, "/"); idx != -1 {
		return url[:idx]
	}
	return url
}

// getFallbackIcons 获取备用图标选项
func getFallbackIcons(domain string) []IconOption {
	var options []IconOption

	// Google Favicon Service
	options = append(options, IconOption{
		URL:       fmt.Sprintf("https://www.google.com/s2/favicons?domain=%s&sz=128", domain),
		Type:      "image/png",
		Rel:       "icon",
		IsFavicon: false,
	})

	// DuckDuckGo Icon Service
	options = append(options, IconOption{
		URL:       fmt.Sprintf("https://icons.duckduckgo.com/ip3/%s.ico", domain),
		Type:      "image/x-icon",
		Rel:       "icon",
		IsFavicon: false,
	})

	return options
}

// extractIconOptions 从 HTML 文档中提取所有图标选项
func extractIconOptions(doc *goquery.Document, pageURL, baseURL string) []IconOption {
	var options []IconOption

	doc.Find("link").Each(func(_ int, s *goquery.Selection) {
		rel, _ := s.Attr("rel")
		href, exists := s.Attr("href")
		if !exists || href == "" || !isIconLink(rel) {
			return
		}

		iconType, _ := s.Attr("type")
		sizes, _ := s.Attr("sizes")

		options = append(options, IconOption{
			URL:       normalizeIconURL(href, baseURL, pageURL),
			Type:      iconType,
			Sizes:     sizes,
			Rel:       rel,
			IsFavicon: false,
		})
	})

	return options
}
