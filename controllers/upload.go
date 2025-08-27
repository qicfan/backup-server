package controllers

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/qicfan/backup-server/helpers"
	"github.com/qicfan/backup-server/models"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type FileChunk struct {
	FileName           string           `json:"fileName"`              // 相对路径，包含文件名：2025/8/25/a.jpg
	Type               models.PhotoType `json:"type"`                  // 照片类型，1-普通照片，2-视频， 3-动态照片
	LivePhotoVideoPath string           `json:"live_photo_video_path"` // 如果是动态照片，这里存储视频的路径，只有动态照片中的图片会保存该字段，如果是动态照片的视频则该字段为空
	MTime              int64            `json:"mtime"`                 // 照片的最后修改时间，Unix时间戳，单位秒
	CTime              int64            `json:"ctime"`                 // 照片的创建时间，Unix时间戳，单位秒
	FileURI            string           `json:"fileUri"`               // 鸿蒙系统的照片资源的URI，可以用来查询照片是否存在，如果有这个字段代表本地存在该照片(动态视频的视频没有该字段)
	Size               int64            `json:"size"`                  // 照片的大小，单位字节
	ChunkIndex         int              `json:"chunkIndex"`
	Data               []byte           `json:"data"`
	IsLast             bool             `json:"isLast"`
}

var fileLocks sync.Map
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func HandleUpload(c *gin.Context) {
	tokenString := c.GetHeader("Sec-WebSocket-Protocol")
	if tokenString == "" {
		c.String(401, "Missing JWT token")
		return
	}
	if err := ValidateJWT(tokenString); err != nil {
		c.String(401, "Invalid JWT token")
		return
	}
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		helpers.AppLogger.Error("WebSocket upgrade error:", err)
		return
	}
	defer conn.Close()

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			helpers.AppLogger.Error("Read error:", err)
			break
		}
		var chunk FileChunk
		if err := json.Unmarshal(msg, &chunk); err != nil {
			helpers.AppLogger.Error("Unmarshal error:", err)
			continue
		}
		relPath := filepath.Dir(chunk.FileName)
		fileName := filepath.Base(chunk.FileName)
		targetPath := filepath.Join(helpers.UPLOAD_ROOT_DIR, relPath)
		targetFile := filepath.Join(targetPath, fileName)
		tempFileName := targetFile + ".uploading"
		lock, _ := fileLocks.LoadOrStore(tempFileName, &sync.Mutex{})
		mu := lock.(*sync.Mutex)
		mu.Lock()
		if err := os.MkdirAll(targetPath, 0755); err != nil {
			helpers.AppLogger.Error("MkdirAll error:", err)
			mu.Unlock()
			continue
		}
		f, err := os.OpenFile(tempFileName, os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			helpers.AppLogger.Error("File open error:", err)
			mu.Unlock()
			continue
		}
		offset := int64(chunk.ChunkIndex) * int64(len(chunk.Data))
		if _, err := f.Seek(offset, io.SeekStart); err != nil {
			helpers.AppLogger.Error("Seek error:", err)
			f.Close()
			mu.Unlock()
			continue
		}
		if _, err := f.Write(chunk.Data); err != nil {
			helpers.AppLogger.Error("Write error:", err)
		}
		f.Close()
		mu.Unlock()
		if chunk.IsLast {
			if err := os.Rename(tempFileName, targetFile); err != nil {
				helpers.AppLogger.Error("重命名上传临时文件是发生错误:", err)
			} else {
				helpers.AppLogger.Infof("文件 %s 上传完成.", targetFile)
				// 插入数据库
				photoType := chunk.Type
				livePhotoVideoPath := chunk.LivePhotoVideoPath
				if err := models.InsertPhoto(fileName, targetFile, chunk.Size, photoType, livePhotoVideoPath, chunk.FileURI, chunk.MTime, chunk.CTime); err != nil {
					helpers.AppLogger.Error("照片写入数据库错误:", err)
				}
				// 通知客户端上传完成
				resp := APIResponse[map[string]string]{Code: Success, Message: "上传完成", Data: map[string]string{"path": chunk.FileName}}
				msg, _ := json.Marshal(resp)
				_ = conn.WriteMessage(websocket.TextMessage, msg)
			}
			fileLocks.Delete(tempFileName)
		}
	}
}
