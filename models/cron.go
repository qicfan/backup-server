package models

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/qicfan/backup-server/helpers"
	"github.com/robfig/cron/v3"
	"gorm.io/gorm"
)

var GlobalCron *cron.Cron
var refreshPhotoCollectionLock bool = false

func RefreshPhotoCollection() {
	if refreshPhotoCollectionLock {
		helpers.AppLogger.Warn("扫描本地文件任务 正在执行，跳过本次调度")
		return
	}
	helpers.AppLogger.Infof("扫描本地文件任务 开始执行")
	refreshPhotoCollectionLock = true
	defer func() {
		refreshPhotoCollectionLock = false
	}()
	// 查询数据库中的所有数据
	// 生成路径到ID的映射
	// 如果本地存在则跳过，否则插入，然后删除映射关系
	// 最后留在映射关系中的记录就是数据库中存在但本地不存在的，删除这些记录
	dbPathMap := make(map[string]string)
	photos := make([]Photo, 0)
	helpers.Db.Select("id", "path", "checksum").Find(&photos)
	for _, p := range photos {
		dbPathMap[p.Path] = p.Checksum
	}
	filepath.Walk(helpers.UPLOAD_ROOT_DIR, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		relPath := strings.TrimPrefix(strings.TrimPrefix(path, helpers.UPLOAD_ROOT_DIR), string(os.PathSeparator))
		// 检查是否在数据库中存在
		if _, exists := dbPathMap[relPath]; exists {
			// 存在，跳过
			delete(dbPathMap, relPath)
			return nil
		}
		name := info.Name()
		var livePhotoVideoPath string = ""
		var livePhotoVideoFullPath string = ""
		var photoType PhotoType = PhotoTypeNormal
		// 查找是否有同名的视频文件
		ext := filepath.Ext(name)
		baseName := strings.TrimSuffix(path, ext)
		ext = strings.ToLower(ext)
		if ext == ".chunk" {
			return nil
		}
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
			photoType = PhotoTypeVideo
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
			// 读取文件的修改时间
			checksum, _ := helpers.FileSHA1(path)
			// 检查checksum是否存在
			if exists, _ := CheckPhotoChecksum(checksum); exists {
				// helpers.AppLogger.Infof("Checksum exists，跳过:%s => %s", relPath, checksum)
				return nil
			}
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
			// if photo.Checksum == "" {
			// 	checksum, _ := helpers.FileSHA1(path)
			// 	helpers.AppLogger.Infof("%s 数据库记录需要哈希摘要: PreChecksum %d Checksum %d", relPath, checksum)
			// 	photo.Checksum = checksum
			// 	photo.Update()
			// }
		}
		return nil
	})
	// 删除数据库中多余的记录
	for p, checksum := range dbPathMap {
		helpers.AppLogger.Infof("删除数据库中多余的记录: %s => %s", p, checksum)
		helpers.EnqueueDBWriteSync(func(db *gorm.DB) error {
			return db.Where("path = ?", p).Delete(&Photo{}).Error
		})
	}
	helpers.AppLogger.Infof("扫描本地文件任务 执行完成")
}

// 初始化定时任务
func InitCron() {
	if GlobalCron != nil {
		GlobalCron.Stop()
	}
	GlobalCron = cron.New()

	GlobalCron.AddFunc("*/5 * * * *", func() {
		// 每30分钟刷新照片集合
		// helpers.AppLogger.Info("刷新照片集合")
		RefreshPhotoCollection()
	})
	helpers.AppLogger.Info("定时任务已初始化，开始运行")
	GlobalCron.Start()
}
