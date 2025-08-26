package helpers

import (
	"fmt"
	"os/exec"
)

// MovToMp4 将 mov 视频转为 mp4 格式
func MovToMp4(srcPath, dstPath string) error {
	cmd := exec.Command("ffmpeg", "-y", "-i", srcPath, "-c:v", "copy", "-c:a", "aac", dstPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg 转码失败: %v, 输出: %s", err, string(output))
	}
	return nil
}

// ExtractVideoThumbnail 提取视频第一秒画面生成缩略图
func ExtractVideoThumbnail(videoPath, thumbPath string) error {
	cmd := exec.Command("ffmpeg", "-y", "-i", videoPath, "-ss", "1", "-vframes", "1", thumbPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg 生成缩略图失败: %v, 输出: %s", err, string(output))
	}
	return nil
}
