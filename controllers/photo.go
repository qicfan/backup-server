package controllers

import (
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/qicfan/backup-server/helpers"
	"github.com/qicfan/backup-server/models"
)

type DownloadQuery struct {
	Path string `form:"path"` // 相对路径
	Cos  string `form:"cos"`  // 客户端操作系统
	Live int    `form:"live"` // 是否为动态照片
}

// 查询图片的的缩略图，构造一个请求
// http://yourserver/photo/thumbnail/MovieBackup%2FHuawei%20Pura%20X%2F2025%2F8%2F27%2F1.jpg/100x100
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
	helpers.AppLogger.Infof("获取缩略图: %s, 尺寸: %s", path, size)
	// 检查path是否存在
	if !helpers.FileExists(fullPath) {
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
	var thumbnailPath string = ""
	if helpers.IsVideo(filepath.Ext(fullPath)) {
		var videoErr error
		thumbnailPath, videoErr = helpers.ExtractVideoThumbnail(fullPath, size)
		if videoErr != nil {
			helpers.AppLogger.Errorf("生成视频缩略图失败: %v", videoErr)
			c.JSON(500, APIResponse[any]{Code: BadRequest, Message: "生成视频缩略图失败", Data: nil})
			return
		}
	} else {
		var thumbNailErr error
		thumbnailPath, thumbNailErr = helpers.Thumbnail(fullPath, size)
		if thumbNailErr != nil {
			helpers.AppLogger.Errorf("生成缩略图失败: %v", thumbNailErr)
			c.JSON(500, APIResponse[any]{Code: BadRequest, Message: "生成缩略图失败", Data: nil})
			return
		}
	}
	if thumbnailPath == "" {
		c.JSON(500, APIResponse[any]{Code: BadRequest, Message: "生成缩略图失败", Data: nil})
		return
	}
	file, _ := os.ReadFile(thumbnailPath)
	c.Data(
		http.StatusOK,
		"image/jpeg",
		file,
	)
}

// 照片或者视频下载
// http://yourserver/photo/download/?path=MovieBackup%2FHuawei%20Pura%20X%2F2025%2F8%2F27%2F1.jpg&cos=HMOS&live=1
func HandlePhotoDownload(c *gin.Context) {
	var queryParams DownloadQuery
	if c.ShouldBind(&queryParams) == nil {
		helpers.AppLogger.Infof("下载请求参数: %+v", queryParams)
	}
	path := queryParams.Path // 相对路径，不以 / 开头，相对helpers.UPLOAD_ROOT_DIR的路径，需要做urldecode
	clientOS := helpers.ClientOS(queryParams.Cos)
	if clientOS == helpers.UNKNOW {
		clientOS = helpers.HMOS // 默认HMOS
	}
	isLive := queryParams.Live == 1
	fullPath := filepath.Join(helpers.UPLOAD_ROOT_DIR, path)
	helpers.AppLogger.Infof("下载文件: %s", path)
	// 检查文件是否存在
	if !helpers.FileExists(fullPath) {
		c.JSON(404, APIResponse[any]{Code: BadRequest, Message: "文件未找到", Data: nil})
		return
	}
	fi, statErr := os.Stat(fullPath)
	if statErr != nil {
		helpers.AppLogger.Errorf("文件状态获取失败: %v", statErr)
		c.JSON(500, APIResponse[any]{Code: BadRequest, Message: "文件状态获取失败", Data: nil})
		return
	}
	ext := strings.ToLower(filepath.Ext(path))
	// 如果是华为下载苹果的额动态照片资源则进行转码
	if clientOS == helpers.HMOS && isLive {
		if ext == ".heic" {
			// HEIC照片转码成JPG
			jpgPath, err := helpers.HEICToJPG(fullPath)
			if err != nil {
				helpers.AppLogger.Errorf("HEIC转JPG失败: %v", err)
				c.JSON(500, APIResponse[any]{Code: BadRequest, Message: "HEIC转JPG失败", Data: nil})
				return
			}
			fullPath = jpgPath
		}
		if ext == ".mov" {
			// 华为下载.mov，转码成.mp4
			mp4Path, err := helpers.MovToMp4(fullPath)
			if err != nil {
				helpers.AppLogger.Errorf("MOV转MP4失败: %v", err)
				c.JSON(500, APIResponse[any]{Code: BadRequest, Message: "MOV转MP4失败", Data: nil})
				return
			}
			fullPath = mp4Path
		}
	}
	if fi.Size() > 10*1024*1024 {
		// 超过10MB，流式传输
		c.FileAttachment(fullPath, filepath.Base(fullPath))
	} else {
		c.File(fullPath)
	}
}

type PhotoListRequest struct {
	Page     int `json:"page"`
	PageSize int `json:"pageSize"`
}

// 照片列表
func HandlePhotoList(c *gin.Context) {
	var req PhotoListRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helpers.AppLogger.Errorf("请求参数绑定失败: %v", err)
		c.JSON(400, APIResponse[any]{Code: BadRequest, Message: "请求参数错误", Data: nil})
		return
	}
	var photos, err = models.ListPhotos(req.Page, req.PageSize)
	if err != nil {
		helpers.AppLogger.Errorf("查询照片列表失败: %v", err)
		c.JSON(500, APIResponse[any]{Code: BadRequest, Message: "查询照片列表失败", Data: nil})
		return
	}
	c.JSON(200, APIResponse[[]*models.Photo]{Code: Success, Message: "", Data: photos})
}

type PhotoUpdateRequest struct {
	Path    string `json:"path"`
	FileUri string `json:"fileUri"`
	Mtime   int64  `json:"mtime"`
	Ctime   int64  `json:"ctime"`
}

func HandlePhotoUpdate(c *gin.Context) {
	var req PhotoUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helpers.AppLogger.Errorf("请求参数绑定失败: %v", err)
		c.JSON(400, APIResponse[any]{Code: BadRequest, Message: "请求参数错误", Data: nil})
		return
	}
	photo, err := models.GetPhotoByPath(req.Path)
	if err != nil {
		helpers.AppLogger.Errorf("查询照片失败: %v", err)
		c.JSON(500, APIResponse[any]{Code: BadRequest, Message: "查询照片失败", Data: nil})
		return
	}
	photo.FileURI = req.FileUri
	photo.MTime = req.Mtime
	photo.CTime = req.Ctime
	if err := photo.Update(); err != nil {
		helpers.AppLogger.Errorf("更新照片失败: %v", err)
		c.JSON(500, APIResponse[any]{Code: BadRequest, Message: "更新照片失败", Data: nil})
		return
	}
	c.JSON(200, APIResponse[any]{Code: Success, Message: "更新成功", Data: nil})
}
