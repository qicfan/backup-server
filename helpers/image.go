package helpers

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// 返回缩略图的保存路径
// srcFilePath: 原图路径，包含文件名
func GetThumbnailFilename(srcFilePath string, size string) string {
	relPath := filepath.Dir(srcFilePath)
	fullPath := filepath.Join(RootDir, "config", "thumbnails", relPath)
	os.MkdirAll(fullPath, 0755) // 保证路径存在
	thumbnailName := filepath.Join(fullPath, fmt.Sprintf("%s_%s.jpg", filepath.Base(srcFilePath), size))
	return thumbnailName
}

// 返回转码后的新资源路径
func GetConvertFilename(srcFilePath string, newExt string) string {
	relPath := filepath.Dir(srcFilePath)
	fullPath := filepath.Join(RootDir, "config", "converted", relPath)
	os.MkdirAll(fullPath, 0755) // 保证路径存在
	convertName := filepath.Join(fullPath, fmt.Sprintf("%s%s", filepath.Base(srcFilePath), newExt))
	return convertName
}

// 生成缩略图
// path: 原图路径
// size: 缩略图尺寸，如 "100x100"
// 返回缩略图的完整文件路径
func Thumbnail(path, size string) (string, error) {
	exeCommand := "magick"
	if _, err := exec.LookPath(exeCommand); err != nil {
		AppLogger.Errorf("%s未安装: %v", exeCommand, err)
		exeCommand = "convert"
		if _, err := exec.LookPath(exeCommand); err != nil {
			return "", fmt.Errorf("%s未安装: %v", exeCommand, err)
		}
	}
	srcFullPath := filepath.Join(UPLOAD_ROOT_DIR, path)
	thumbnailPath := GetThumbnailFilename(path, size)
	if strings.HasPrefix(path, "/") {
		srcFullPath = path
		thumbnailPath = fmt.Sprintf("%s_%s.jpg", srcFullPath, size)
		AppLogger.Infof("使用绝对路径:%s, 缩略图路径：%s", path, thumbnailPath)
	} else {
		AppLogger.Infof("使用相对路径:%s => %s, 缩略图路径：%s", path, srcFullPath, thumbnailPath)
	}

	if !FileExists(thumbnailPath) {
		// 执行 ImageMagick 缩略图命令，强制输出jpg
		cmd := exec.Command(exeCommand, srcFullPath, "-thumbnail", size, thumbnailPath)
		output, err := cmd.CombinedOutput()
		if err != nil {
			AppLogger.Errorf("生成缩略图失败: %v, 输出: %s", err, string(output))
			return "", fmt.Errorf("生成缩略图失败: %v, 输出: %s", err, string(output))
		}
	}
	return thumbnailPath, nil
}

// HeicToJpg 将 HEIC 图片转为 JPG 格式
func HEICToJPG(inputPath string) (string, error) {
	// 检查ImageMagick是否安装
	exeCommand := "magick"
	if _, err := exec.LookPath(exeCommand); err != nil {
		AppLogger.Errorf("%s未安装: %v", exeCommand, err)
		exeCommand = "convert"
		if _, err := exec.LookPath(exeCommand); err != nil {
			return "", fmt.Errorf("%s未安装: %v", exeCommand, err)
		}
	}
	outputPath := GetConvertFilename(inputPath, ".jpg")
	if FileExists(outputPath) {
		return outputPath, nil
	}
	srcFullPath := filepath.Join(UPLOAD_ROOT_DIR, inputPath)
	// 执行转换命令
	cmd := exec.Command(exeCommand, srcFullPath, outputPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		AppLogger.Errorf("转换失败: %v, 输出: %s", err, string(output))
		return "", fmt.Errorf("转换失败: %v, 输出: %s", err, string(output))
	}

	AppLogger.Infof("转换成功: %s -> %s", inputPath, outputPath)
	return outputPath, nil
}

// 将图片进行转码
// srcPath：源图片路径，不包含UPLOAD_ROOT_DIR
// format: 目标图片格式，如 ".jpg", ".png"
func TransImage(srcPath string, format string) (string, string, error) {
	// 检查ImageMagick是否安装
	exeCommand := "magick"
	if _, err := exec.LookPath(exeCommand); err != nil {
		AppLogger.Errorf("%s未安装: %v", exeCommand, err)
		exeCommand = "convert"
		if _, err := exec.LookPath(exeCommand); err != nil {
			return "", "", fmt.Errorf("%s未安装: %v", exeCommand, err)
		}
	}
	srcFullPath := filepath.Join(UPLOAD_ROOT_DIR, srcPath)
	destPath := fmt.Sprintf("%s%s", srcPath, format)
	destFullPath := filepath.Join(UPLOAD_ROOT_DIR, destPath)

	// 执行转换命令
	cmd := exec.Command(exeCommand, srcFullPath, destFullPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		AppLogger.Errorf("转换失败: %v, 输出: %s", err, string(output))
		return "", "", fmt.Errorf("转换失败: %v, 输出: %s", err, string(output))
	}

	AppLogger.Infof("转换成功: %s -> %s", srcPath, destPath)
	return destPath, destFullPath, nil
}
