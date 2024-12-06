package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/schollz/progressbar/v3"
)

const (
	baseURL    = "https://esolutions.dpd.com/partnerloesungen/hazdistributionservice.aspx"
	maxRetries = 3
)

// VersionResponse 用于解析版本号响应
type VersionResponse struct {
	Version string `json:"version"`
}

// 获取当前版本号
func getCurrentVersion() (string, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// 使用baseURL获取版本号
	resp, err := client.Get(baseURL)
	if err != nil {
		return "", fmt.Errorf("获取版本号失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("获取版本号时服务器返回错误状态码: %d", resp.StatusCode)
	}

	// 解析JSON响应
	var versionResp VersionResponse
	if err := json.NewDecoder(resp.Body).Decode(&versionResp); err != nil {
		return "", fmt.Errorf("解析版本号响应失败: %v", err)
	}

	if versionResp.Version == "" {
		return "", fmt.Errorf("获取到的版本号为空")
	}

	return versionResp.Version, nil
}

func main() {
	// 首先获取版本号
	version, err := getCurrentVersion()
	if err != nil {
		fmt.Printf("获取版本号失败: %v\n", err)
		return
	}
	fmt.Printf("当前版本号: %s\n", version)

	// 使用版本号构建下载URL
	downloadURL := fmt.Sprintf("%s?version=%s", baseURL, version)

	// 尝试下载文件
	for i := 0; i < maxRetries; i++ {
		if err := downloadFile(downloadURL, version); err != nil {
			fmt.Printf("下载尝试 %d 失败: %v\n", i+1, err)
			if i < maxRetries-1 {
				fmt.Println("等待 5 秒后重试...")
				time.Sleep(5 * time.Second)
				continue
			}
		} else {
			return
		}
	}
	fmt.Println("达到最大重试次数，下载失败")
}

func downloadFile(url string, version string) error {
	client := &http.Client{
		Timeout: 10 * time.Minute,
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("创建请求失败: %v", err)
	}

	fmt.Println("正在下载文件...")
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("下载失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("服务器返回错误状态码: %d", resp.StatusCode)
	}

	// 生成文件名
	filename := fmt.Sprintf("dpd_distribution_HAZ_%s.zip", version)

	downloadDir := "downloads"
	if err := os.MkdirAll(downloadDir, 0755); err != nil {
		return fmt.Errorf("创建下载目录失败: %v", err)
	}

	filepath := filepath.Join(downloadDir, filename)
	file, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("创建文件失败: %v", err)
	}
	defer file.Close()

	bar := progressbar.DefaultBytes(
		resp.ContentLength,
		"下载进度",
	)

	_, err = io.Copy(io.MultiWriter(file, bar), resp.Body)
	if err != nil {
		return fmt.Errorf("保存文件失败: %v", err)
	}

	fmt.Printf("\n文件下载成功: %s\n", filepath)
	return nil
}
