package models

import (
	"os"
	"path/filepath"

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
}

// 返回绝对路径
func (p *Photo) FullPath() string {
	return filepath.Join(helpers.UPLOAD_ROOT_DIR, p.Path)
}

// 插入一张照片
func InsertPhoto(name string, path string, size int64, photoType PhotoType, livePhotoVideoPath string) error {
	photo := Photo{
		Name:               name,
		Path:               path,
		Size:               size,
		Type:               photoType,
		LivePhotoVideoPath: livePhotoVideoPath,
	}
	fullPath := photo.FullPath()
	if _, statErr := os.Stat(fullPath); statErr != nil {
		// 文件不存在
		return statErr
	}
	return helpers.EnqueueDBWriteSync(func(db *gorm.DB) error {
		return db.Create(&photo).Error
	})
}

// 通过路径查询照片
func GetPhotoByPath(path string) (*Photo, error) {
	var photo Photo
	if err := helpers.Db.Where("path = ?", path).First(&photo).Error; err != nil {
		return nil, err
	}
	return &photo, nil
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
