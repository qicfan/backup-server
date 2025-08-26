package helpers

import (
	"fmt"

	"os/exec"

	"github.com/disintegration/imaging"
)

func Thumbnail(path, thumbnailPath string, width int, height int) error {
	// TODO 如果是HEIC图片需要先转成jpg
	img, err := imaging.Open(path)
	if err != nil {
		return fmt.Errorf("打开图片失败: %v", err)
	}
	thumb := imaging.Thumbnail(img, width, height, imaging.Lanczos)
	err = imaging.Save(thumb, thumbnailPath)
	if err != nil {
		return fmt.Errorf("保存缩略图失败: %v", err)
	}
	return nil
}

// HeicToJpg 将 HEIC 图片转为 JPG 格式
func HEICToJPG(inputPath, outputPath string) error {
	cmd := exec.Command("ffmpeg", "-y", "-i", inputPath, outputPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg 转换失败: %v, 输出: %s", err, string(output))
	}
	return nil
}
