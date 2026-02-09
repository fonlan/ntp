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
	}
}

// IsValidLocalIconPath 判断 iconPath 是否为合法的本地图标路径。
//
// 为了避免路径穿越，只允许 `/data/icons/<filename>` 且 filename 不包含路径分隔符，扩展名必须是支持的图片格式。
func (s *IconService) IsValidLocalIconPath(iconPath string) bool {
	if !strings.HasPrefix(iconPath, localIconPrefix) {
		return false
	}

	name := strings.TrimPrefix(iconPath, localIconPrefix)
	if name == "" || name == "." || name == ".." {
		return false
	}

	// 防止路径穿越：只允许文件名，不允许包含路径分隔符。
	if strings.ContainsAny(name, `/\\`) {
		return false
	}

	ext := strings.ToLower(filepath.Ext(name))
	if ext == "" {
		return false
	}
	return validImageExts[ext]
}

// LocalIconFilePath 将 /data/icons/... 转为本地文件路径，并校验路径合法性。
func (s *IconService) LocalIconFilePath(iconPath string) (string, error) {
	if !s.IsValidLocalIconPath(iconPath) {
		return "", fmt.Errorf("无效的图标路径")
	}
	name := strings.TrimPrefix(iconPath, localIconPrefix)
	return filepath.Join(s.iconDir, name), nil
}

func (s *IconService) DownloadIcon(iconURL string) (string, error) {
	return s.downloadIconWithContext(context.Background(), iconURL)
}

func (s *IconService) downloadIconWithContext(ctx context.Context, iconURL string) (string, error) {
	if strings.HasPrefix(iconURL, localIconPrefix) {
		if !s.IsValidLocalIconPath(iconURL) {
			return "", fmt.Errorf("无效的图标路径")
		}
		return iconURL, nil
	}

	ext := extractExtFromURL(iconURL)
	expectedPath := s.generateIconPath(iconURL, ext)
	if _, err := os.Stat(expectedPath); err == nil {
		return localIconPrefix + filepath.Base(expectedPath), nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, iconURL, nil)
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %w", err)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("下载图标失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("下载图标失败: HTTP %d", resp.StatusCode)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
	if err != nil {
		return "", fmt.Errorf("读取图标数据失败: %w", err)
	}

	path, err := s.saveIcon(iconURL, data, ext)
	if err != nil {
		return "", err
	}

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

	resultChan := make(chan downloadResult, len(faviconURLs))

	for _, faviconURL := range faviconURLs {
		go func(iconCtx context.Context, url string) {
			path, err := s.downloadIconWithContext(iconCtx, url)
			resultChan <- downloadResult{path, err}
		}(ctx, faviconURL)
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

	filePath, err := s.LocalIconFilePath(iconPath)
	if err != nil {
		return err
	}

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil // 文件不存在，无需删除
	}

	return os.Remove(filePath)
}

func (s *IconService) GetIconDir() string {
	return s.iconDir
}
