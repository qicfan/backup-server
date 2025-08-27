package helpers

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// MovToMp4 将 mov 视频转为 mp4 格式
func MovToMp4(srcPath string) (string, error) {
	dstPath := GetConvertFilename(srcPath, ".mp4")
	if FileExists(dstPath) {
		return dstPath, nil
	}
	cmd := exec.Command("ffmpeg", "-y", "-i", srcPath, "-c:v", "copy", "-c:a", "aac", dstPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("ffmpeg 转码失败: %v, 输出: %s", err, string(output))
	}
	return dstPath, nil
}

// ExtractVideoThumbnail 提取视频第一秒画面生成缩略图
// 先提取图片，再生成缩略图
func ExtractVideoThumbnail(videoPath string, size string) (string, error) {
	coverFullPath := GetConvertFilename(videoPath, ".jpg")
	if !FileExists(coverFullPath) {
		cmd := exec.Command("ffmpeg", "-y", "-i", videoPath, "-ss", "1", "-vframes", "1", coverFullPath)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("ffmpeg 生成缩略图失败: %v, 输出: %s", err, string(output))
		}
	}
	rootDir := filepath.Join(RootDir, "config", "converted")
	coverPath := strings.TrimPrefix(strings.Replace(coverFullPath, rootDir, "", 1), "/")
	thumbPath, err := Thumbnail(coverPath, size)
	if err != nil {
		return "", err
	}
	return thumbPath, nil
}
