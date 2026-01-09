package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
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
	Title   string `json:"title"`
	IconURL string `json:"icon_url"`
}

// FetchMetadata 获取网站元数据
func (h *FetchHandler) FetchMetadata(w http.ResponseWriter, r *http.Request) {
	var req FetchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "无效的请求格式")
		return
	}

	title, iconURL, err := FetchMetadataFromURL(req.URL)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "获取失败: "+err.Error())
		return
	}

	respondJSON(w, http.StatusOK, FetchResponse{
		Title:   title,
		IconURL: iconURL,
	})
}

// FetchMetadataFromURL 从 URL 获取标题和图标
func FetchMetadataFromURL(url string) (string, string, error) {
	// 确保URL有协议前缀
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "https://" + url
	}

	// 创建 HTTP 客户端，设置超时
	client := &http.Client{
		Timeout: 5 * time.Second,
		// 不跟随重定向
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	// 发送 GET 请求
	resp, err := client.Get(url)
	if err != nil {
		return "", "", fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态码
	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("HTTP 状态码: %d", resp.StatusCode)
	}

	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("读取响应失败: %w", err)
	}

	// 使用 goquery 解析 HTML
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		return "", "", fmt.Errorf("解析 HTML 失败: %w", err)
	}

	// 提取标题
	title := doc.Find("title").First().Text()
	if title == "" {
		title = url
	}

	// 提取图标
	iconURL := ""
	doc.Find("link[rel='icon'], link[rel='shortcut icon']").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if exists && iconURL == "" {
			iconURL = href
		}
	})

	// 如果没有找到图标，尝试 /favicon.ico
	if iconURL == "" {
		baseURL := url
		start := 8 // https://
		if strings.HasPrefix(url, "http://") {
			start = 7 // http://
		}
		if idx := strings.Index(url[start:], "/"); idx != -1 {
			baseURL = url[:start+idx]
		}
		iconURL = baseURL + "/favicon.ico"
	} else if strings.HasPrefix(iconURL, "//") {
		// 处理协议相对 URL
		protocol := "https:"
		if strings.HasPrefix(url, "http://") {
			protocol = "http:"
		}
		iconURL = protocol + iconURL
	} else if !strings.HasPrefix(iconURL, "http://") && !strings.HasPrefix(iconURL, "https://") {
		// 处理相对路径
		baseURL := url
		start := 8 // https://
		if strings.HasPrefix(url, "http://") {
			start = 7 // http://
		}
		if idx := strings.Index(url[start:], "/"); idx != -1 {
			baseURL = url[:start+idx]
		}
		iconURL = baseURL + iconURL
	}

	return strings.TrimSpace(title), iconURL, nil
}
