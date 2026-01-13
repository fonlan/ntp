package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	localIconPrefix = "/data/icons/"
	defaultIconExt  = ".png"
)

var validImageExts = map[string]bool{
	".png":  true,
	".jpg":  true,
	".jpeg": true,
	".gif":  true,
	".svg":  true,
	".ico":  true,
	".webp": true,
}

type IconService struct {
	iconDir string
	baseURL string
	client  *http.Client
	cache   *sync.Map
}

func NewIconService(iconDir, baseURL string) *IconService {
	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
	}

	return &IconService{
		iconDir: iconDir,
		baseURL: baseURL,
		client: &http.Client{
			Timeout:   10 * time.Second,
			Transport: transport,
		},
		cache: &sync.Map{},
	}
}

func (s *IconService) DownloadIcon(iconURL string) (string, error) {
	// 检查内存缓存
	if cached, ok := s.cache.Load(iconURL); ok {
		return cached.(string), nil
	}

	// 如果是本地路径，直接返回
	if strings.HasPrefix(iconURL, localIconPrefix) {
		s.cache.Store(iconURL, iconURL)
		return iconURL, nil
	}

	// 下载图标（使用共享客户端）
	resp, err := s.client.Get(iconURL)
	if err != nil {
		return "", fmt.Errorf("下载图标失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("下载图标失败: HTTP %d", resp.StatusCode)
	}

	// 读取内容
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取图标数据失败: %w", err)
	}

	// 保存图标
	path, err := s.saveIcon(iconURL, data, extractExtFromURL(iconURL))
	if err != nil {
		return "", err
	}

	// 写入缓存
	s.cache.Store(iconURL, path)
	return path, nil
}

func extractExtFromURL(iconURL string) string {
	u, err := url.Parse(iconURL)
	if err != nil {
		return defaultIconExt
	}

	path := u.Path
	idx := strings.LastIndex(path, ".")
	if idx == -1 {
		return defaultIconExt
	}

	ext := strings.ToLower(path[idx:])
	if validImageExts[ext] {
		return ext
	}
	return defaultIconExt
}

func (s *IconService) ensureIconDir() error {
	return os.MkdirAll(s.iconDir, 0755)
}

func (s *IconService) generateIconPath(key string, ext string) string {
	hash := sha256.Sum256([]byte(key))
	filename := hex.EncodeToString(hash[:]) + ext
	return filepath.Join(s.iconDir, filename)
}

func (s *IconService) saveIcon(key string, data []byte, ext string) (string, error) {
	if !validImageExts[ext] {
		return "", fmt.Errorf("不支持的文件格式: %s", ext)
	}

	filePath := s.generateIconPath(key, ext)

	if err := s.ensureIconDir(); err != nil {
		return "", fmt.Errorf("创建图标目录失败: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return "", fmt.Errorf("保存图标失败: %w", err)
	}

	return localIconPrefix + filepath.Base(filePath), nil
}

func (s *IconService) DownloadFavicon(websiteURL string) (string, error) {
	// 解析URL获取域名
	u, err := url.Parse(websiteURL)
	if err != nil {
		return "", fmt.Errorf("解析URL失败: %w", err)
	}

	domain := u.Hostname()

	// 使用多个favicon源（并发优化）
	// 注意：URL 参数使用书签原始 URL 的协议（http 或 https），与书签保持一致
	faviconURLs := []string{
		fmt.Sprintf("https://t0.gstatic.com/faviconV2?client=SOCIAL&type=FAVICON&fallback_opts=TYPE,SIZE,URL&url=%s&size=128", websiteURL),
		fmt.Sprintf("https://www.google.com/s2/favicons?domain=%s&sz=128", domain),
		fmt.Sprintf("https://favicon.yandex.net/favicon/%s", domain),
	}

	// 设置 15 秒超时
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// 并发发起所有请求，使用 goroutine
	type downloadResult struct {
		path string
		err  error
	}

	resultChan := make(chan downloadResult, 1)

	for _, faviconURL := range faviconURLs {
		go func(url string) {
			path, err := s.DownloadIcon(url)
			resultChan <- downloadResult{path, err}
		}(faviconURL)
	}

	// 等待第一个成功结果
	var successPath string
	for {
		select {
		case result := <-resultChan:
			if result.err == nil && result.path != "" {
				successPath = result.path
				cancel() // 成功后取消其他请求
				goto done
			}
		case <-ctx.Done():
			// 超时，返回超时错误
			return "", fmt.Errorf("favicon 下载超时（15秒）")
		}
	}

done:
	if successPath != "" {
		return successPath, nil
	}

	return "", fmt.Errorf("所有favicon源都下载失败")
}

func (s *IconService) SaveUploadedIcon(filename string, data []byte) (string, error) {
	ext := strings.ToLower(filepath.Ext(filename))
	return s.saveIcon(filename, data, ext)
}

func (s *IconService) DeleteIcon(iconPath string) error {
	if !strings.HasPrefix(iconPath, localIconPrefix) {
		return nil // 不是本地文件，不删除
	}

	filePath := filepath.Join(s.iconDir, strings.TrimPrefix(iconPath, localIconPrefix))

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil // 文件不存在，无需删除
	}

	return os.Remove(filePath)
}

func (s *IconService) GetIconDir() string {
	return s.iconDir
}
