package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	"github.com/PuerkitoBio/goquery"
	_ "golang.org/x/image/webp"
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
	options := []IconOption{
		{
			URL:       fmt.Sprintf("https://www.google.com/s2/favicons?domain=%s&sz=128", domain),
			Type:      "image/png",
			Sizes:     "128x128",
			Rel:       "icon",
			IsFavicon: false,
		},
		{
			URL:       fmt.Sprintf("https://icons.duckduckgo.com/ip3/%s.ico", domain),
			Type:      "image/x-icon",
			Rel:       "icon",
			IsFavicon: false,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if size := detectIconSize(ctx, options[1].URL); size != "" {
		options[1].Sizes = size
	}

	return options
}

// detectIconSize 下载图标并检测尺寸，返回 "宽x高" 格式字符串
func detectIconSize(ctx context.Context, iconURL string) string {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, iconURL, nil)
	if err != nil {
		return ""
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; NTP/1.0)")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		return ""
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(io.LimitReader(resp.Body, 512*1024))
	if err != nil {
		return ""
	}

	contentType := resp.Header.Get("Content-Type")
	reader := bytes.NewReader(data)

	var width, height int

	if strings.Contains(contentType, "svg") || strings.HasSuffix(strings.ToLower(iconURL), ".svg") {
		if w, h := parseSVGSize(data); w > 0 && h > 0 {
			width, height = w, h
		}
	} else if cfg, _, err := image.DecodeConfig(reader); err == nil {
		width, height = cfg.Width, cfg.Height
	} else if strings.Contains(contentType, "icon") || strings.HasSuffix(strings.ToLower(iconURL), ".ico") {
		if w, h := parseICOSize(data); w > 0 && h > 0 {
			width, height = w, h
		}
	}

	if width > 0 && height > 0 {
		return fmt.Sprintf("%dx%d", width, height)
	}
	return ""
}

func parseSVGSize(data []byte) (int, int) {
	content := string(data)
	var width, height int

	if idx := strings.Index(content, "viewBox"); idx != -1 {
		end := strings.Index(content[idx:], ">")
		if end > 0 {
			viewBox := content[idx : idx+end]
			var x, y, w, h float64
			if _, err := fmt.Sscanf(extractAttrValue(viewBox, "viewBox"), "%f %f %f %f", &x, &y, &w, &h); err == nil {
				return int(w), int(h)
			}
		}
	}

	if idx := strings.Index(content, "<svg"); idx != -1 {
		end := strings.Index(content[idx:], ">")
		if end > 0 {
			svgTag := content[idx : idx+end]
			if w := extractNumericAttr(svgTag, "width"); w > 0 {
				width = w
			}
			if h := extractNumericAttr(svgTag, "height"); h > 0 {
				height = h
			}
		}
	}

	return width, height
}

func extractAttrValue(tag, attr string) string {
	patterns := []string{attr + `="`, attr + `='`}
	for _, p := range patterns {
		if idx := strings.Index(tag, p); idx != -1 {
			start := idx + len(p)
			end := strings.IndexAny(tag[start:], `"'`)
			if end > 0 {
				return tag[start : start+end]
			}
		}
	}
	return ""
}

func extractNumericAttr(tag, attr string) int {
	val := extractAttrValue(tag, attr)
	if val == "" {
		return 0
	}
	val = strings.TrimSuffix(strings.TrimSuffix(val, "px"), "pt")
	var num int
	fmt.Sscanf(val, "%d", &num)
	return num
}

func parseICOSize(data []byte) (int, int) {
	if len(data) < 6 {
		return 0, 0
	}

	count := int(data[4]) | int(data[5])<<8
	if count == 0 || len(data) < 6+count*16 {
		return 0, 0
	}

	var maxW, maxH int
	for i := 0; i < count; i++ {
		offset := 6 + i*16
		w := int(data[offset])
		h := int(data[offset+1])
		if w == 0 {
			w = 256
		}
		if h == 0 {
			h = 256
		}
		if w > maxW || h > maxH {
			maxW, maxH = w, h
		}
	}
	return maxW, maxH
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

	if len(options) == 0 {
		return options
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	results := make([]string, len(options))

	for i, opt := range options {
		if opt.Sizes != "" {
			continue
		}
		wg.Add(1)
		go func(idx int, url string) {
			defer wg.Done()
			results[idx] = detectIconSize(ctx, url)
		}(i, opt.URL)
	}
	wg.Wait()

	for i := range options {
		if options[i].Sizes == "" && results[i] != "" {
			options[i].Sizes = results[i]
		}
	}

	return options
}
