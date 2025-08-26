package models

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/qicfan/backup-server/helpers"
	"github.com/robfig/cron/v3"
	"gorm.io/gorm"
)

var (
	ImageExtensions = []string{".jpg", ".jpeg", ".png", ".gif", ".bmp", ".webp"}
	VideoExtensions = []string{".mp4", ".avi", ".mov", ".mkv", ".wmv", ".flv", ".webm"}
	UPLOAD_ROOT_DIR = "/path/to/upload/root" // 请根据实际情况修改
)

func isImage(ext string) bool {
	ext = strings.ToLower(ext)
	for _, e := range ImageExtensions {
		if ext == e {
			return true
		}
	}
	return false
}

func isVideo(ext string) bool {
	ext = strings.ToLower(ext)
	for _, e := range VideoExtensions {
		if ext == e {
			return true
		}
	}
	return false
}

var GlobalCron *cron.Cron

func RefreshPhotoCollection() {
	// TODO: 实现照片集合的刷新逻辑
	filepath.Walk(helpers.UPLOAD_ROOT_DIR, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		relPath := strings.TrimPrefix(path, helpers.UPLOAD_ROOT_DIR)
		name := info.Name()
		var livePhotoVideoPath string = ""
		var photoType PhotoType = PhotoTypeNormal
		// 查找是否有同名的视频文件
		ext := strings.ToLower(filepath.Ext(name))
		baseName := strings.TrimSuffix(path, ext)
		// 处理苹果的动态图片
		if ext == ".heic" {
			// 查询是否有同名的.mov文件
			movPath := baseName + ".mov"
			livePhotoVideoPath = strings.TrimPrefix(movPath, helpers.UPLOAD_ROOT_DIR)
		}
		if isImage(ext) {
			// 查找是否有同名的mp4文件
			mp4Path := baseName + ".mp4"
			livePhotoVideoPath = strings.TrimPrefix(mp4Path, helpers.UPLOAD_ROOT_DIR)
		}
		_, statErr := os.Stat(livePhotoVideoPath)
		if livePhotoVideoPath != "" && statErr == nil {
			photoType = PhotoTypeLivePhoto
		}
		if isImage(ext) || isVideo(ext) {
			// 查询数据库是否存在
			photo, photoGetErr := GetPhotoByPath(relPath)
			if photoGetErr != nil && photoGetErr != gorm.ErrRecordNotFound {
				// 没有找到记录，插入
				helpers.AppLogger.Errorf("%s 没有数据库记录，准备插入: ", relPath)
				if insertErr := InsertPhoto(name, relPath, info.Size(), photoType, livePhotoVideoPath); insertErr != nil {
					helpers.AppLogger.Error("插入数据库失败: ", insertErr)
				}
				return nil
			}
			if photoGetErr == nil && photo != nil {
				// 记录存在，检查是否需要更新

			}
		}
		return nil
	})
}

// 初始化定时任务
func InitCron() {
	if GlobalCron != nil {
		GlobalCron.Stop()
	}
	GlobalCron = cron.New()

	GlobalCron.AddFunc("*/5 * * * *", func() {
		// 每5分钟刷新照片集合
		helpers.AppLogger.Info("刷新照片集合")
		RefreshPhotoCollection()
	})
	helpers.AppLogger.Info("定时任务已初始化，开始运行")
	GlobalCron.Start()
}
