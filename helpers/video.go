package helpers

import (
	"fmt"
	"os/exec"
	"path/filepath"
)

// MovToMp4 将 mov 视频转为 mp4 格式
func MovToMp4(srcPath string) (string, error) {
	dstPath := GetConvertFilename(srcPath, ".mp4")
	if FileExists(dstPath) {
		return dstPath, nil
	}
	srcFullPath := filepath.Join(UPLOAD_ROOT_DIR, srcPath)
	cmd := exec.Command("ffmpeg", "-y", "-i", srcFullPath, "-c:v", "copy", "-c:a", "aac", dstPath)
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
	srcFullPath := filepath.Join(UPLOAD_ROOT_DIR, videoPath)
	if !FileExists(coverFullPath) {
		cmd := exec.Command("ffmpeg", "-y", "-i", srcFullPath, "-ss", "1", "-vframes", "1", coverFullPath)
		output, err := cmd.CombinedOutput()
		if err != nil {
			AppLogger.Errorf("提取视频缩略图失败: %v, 输出: %s", err, string(output))
			return "", fmt.Errorf("ffmpeg 生成缩略图失败: %v, 输出: %s", err, string(output))
		}
		AppLogger.Infof("提取视频缩略图成功: %s", coverFullPath)
	}
	// rootDir := filepath.Join(RootDir, "config")
	// coverPath := strings.TrimPrefix(strings.Replace(coverFullPath, rootDir, "", 1), string(os.PathSeparator))
	thumbPath, err := Thumbnail(coverFullPath, size)
	if err != nil {
		AppLogger.Errorf("生成缩略图 %s 失败: %v", coverFullPath, err)
		return "", err
	}
	AppLogger.Infof("生成缩略图成功: %s => %s", coverFullPath, thumbPath)
	return thumbPath, nil
}
