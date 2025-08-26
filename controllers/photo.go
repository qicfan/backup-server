package controllers

import (
	"net/url"
	"os"
	"path/filepath"

	"fmt"

	"github.com/disintegration/imaging"
	"github.com/gin-gonic/gin"
	"github.com/qicfan/backup-server/helpers"
	"github.com/qicfan/backup-server/models"
)

func HandleGetThumbnail(c *gin.Context) {
	path := c.Param("path") // 相对路径，不以 / 开头，相对helpers.UPLOAD_ROOT_DIR的路径，需要做urldecode
	decodedPath, err := url.QueryUnescape(path)
	if err != nil {
		helpers.AppLogger.Errorf("路径解码失败: %v", err)
		c.JSON(400, APIResponse[any]{Code: BadRequest, Message: "路径解码失败", Data: nil})
		return
	}
	path = decodedPath
	fullPath := filepath.Join(helpers.UPLOAD_ROOT_DIR, path)
	size := c.Param("size") // 尺寸 100x100格式
	// TODO 实现获取缩略图的逻辑
	helpers.AppLogger.Infof("获取缩略图: %s, 尺寸: %s", path, size)
	// 根据path查询photo
	if _, err := models.GetPhotoByPath(path); err != nil {
		helpers.AppLogger.Errorf("根据路径查询照片失败: %v", err)
		c.JSON(404, APIResponse[any]{Code: BadRequest, Message: "照片未找到", Data: nil})
		return
	}
	// 检查path是否存在
	if _, err := os.Stat(fullPath); err != nil {
		helpers.AppLogger.Errorf("检查照片路径失败: %v", err)
		c.JSON(404, APIResponse[any]{Code: BadRequest, Message: "照片未找到", Data: nil})
		return
	}
	// 解析尺寸参数
	var width, height int
	_, err = fmt.Sscanf(size, "%dx%d", &width, &height)
	if err != nil || width <= 0 || height <= 0 {
		helpers.AppLogger.Errorf("尺寸参数错误: %v", err)
		c.JSON(400, APIResponse[any]{Code: BadRequest, Message: "尺寸参数错误", Data: nil})
		return
	}
	// TODO 检查缩略图文件是否存在，如果存在直接返回文件内容

	// 生成缩略图
	img, err := imaging.Open(fullPath)
	if err != nil {
		helpers.AppLogger.Errorf("打开图片失败: %v", err)
		c.JSON(500, APIResponse[any]{Code: BadRequest, Message: "图片打开失败", Data: nil})
		return
	}
	thumb := imaging.Thumbnail(img, width, height, imaging.Lanczos)
	// @TODO 保存缩略图到文件，供下次使用

	// 返回图片内容
	c.Header("Content-Type", "image/jpeg")
	err = imaging.Encode(c.Writer, thumb, imaging.JPEG)
	if err != nil {
		helpers.AppLogger.Errorf("缩略图编码失败: %v", err)
		c.JSON(500, APIResponse[any]{Code: BadRequest, Message: "缩略图编码失败", Data: nil})
		return
	}
}
