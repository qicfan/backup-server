package helpers

import (
	"crypto/sha1"
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
	ImageExtensions = []string{".jpg", ".jpeg", ".png", ".gif", ".bmp", ".webp", ".heic"}
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

// 计算文件整体SHA1
func FileSHA1(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha1.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// 计算文件64kb到65kb的sha1，如果文件大小不足65kb则计算文件最后1kb的sha1
func FileHeadSHA1(path string) (string, error) {
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
	var buf []byte
	if size >= 65*1024 {
		// 读取64kb到65kb区间
		buf = make([]byte, 1024)
		if _, err := f.Seek(64*1024, io.SeekStart); err != nil {
			return "", err
		}
		n, err := f.Read(buf)
		if err != nil && err != io.EOF {
			return "", err
		}
		buf = buf[:n]
	} else {
		// 文件不足65kb，读取最后1kb
		readSize := int64(1024)
		if size < 1024 {
			readSize = size
		}
		buf = make([]byte, readSize)
		if _, err := f.Seek(size-readSize, io.SeekStart); err != nil {
			return "", err
		}
		n, err := f.Read(buf)
		if err != nil && err != io.EOF {
			return "", err
		}
		buf = buf[:n]
	}
	h := sha1.Sum(buf)
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
