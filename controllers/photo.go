package controllers

import (
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/qicfan/backup-server/helpers"
	"github.com/qicfan/backup-server/models"
)

type DownloadQuery struct {
	Path          string `json:"path" form:"path"`                       // 相对路径
	Cos           string `json:"cos" form:"cos"`                         // 客户端操作系统
	Live          int    `json:"live" form:"live"`                       // 是否为动态照片
	Transcode     int    `json:"transcode" form:"transcode"`             // 是否转码，0-不转码，1-转码，默认0
	TransImageExt string `json:"trans_image_ext" form:"trans_image_ext"` // 转码后图片的扩展名
	TransVideoExt string `json:"trans_video_ext" form:"trans_video_ext"` // 转码后视频的扩展名
}

// 查询图片的的缩略图，构造一个请求
// http://yourserver/photo/thumbnail/MovieBackup%2FHuawei%20Pura%20X%2F2025%2F8%2F27%2F1.jpg/100x100
func HandleGetThumbnail(c *gin.Context) {
	path := c.Param("path") // 相对路径，不以 / 开头，相对helpers.UPLOAD_ROOT_DIR的路径，需要做base64_decode
	urldecodePath, _ := url.QueryUnescape(path)
	decodedPath, err := helpers.Base64Decode(urldecodePath)
	if err != nil {
		helpers.AppLogger.Errorf("路径解码失败: %v", err)
		c.JSON(http.StatusBadRequest, APIResponse[any]{Code: BadRequest, Message: "路径解码失败", Data: nil})
		return
	}
	path = decodedPath
	fullPath := filepath.Join(helpers.UPLOAD_ROOT_DIR, path)
	size := c.Param("size") // 尺寸 100x100格式
	helpers.AppLogger.Infof("获取缩略图: %s, 尺寸: %s", path, size)
	// 检查path是否存在
	if !helpers.FileExists(fullPath) {
		helpers.AppLogger.Errorf("照片 %s 不存在", fullPath)
		c.JSON(http.StatusNotFound, APIResponse[any]{Code: BadRequest, Message: "照片未找到", Data: nil})
		return
	}
	// 解析尺寸参数
	var width, height int
	_, err = fmt.Sscanf(size, "%dx%d", &width, &height)
	if err != nil || width <= 0 || height <= 0 {
		helpers.AppLogger.Errorf("尺寸参数错误: %v", err)
		c.JSON(http.StatusBadRequest, APIResponse[any]{Code: BadRequest, Message: "尺寸参数错误", Data: nil})
		return
	}
	var thumbnailPath string = ""
	if helpers.IsVideo(fullPath) {
		var videoErr error
		thumbnailPath, videoErr = helpers.ExtractVideoThumbnail(path, size)
		if videoErr != nil {
			// helpers.AppLogger.Errorf("生成视频缩略图失败: %v", videoErr)
			c.JSON(http.StatusInternalServerError, APIResponse[any]{Code: BadRequest, Message: "生成视频缩略图失败: " + videoErr.Error(), Data: nil})
			return
		}
	}
	if helpers.IsImage(fullPath) {
		var thumbNailErr error
		thumbnailPath, thumbNailErr = helpers.Thumbnail(path, size)
		if thumbNailErr != nil {
			// helpers.AppLogger.Errorf("生成缩略图失败: %v", thumbNailErr)
			c.JSON(http.StatusInternalServerError, APIResponse[any]{Code: BadRequest, Message: "生成缩略图失败", Data: nil})
			return
		}
	}
	if thumbnailPath == "" {
		c.JSON(http.StatusInternalServerError, APIResponse[any]{Code: BadRequest, Message: "生成缩略图失败", Data: nil})
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
	path = strings.TrimPrefix(path, string(os.PathSeparator))
	clientOS := helpers.ClientOS(queryParams.Cos)
	if clientOS == helpers.UNKNOW {
		clientOS = helpers.HMOS // 默认HMOS
	}
	isLive := queryParams.Live == 1
	fullPath := filepath.Join(helpers.UPLOAD_ROOT_DIR, path)
	helpers.AppLogger.Infof("下载文件: %s", fullPath)
	// 检查文件是否存在
	if !helpers.FileExists(fullPath) {
		c.JSON(http.StatusNotFound, APIResponse[any]{Code: BadRequest, Message: "文件未找到", Data: nil})
		return
	}
	// 查找photo
	photo, err := models.GetPhotoByPath(path)
	if err != nil {
		helpers.AppLogger.Errorf("查找照片失败: %v", err)
		c.JSON(http.StatusInternalServerError, APIResponse[any]{Code: BadRequest, Message: "查找照片失败", Data: nil})
		return
	}
	var destPath = path
	var destFullPath = fullPath
	var livePhotoVideoPath = photo.LivePhotoVideoPath
	var size int64 = 0
	// var preChecksum string
	var checksum string
	if helpers.IsImage(fullPath) && queryParams.Transcode == 1 {
		if isLive {
			// 如果是动态照片的图片，则处理视频处理
			livePhotoVideoPath = fmt.Sprintf("%s%s", photo.LivePhotoVideoPath, queryParams.TransVideoExt)
		}
		var transErr error
		// 进行图片转码
		helpers.AppLogger.Infof("进行图片转码: %s", path)
		if destPath, destFullPath, transErr = helpers.TransImage(path, queryParams.TransImageExt); transErr != nil {
			helpers.AppLogger.Errorf("图片转码失败: %v", transErr)
			c.JSON(http.StatusInternalServerError, APIResponse[any]{Code: BadRequest, Message: "图片转码失败", Data: nil})
			return
		} else {
			helpers.AppLogger.Infof("图片转码成功: %s -> %s", path, destPath)
		}
	}
	if helpers.IsVideo(fullPath) && queryParams.Transcode == 1 {
		// 进行视频转码
		var transErr error
		helpers.AppLogger.Infof("进行视频转码: %s", path)
		if destPath, destFullPath, transErr = helpers.TransVideo(path, queryParams.TransVideoExt); transErr != nil {
			helpers.AppLogger.Errorf("视频转码失败: %v", transErr)
			c.JSON(http.StatusInternalServerError, APIResponse[any]{Code: BadRequest, Message: "视频转码失败", Data: nil})
			return
		} else {
			helpers.AppLogger.Infof("视频转码成功: %s -> %s", path, destPath)
		}
	}
	if queryParams.Transcode == 1 {
		if fileInfo, err := os.Stat(destFullPath); err == nil {
			size = fileInfo.Size()
		}
		// 修改文件的创建时间和修改时间为photo的MTime和CTime（秒转time.Time）
		mtime := time.Unix(photo.MTime, 0)
		ctime := time.Unix(photo.CTime, 0)
		os.Chtimes(destFullPath, mtime, ctime)
		// preChecksum, _ = helpers.FileHeadSHA1(destFullPath)
		checksum, _ = helpers.FileSHA1(destFullPath)
		// 写入数据库
		if err := models.InsertPhoto(photo.Name, destPath, size, photo.Type, livePhotoVideoPath, "", photo.MTime, photo.CTime, checksum, photo.ID); err != nil {
			helpers.AppLogger.Errorf("将转码的Photo插入数据库失败: %v", err)
			c.JSON(http.StatusInternalServerError, APIResponse[any]{Code: BadRequest, Message: "更新照片路径失败", Data: nil})
			return
		}
	}
	// 流式传输
	c.FileAttachment(fullPath, filepath.Base(destFullPath))
}

type PhotoListRequest struct {
	Page     int `json:"page" form:"page"`
	PageSize int `json:"page_size" form:"page_size"`
}

// 照片列表
func HandlePhotoList(c *gin.Context) {
	var req PhotoListRequest
	if err := c.ShouldBind(&req); err != nil {
		helpers.AppLogger.Errorf("请求参数绑定失败: %v", err)
		c.JSON(http.StatusBadRequest, APIResponse[any]{Code: BadRequest, Message: "请求参数错误", Data: nil})
		return
	}
	helpers.AppLogger.Infof("查询照片列表: 页码 %d, 每页 %d", req.Page, req.PageSize)
	var total, photos, err = models.ListPhotos(req.Page, req.PageSize)
	if err != nil {
		helpers.AppLogger.Errorf("查询照片列表失败: %v", err)
		c.JSON(http.StatusInternalServerError, APIResponse[any]{Code: BadRequest, Message: "查询照片列表失败", Data: nil})
		return
	}
	helpers.AppLogger.Infof("查询照片列表成功: 总%d张， 本次返回 %d 张", total, len(photos))
	c.JSON(http.StatusOK, APIResponse[map[string]any]{Code: Success, Message: "", Data: map[string]any{"total": total, "photos": photos}})
}

type PhotoUpdateRequest struct {
	Path    string `json:"path" form:"path" binding:"required"`
	FileUri string `json:"fileUri" form:"fileUri" binding:"required"`
}

// 更新照片的fileUri
func HandlePhotoUpdate(c *gin.Context) {
	var req PhotoUpdateRequest
	if err := c.ShouldBind(&req); err != nil {
		helpers.AppLogger.Errorf("请求参数绑定失败: %v", err)
		c.JSON(http.StatusBadRequest, APIResponse[any]{Code: BadRequest, Message: "请求参数错误: " + err.Error(), Data: nil})
		return
	}
	photo, err := models.GetPhotoByPath(req.Path)
	if err != nil {
		helpers.AppLogger.Errorf("查询照片失败: %v", err)
		c.JSON(http.StatusInternalServerError, APIResponse[any]{Code: BadRequest, Message: "查询照片失败: " + err.Error(), Data: nil})
		return
	}
	photo.FileURI = req.FileUri
	if err := photo.Update(); err != nil {
		helpers.AppLogger.Errorf("更新照片失败: %v", err)
		c.JSON(http.StatusInternalServerError, APIResponse[any]{Code: BadRequest, Message: "更新照片失败: " + err.Error(), Data: nil})
		return
	}
	// 将sourceID=photo.ID的记录的fileUri也更新
	models.UpdatePhotoFileUri(photo.ID, photo.FileURI)
	// if photo.SourceId != 0 {
	// 	// 根据sourceId查询photo
	// 	sourcePhoto, err := models.GetPhotoById(photo.SourceId)
	// 	if err != nil {
	// 		helpers.AppLogger.Errorf("根据sourceId查询照片失败: %v", err)
	// 		c.JSON(http.StatusInternalServerError, APIResponse[any]{Code: BadRequest, Message: "根据sourceId查询照片失败: " + err.Error(), Data: nil})
	// 		return
	// 	}
	// 	sourcePhoto.FileURI = photo.FileURI
	// 	if err := sourcePhoto.Update(); err != nil {
	// 		helpers.AppLogger.Errorf("更新源照片失败: %v", err)
	// 		c.JSON(http.StatusInternalServerError, APIResponse[any]{Code: BadRequest, Message: "更新源照片失败: " + err.Error(), Data: nil})
	// 		return
	// 	}
	// }
	c.JSON(http.StatusOK, APIResponse[any]{Code: Success, Message: "更新成功", Data: req.Path})
}
