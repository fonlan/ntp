package services

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// validImageExts 支持的图片扩展名
var validImageExts = map[string]bool{
	".png":  true,
	".jpg":  true,
	".jpeg": true,
	".gif":  true,
	".svg":  true,
	".ico":  true,
	".webp": true,
}

const (
	localIconPrefix = "/data/icons/"
	defaultIconExt  = ".png"
)

// IconService 图标服务
type IconService struct {
	iconDir string
	baseURL string
}

// NewIconService 创建图标服务
func NewIconService(iconDir, baseURL string) *IconService {
	return &IconService{
		iconDir: iconDir,
		baseURL: baseURL,
	}
}

// DownloadIcon 下载图标并保存到本地
func (s *IconService) DownloadIcon(iconURL string) (string, error) {
	// 如果是本地路径，直接返回
	if strings.HasPrefix(iconURL, localIconPrefix) {
		return iconURL, nil
	}

	// 下载图标
	resp, err := http.Get(iconURL)
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
	return s.saveIcon(iconURL, data, extractExtFromURL(iconURL))
}

// extractExtFromURL 从 URL 中提取文件扩展名
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

// ensureIconDir 确保图标目录存在
func (s *IconService) ensureIconDir() error {
	return os.MkdirAll(s.iconDir, 0755)
}

// generateIconPath 生成图标文件路径
func (s *IconService) generateIconPath(key string, ext string) string {
	hash := sha256.Sum256([]byte(key))
	filename := hex.EncodeToString(hash[:]) + ext
	return filepath.Join(s.iconDir, filename)
}

// saveIcon 保存图标数据到本地
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

// DownloadFavicon 从网站域名下载favicon
func (s *IconService) DownloadFavicon(websiteURL string) (string, error) {
	// 解析URL获取域名
	u, err := url.Parse(websiteURL)
	if err != nil {
		return "", fmt.Errorf("解析URL失败: %w", err)
	}

	domain := u.Hostname()

	// 使用多个favicon源
	faviconURLs := []string{
		fmt.Sprintf("https://www.google.com/s2/favicons?domain=%s&sz=128", domain),
		fmt.Sprintf("https://favicon.yandex.net/favicon/%s", domain),
	}

	// 尝试从各个源下载
	for _, faviconURL := range faviconURLs {
		iconPath, err := s.DownloadIcon(faviconURL)
		if err == nil {
			return iconPath, nil
		}
	}

	return "", fmt.Errorf("所有favicon源都下载失败")
}

// SaveUploadedIcon 保存上传的图标
func (s *IconService) SaveUploadedIcon(filename string, data []byte) (string, error) {
	ext := strings.ToLower(filepath.Ext(filename))
	return s.saveIcon(filename, data, ext)
}

// DeleteIcon 删除本地图标文件
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

// GetIconDir 获取图标目录路径
func (s *IconService) GetIconDir() string {
	return s.iconDir
}
