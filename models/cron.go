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
		ext := filepath.Ext(name)
		baseName := strings.TrimSuffix(path, ext)
		// ext = strings.ToLower(ext)
		needProcess := false
		if helpers.IsImage(path) {
			needProcess = true
			// 查找是否有同名的mp4或mov文件
			var ext []string = make([]string, 4)
			ext = append(ext, ".mp4", ".MP4", ".mov", ".MOV")
			for _, e := range ext {
				livePhotoVideoFullPath = baseName + e
				if helpers.FileExists(livePhotoVideoFullPath) {
					photoType = PhotoTypeLivePhoto
					livePhotoVideoPath = strings.TrimPrefix(strings.TrimPrefix(livePhotoVideoFullPath, helpers.UPLOAD_ROOT_DIR), string(os.PathSeparator))
					break
				}
			}
		}
		if helpers.IsVideo(path) {
			needProcess = true
			// 查询是否有同名的jpg或者heic文件
			var ext []string = make([]string, 4)
			ext = append(ext, ".jpg", ".JPG", ".heic", ".HEIC")
			for _, e := range ext {
				livePhotoVideoFullPath = baseName + e
				if helpers.FileExists(livePhotoVideoFullPath) {
					photoType = PhotoTypeLivePhoto
					break
				}
			}
		}
		if !needProcess {
			return nil
		}
		// 查询数据库是否存在
		photo, photoGetErr := GetPhotoByPath(relPath)
		if photoGetErr != nil && photoGetErr == gorm.ErrRecordNotFound {
			// 入库前计算SHA1
			// preChecksum, _ := helpers.FileHeadSHA1(path)
			checksum, _ := helpers.FileSHA1(path)
			// 没有找到记录，插入
			// helpers.AppLogger.Errorf("%s 没有数据库记录，准备插入: ", relPath)
			// 读取文件的修改时间
			modificationTime := info.ModTime().Unix()
			if insertErr := InsertPhoto(name, relPath, info.Size(), photoType, livePhotoVideoPath, "", modificationTime, modificationTime, checksum, 0); insertErr != nil {
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
			if photo.Checksum == "" {
				// 入库前计算SHA1
				// preChecksum, _ := helpers.FileHeadSHA1(path)
				checksum, _ := helpers.FileSHA1(path)
				helpers.AppLogger.Infof("%s 数据库记录需要哈希摘要: PreChecksum %d Checksum %d", relPath, checksum)
				// photo.PreChecksum = preChecksum
				photo.Checksum = checksum
				photo.Update()
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
		// helpers.AppLogger.Info("刷新照片集合")
		RefreshPhotoCollection()
	})
	helpers.AppLogger.Info("定时任务已初始化，开始运行")
	GlobalCron.Start()
}
