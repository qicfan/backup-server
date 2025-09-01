package models

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/qicfan/backup-server/helpers"
	"gorm.io/gorm"
)

type PhotoType int

const (
	PhotoTypeNormal PhotoType = iota + 1
	PhotoTypeVideo
	PhotoTypeLivePhoto
)

type Photo struct {
	BaseModel
	Name               string    `json:"name"`                  // 照片名称，文件名：a.jpg / b.mp4
	Path               string    `json:"path" gorm:"unique"`    // 照片存储路径，包含照片名称，相对helpers.UPLOAD_ROOT_DIR的路径
	Size               int64     `json:"size"`                  // 照片大小
	Type               PhotoType `json:"type"`                  // 照片类型，1-普通照片，2-视频， 3-动态照片
	LivePhotoVideoPath string    `json:"live_photo_video_path"` // 如果是动态照片，这里存储视频的路径，只有动态照片中的图片会保存该字段，如果是动态照片的视频则该字段为空
	FileURI            string    `json:"fileUri"`               // 鸿蒙系统的照片资源的URI，可以用来查询照片是否存在，如果有这个字段代表本地存在该照片
	MTime              int64     `json:"mtime"`                 // 照片的最后修改时间，Unix时间戳，单位秒
	CTime              int64     `json:"ctime"`                 // 照片的创建时间，Unix时间戳，单位秒
	PreChecksum        string    `json:"pre_checksum"`          // 照片的64kb-65kb之间的1kb做sha1来判断是否一致，如果这个值有重复则判断完整的checksum是否一致
	Checksum           string    `json:"checksum"`              // 照片的SHA1哈希值，用来判定照片的唯一性
	SourceId           uint      `json:"source_id"`             // 照片的来源ID，转码前的原图ID
}

// 返回绝对路径
func (p *Photo) FullPath() string {
	return filepath.Join(helpers.UPLOAD_ROOT_DIR, p.Path)
}

// 更新照片信息
func (p *Photo) Update() error {
	return helpers.EnqueueDBWriteSync(func(db *gorm.DB) error {
		return db.Save(p).Error
	})
}

// 插入一张照片
func InsertPhoto(name string, path string, size int64, photoType PhotoType, livePhotoVideoPath string, fileUri string, mtime int64, ctime int64, preChecksum string, checksum string, sourceId uint) error {
	if mtime == 0 {
		mtime = time.Now().Unix()
	}
	if ctime == 0 {
		ctime = time.Now().Unix()
	}
	photo := Photo{
		Name:               name,
		Path:               strings.TrimPrefix(path, string(os.PathSeparator)),
		Size:               size,
		Type:               photoType,
		LivePhotoVideoPath: strings.TrimPrefix(livePhotoVideoPath, string(os.PathSeparator)),
		FileURI:            fileUri,
		MTime:              mtime,
		CTime:              ctime,
		PreChecksum:        preChecksum,
		Checksum:           checksum,
		SourceId:           sourceId,
	}
	fullPath := photo.FullPath()
	if !helpers.FileExists(fullPath) {
		// 文件不存在
		return os.ErrNotExist
	}
	return helpers.EnqueueDBWriteSync(func(db *gorm.DB) error {
		return db.Create(&photo).Error
	})
}

// 通过ID查询照片
func GetPhotoById(id uint) (*Photo, error) {
	var photo Photo
	if err := helpers.Db.Where("id = ?", id).First(&photo).Error; err != nil {
		return nil, err
	}
	return &photo, nil
}

// 通过路径查询照片
func GetPhotoByPath(path string) (*Photo, error) {
	var photo Photo
	if err := helpers.Db.Where("path = ?", path).First(&photo).Error; err != nil {
		return nil, err
	}
	return &photo, nil
}

// 通过fileUri查找照片
func GetPhotoByFileUri(fileUri string) (*Photo, error) {
	var photo Photo
	if err := helpers.Db.Where("file_uri = ?", fileUri).First(&photo).Error; err != nil {
		return nil, err
	}
	return &photo, nil
}

// 更新sourceId对应的记录的fileUri字段
func UpdatePhotoFileUri(sourceId uint, fileUri string) error {
	return helpers.EnqueueDBWriteSync(func(db *gorm.DB) error {
		return db.Model(&Photo{}).Where("source_id = ?", sourceId).Update("file_uri", fileUri).Error
	})
}

// 判断PreChecksum是否存在
func CheckPhotoPreChecksum(PreChecksum string) (bool, error) {
	var photo Photo
	if err := helpers.Db.Where("pre_checksum = ?", PreChecksum).First(&photo).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// 判断checksum是否存在
func CheckPhotoChecksum(Checksum string) (bool, error) {
	var photo Photo
	if err := helpers.Db.Where("checksum = ?", Checksum).First(&photo).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// 根据路径删除一张照片
func DeletePhotoByPath(path string) error {
	photo, err := GetPhotoByPath(path)
	if err != nil {
		return err
	}
	// 数据库中先删除
	dbErr := helpers.EnqueueDBWriteSync(func(db *gorm.DB) error {
		return db.Delete(&photo).Error
	})
	if dbErr != nil {
		return dbErr
	}
	// 删除本地文件
	if err := os.Remove(photo.FullPath()); err != nil {
		return err
	}
	return nil
}

// 查询照片列表
func ListPhotos(page int, pageSize int) (int64, []*Photo, error) {
	var photos []*Photo = make([]*Photo, 0)
	// 先查询总数
	var total int64
	if err := helpers.Db.Model(&Photo{}).Where("source_id=0 AND (type <> ? OR (type=? AND live_photo_video_path != ''))", PhotoTypeLivePhoto, PhotoTypeLivePhoto).Count(&total).Error; err != nil {
		return 0, nil, err
	}

	// 再分页查询列表
	if err := helpers.Db.Offset((page-1)*pageSize).Limit(pageSize).Where("source_id=0 AND (type <> ? OR (type=? AND live_photo_video_path != ''))", PhotoTypeLivePhoto, PhotoTypeLivePhoto).Order("m_time DESC").Find(&photos).Error; err != nil {
		helpers.AppLogger.Error("查询照片列表失败: ", err)
		return 0, nil, err
	}
	return total, photos, nil
}
