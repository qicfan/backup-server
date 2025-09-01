package controllers

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/qicfan/backup-server/helpers"
	"github.com/qicfan/backup-server/models"
)

type PathRequest struct {
	Path string `json:"path" form:"path"`
}

type PathExistsRequest struct {
	Path     string `json:"path" form:"path"`
	PathType string `json:"pathType" form:"pathType"`
}

type ExistsResponse struct {
	Exists bool `json:"exists"`
}

type DirOrFileEntry struct {
	Name    string `json:"name"`
	RelPath string `json:"relPath"` // 相对路径，不以 / 开头，相对helpers.UPLOAD_ROOT_DIR的路径
	IsDir   bool   `json:"isDir"`   // 是否文件夹
}

type CreateDirRequest struct {
	Parent string `json:"parent" form:"parent" binding:"required"`
	Name   string `json:"name" form:"name" binding:"required"`
}

type CreateDirResponse struct {
	Success bool   `json:"success"`
	Path    string `json:"path"`
	Error   string `json:"error,omitempty"`
}

// 创建目录接口
// parent: 目录的父路径
// name: 目录名称
// return: data.path 新创建的目录的相对路径
func HandleCreateDir(c *gin.Context) {
	var req CreateDirRequest
	if err := c.ShouldBind(&req); err != nil {
		helpers.AppLogger.Warnf("请求的参数错误: %v", err)
		c.JSON(http.StatusBadRequest, APIResponse[any]{Code: BadRequest, Message: "请求的参数错误: " + err.Error(), Data: nil})
		return
	}
	parent := req.Parent
	name := req.Name
	relPath := filepath.Join(parent, name)
	if after, ok := strings.CutPrefix(relPath, "/"); ok {
		relPath = after
	}
	absPath := filepath.Join(helpers.UPLOAD_ROOT_DIR, relPath)
	if err := os.MkdirAll(absPath, 0755); err != nil {
		helpers.AppLogger.Errorf("Create dir error: %v", err)
		c.JSON(http.StatusInternalServerError, APIResponse[any]{Code: BadRequest, Message: err.Error(), Data: map[string]interface{}{"path": relPath}})
		return
	}
	helpers.AppLogger.Infof("Created dir: %s", absPath)
	c.JSON(http.StatusOK, APIResponse[map[string]string]{Code: Success, Message: "", Data: map[string]string{"path": relPath}})
}

// 检查目录或文件是否存在
// path: 目录或文件的路径
// return: data.exists 是否存在，bool型
func HandleExists(c *gin.Context) {
	var req PathExistsRequest
	if err := c.ShouldBind(&req); err != nil {
		helpers.AppLogger.Warnf("Invalid request: %v", err)
		c.JSON(http.StatusBadRequest, APIResponse[any]{Code: BadRequest, Message: "参数错误: " + err.Error(), Data: nil})
		return
	}
	var exists bool
	if req.PathType == "1" {
		fullPath := filepath.Join(helpers.UPLOAD_ROOT_DIR, req.Path)
		exists = helpers.FileExists(fullPath)
		helpers.AppLogger.Infof("Check exists: %s : %s : %v", req.Path, fullPath, exists)
	} else {
		// req.Path是Photos.fileUri
		_, photoErr := models.GetPhotoByFileUri(req.Path)
		exists = photoErr == nil
		helpers.AppLogger.Infof("Check exists: %s : %v", req.Path, exists)
	}
	if !exists {
		// 如果文件不存在则判断PreChecksum是否一致，如果PreChecksum有重复项责判断checksum是否一致，如果checksum也有重复项责判定为重复
		preChecksum := c.Request.Header.Get("X-Pre-Checksum")
		if preChecksum != "" {
			if exists, _ := models.CheckPhotoPreChecksum(preChecksum); exists {
				helpers.AppLogger.Infof("PreChecksum exists: %s", preChecksum)
				exists = true
			} else {
				helpers.AppLogger.Infof("PreChecksum not exists: %s", preChecksum)
				exists = false
			}
		}
	}

	c.JSON(http.StatusOK, APIResponse[map[string]bool]{Code: Success, Message: "", Data: map[string]bool{"exists": exists}})
}

// 检查checksum是否存在
func HandleChecksumExists(c *gin.Context) {
	checksum := c.Request.Header.Get("X-Checksum")
	if checksum == "" {
		helpers.AppLogger.Warnf("Invalid request: missing X-Checksum header")
		c.JSON(http.StatusBadRequest, APIResponse[any]{Code: BadRequest, Message: "缺少X-Checksum头部", Data: nil})
		return
	}
	exists, err := models.CheckPhotoChecksum(checksum)
	if err != nil {
		helpers.AppLogger.Errorf("Check checksum exists error: %v", err)
		c.JSON(http.StatusInternalServerError, APIResponse[any]{Code: http.StatusInternalServerError, Message: err.Error(), Data: nil})
		return
	}
	c.JSON(http.StatusOK, APIResponse[map[string]bool]{Code: Success, Message: "", Data: map[string]bool{"exists": exists}})
}

// 目录列表
// path: 目录的路径
// return: data.entries 目录下的子目录列表
func HandleListDir(c *gin.Context) {
	var req PathRequest
	if err := c.ShouldBind(&req); err != nil {
		helpers.AppLogger.Warnf("Invalid request: %v", err)
		c.JSON(http.StatusBadRequest, APIResponse[interface{}]{Code: BadRequest, Message: "参数错误: " + err.Error(), Data: nil})
		return
	}
	path := req.Path
	if strings.HasPrefix(path, "/") {
		path = strings.TrimLeft(path, "/")
	}
	// 构造绝路路径
	absPath := filepath.Join(helpers.UPLOAD_ROOT_DIR, path)
	helpers.AppLogger.Infof("Listing dir: %s => %s", path, absPath)
	entries, err := os.ReadDir(absPath)
	if err != nil {
		helpers.AppLogger.Errorf("ReadDir error: %v", err)
		c.JSON(http.StatusInternalServerError, APIResponse[interface{}]{Code: BadRequest, Message: err.Error(), Data: nil})
		return
	}
	dirs := make([]DirOrFileEntry, 0)
	for _, entry := range entries {
		fileName := strings.TrimLeft(entry.Name(), "/")
		relPath := filepath.Join(path, fileName)
		isDir := entry.IsDir()
		// 过滤掉非目录，.开头的隐藏文件，.和..
		if !isDir || strings.HasPrefix(relPath, ".") {
			continue
		}
		dirs = append(dirs, DirOrFileEntry{Name: entry.Name(), RelPath: relPath, IsDir: isDir})
	}
	c.JSON(http.StatusOK, APIResponse[[]DirOrFileEntry]{Code: Success, Message: "", Data: dirs})
}
