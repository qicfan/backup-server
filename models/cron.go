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
		var livePhotoVideoFullPath1 string = ""
		var photoType PhotoType = PhotoTypeNormal
		isLivePhotoVideo := false
		// 查找是否有同名的视频文件
		ext := filepath.Ext(name)
		baseName := strings.TrimSuffix(path, ext)
		ext = strings.ToLower(ext)
		if helpers.IsImage(name) {
			// 查找是否有同名的mp4文件
			livePhotoVideoFullPath = baseName + ".mp4"
			livePhotoVideoFullPath1 = baseName + ".MP4"
		}
		// 处理苹果的动态图片
		if ext == ".heic" {
			// 查询是否有同名的.mov文件
			livePhotoVideoFullPath = baseName + ".mov"
			livePhotoVideoFullPath1 = baseName + ".MOV"
		}
		if ext == ".mp4" {
			// 查询是否有同名的.jpg文件
			livePhotoVideoFullPath = baseName + ".jpg"
			livePhotoVideoFullPath1 = baseName + ".JPG"
			isLivePhotoVideo = true
		}
		if ext == ".mov" {
			// 查询是否有同名的.heic文件
			livePhotoVideoFullPath = baseName + ".heic"
			livePhotoVideoFullPath1 = baseName + ".HEIC"
			isLivePhotoVideo = true
		}
		// helpers.AppLogger.Infof("扩展名：%s, 文件名：%s, 动态照片的视频文件路径：%s，大写扩展名：%s", ext, baseName, livePhotoVideoFullPath, livePhotoVideoFullPath1)
		if livePhotoVideoFullPath != "" && helpers.FileExists(livePhotoVideoFullPath) {
			photoType = PhotoTypeLivePhoto
			if !isLivePhotoVideo {
				livePhotoVideoPath = strings.TrimPrefix(strings.TrimPrefix(livePhotoVideoFullPath, helpers.UPLOAD_ROOT_DIR), string(os.PathSeparator))
			}
		} else {
			if livePhotoVideoFullPath1 != "" && helpers.FileExists(livePhotoVideoFullPath1) {
				photoType = PhotoTypeLivePhoto
				if !isLivePhotoVideo {
					livePhotoVideoPath = strings.TrimPrefix(strings.TrimPrefix(livePhotoVideoFullPath1, helpers.UPLOAD_ROOT_DIR), string(os.PathSeparator))
				}
			}
		}
		if helpers.IsImage(name) || helpers.IsVideo(name) {
			// 查询数据库是否存在
			photo, photoGetErr := GetPhotoByPath(relPath)
			if photoGetErr != nil && photoGetErr == gorm.ErrRecordNotFound {
				// 入库前计算SHA1
				preChecksum, _ := helpers.FileHeadSHA1(path)
				checksum, _ := helpers.FileSHA1(path)
				// 没有找到记录，插入
				// helpers.AppLogger.Errorf("%s 没有数据库记录，准备插入: ", relPath)
				// 读取文件的修改时间
				modificationTime := info.ModTime().Unix()
				if insertErr := InsertPhoto(name, relPath, info.Size(), photoType, livePhotoVideoPath, "", modificationTime, modificationTime, preChecksum, checksum, 0); insertErr != nil {
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
				if photo.PreChecksum == "" || photo.Checksum == "" {
					// 入库前计算SHA1
					preChecksum, _ := helpers.FileHeadSHA1(path)
					checksum, _ := helpers.FileSHA1(path)
					helpers.AppLogger.Infof("%s 数据库记录需要哈希摘要: PreChecksum %d Checksum %d", relPath, preChecksum, checksum)
					photo.PreChecksum = preChecksum
					photo.Checksum = checksum
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
		// helpers.AppLogger.Info("刷新照片集合")
		RefreshPhotoCollection()
	})
	helpers.AppLogger.Info("定时任务已初始化，开始运行")
	GlobalCron.Start()
}
