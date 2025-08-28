package helpers

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

var (
	ImageExtensions = []string{".jpg", ".jpeg", ".png", ".gif", ".bmp", ".webp"}
	VideoExtensions = []string{".mp4", ".avi", ".mov", ".mkv", ".wmv", ".flv", ".webm"}
)

func IsImage(ext string) bool {
	ext = strings.ToLower(ext)
	for _, e := range ImageExtensions {
		if ext == e {
			return true
		}
	}
	return false
}

func IsVideo(ext string) bool {
	ext = strings.ToLower(ext)
	for _, e := range VideoExtensions {
		if ext == e {
			return true
		}
	}
	return false
}

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

// FileExists checks if a given file exists.
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil || os.IsExist(err)
}

// 计算文件整体SHA256
func FileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// 计算文件前512字节SHA256
func FileHeadSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	buf := make([]byte, 512)
	n, err := f.Read(buf)
	if err != nil && err != io.EOF {
		return "", err
	}
	h := sha256.Sum256(buf[:n])
	return hex.EncodeToString(h[:]), nil
}

// 计算文件最后512字节SHA256
func FileTailSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		return "", err
	}
	size := fi.Size()
	if size < 512 {
		// 文件不足512字节，直接读全部
		buf := make([]byte, size)
		_, err := f.ReadAt(buf, 0)
		if err != nil && err != io.EOF {
			return "", err
		}
		h := sha256.Sum256(buf)
		return hex.EncodeToString(h[:]), nil
	}
	buf := make([]byte, 512)
	_, err = f.ReadAt(buf, size-512)
	if err != nil && err != io.EOF {
		return "", err
	}
	h := sha256.Sum256(buf)
	return hex.EncodeToString(h[:]), nil
}

func Base64Decode(s string) (string, error) {
	decoded, err := base64.URLEncoding.DecodeString(s)
	if err != nil {
		return "", err
	}
	return string(decoded), nil
}

func BytesSHA256(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}
