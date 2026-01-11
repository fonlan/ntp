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

// IconService 图标服务
type IconService struct {
	iconDir  string
	baseURL  string
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
	if strings.HasPrefix(iconURL, "/data/icons/") {
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

	// 从 URL 中提取文件扩展名
	ext := ".png" // 默认使用 PNG
	u, err := url.Parse(iconURL)
	if err == nil {
		path := u.Path
		if idx := strings.LastIndex(path, "."); idx != -1 {
			potentialExt := strings.ToLower(path[idx:])
			// 只接受常见的图片格式
			switch potentialExt {
			case ".png", ".jpg", ".jpeg", ".gif", ".svg", ".ico", ".webp":
				ext = potentialExt
			}
		}
	}

	// 生成文件名（使用URL的hash）
	hash := sha256.Sum256([]byte(iconURL))
	filename := hex.EncodeToString(hash[:]) + ext
	filePath := filepath.Join(s.iconDir, filename)

	// 确保目录存在
	if err := os.MkdirAll(s.iconDir, 0755); err != nil {
		return "", fmt.Errorf("创建图标目录失败: %w", err)
	}

	// 保存文件
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return "", fmt.Errorf("保存图标失败: %w", err)
	}

	// 返回URL路径
	return "/data/icons/" + filename, nil
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
	// 验证文件扩展名
	ext := strings.ToLower(filepath.Ext(filename))
	if ext != ".png" && ext != ".jpg" && ext != ".jpeg" && ext != ".gif" && ext != ".ico" && ext != ".svg" {
		return "", fmt.Errorf("不支持的文件格式: %s", ext)
	}

	// 生成唯一文件名
	hash := sha256.Sum256(data)
	newFilename := hex.EncodeToString(hash[:]) + ext
	filePath := filepath.Join(s.iconDir, newFilename)

	// 确保目录存在
	if err := os.MkdirAll(s.iconDir, 0755); err != nil {
		return "", fmt.Errorf("创建图标目录失败: %w", err)
	}

	// 保存文件
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return "", fmt.Errorf("保存图标失败: %w", err)
	}

	// 返回URL路径
	return "/data/icons/" + newFilename, nil
}

// DeleteIcon 删除本地图标文件
func (s *IconService) DeleteIcon(iconPath string) error {
	if !strings.HasPrefix(iconPath, "/data/icons/") {
		return nil // 不是本地文件，不删除
	}

	// 提取文件名
	filename := strings.TrimPrefix(iconPath, "/data/icons/")
	filePath := filepath.Join(s.iconDir, filename)

	// 检查文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil // 文件不存在，无需删除
	}

	// 删除文件
	return os.Remove(filePath)
}
