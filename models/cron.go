package models

import (
	"os"
	"path/filepath"
	"strings"

	"sync"

	"github.com/qicfan/backup-server/helpers"
	"github.com/robfig/cron/v3"
	"gorm.io/gorm"
)

var GlobalCron *cron.Cron
var refreshPhotoCollectionLock sync.Mutex

func RefreshPhotoCollection() {
	if !refreshPhotoCollectionLock.TryLock() {
		helpers.AppLogger.Warn("RefreshPhotoCollection 正在执行，跳过本次调度")
		return
	}
	defer refreshPhotoCollectionLock.Unlock()
	filepath.Walk(helpers.UPLOAD_ROOT_DIR, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		relPath := strings.TrimPrefix(strings.TrimPrefix(path, helpers.UPLOAD_ROOT_DIR), string(os.PathSeparator))
		name := info.Name()
		var livePhotoVideoPath string = ""
		var livePhotoVideoFullPath string = ""
		var photoType PhotoType = PhotoTypeNormal
		// 查找是否有同名的视频文件
		ext := strings.ToLower(filepath.Ext(name))
		baseName := strings.TrimSuffix(path, ext)
		// 处理苹果的动态图片
		if ext == ".heic" {
			// 查询是否有同名的.mov文件
			livePhotoVideoFullPath = baseName + ".mov"
		}
		if helpers.IsImage(ext) {
			// 查找是否有同名的mp4文件
			livePhotoVideoFullPath = baseName + ".mp4"
		}
		if livePhotoVideoFullPath != "" && helpers.FileExists(livePhotoVideoFullPath) {
			photoType = PhotoTypeLivePhoto
			livePhotoVideoPath = strings.TrimPrefix(strings.TrimPrefix(livePhotoVideoFullPath, helpers.UPLOAD_ROOT_DIR), string(os.PathSeparator))
		}
		if helpers.IsImage(ext) || helpers.IsVideo(ext) {
			// 查询数据库是否存在
			photo, photoGetErr := GetPhotoByPath(relPath)
			if photoGetErr != nil && photoGetErr == gorm.ErrRecordNotFound {
				// 没有找到记录，插入
				helpers.AppLogger.Errorf("%s 没有数据库记录，准备插入: ", relPath)
				// 读取文件的修改时间
				modificationTime := info.ModTime().Unix()
				if insertErr := InsertPhoto(name, relPath, info.Size(), photoType, livePhotoVideoPath, "", modificationTime, modificationTime); insertErr != nil {
					helpers.AppLogger.Error("插入数据库失败: ", insertErr)
				}
				return nil
			}
			if photoGetErr == nil && photo != nil {
				// 记录存在，检查是否需要更新
				if photo.Type != photoType || photo.LivePhotoVideoPath != livePhotoVideoPath {
					helpers.AppLogger.Infof("%s 数据库记录需要更新: %d => %d", relPath, photo.Type, photoType)
					photo.Type = photoType
					photo.LivePhotoVideoPath = livePhotoVideoPath
					photo.Update()
				}
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
