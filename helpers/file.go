package helpers

import (
	"net/http"
	"os"
	"path/filepath"
)

func CleanupUploadingFiles() {
	root := UPLOAD_ROOT_DIR
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			AppLogger.Error("Walk error:", err)
			return nil
		}
		// 取扩展名
		ext := filepath.Ext(info.Name())
		if ext == ".uploading" {
			if rmErr := os.Remove(path); rmErr != nil {
				AppLogger.Errorf("删除上传临时文件失败： %s: %v", path, rmErr)
			} else {
				AppLogger.Infof("删除上传临时文件: %s", path)
			}
		}
		return nil
	})
	if err != nil {
		AppLogger.Error("递归读取目录时发生错误:", err)
	}
}

// GetFileMIME 返回文件的 MIME 类型
func GetFileMIME(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	buf := make([]byte, 512)
	n, err := f.Read(buf)
	if err != nil {
		return "", err
	}
	mimeType := http.DetectContentType(buf[:n])
	return mimeType, nil
}
