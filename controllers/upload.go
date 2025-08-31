package controllers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

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
	ChunkIndex         int              `json:"chunkIndex"`            // 当前块索引
	ChunkCount         int              `json:"chunkCount"`            // 总块数
	ChunkHash          string           `json:"chunkHash"`             // 分片SHA256
	// Data字段废弃，分片数据通过WebSocket二进制帧单独发送
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024 * 1024 * 150, // 150MB
	WriteBufferSize: 1024 * 1024 * 10,  // 10MB
	CheckOrigin:     func(r *http.Request) bool { return true },
}

func HandleUpload(c *gin.Context) {
	tokenString := c.GetHeader("Sec-WebSocket-Protocol")
	helpers.AppLogger.Infof("HandleUpload called with token %s", tokenString)
	if tokenString == "" {
		helpers.AppLogger.Error("Missing JWT token")
		c.String(401, "Missing JWT token")
		return
	}
	if _, err := ValidateJWT(tokenString); err != nil {
		helpers.AppLogger.Error("Invalid JWT token:", err)
		c.String(401, "Invalid JWT token: %s", err.Error())
		return
	}
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		helpers.AppLogger.Error("WebSocket upgrade error:", err)
		return
	}
	defer conn.Close()
	// 设置300秒超时
	timeoutSec := 300
	// conn.WriteMessage(websocket.TextMessage, []byte("hello, welcome connect this ws"))
	for {
		// 设置读取超时时间
		_ = conn.SetReadDeadline(time.Now().Add(time.Duration(timeoutSec) * time.Second))
		// 先收JSON帧
		mt, msg, err := conn.ReadMessage()
		if err != nil {
			helpers.AppLogger.Error("Read error:", err)
			break
		}
		if mt != websocket.TextMessage {
			helpers.AppLogger.Error("预期JSON帧，收到非文本帧")
			continue
		}
		helpers.AppLogger.Debugf("Received message: %s", string(msg))
		var chunk FileChunk
		if err := json.Unmarshal(msg, &chunk); err != nil {
			helpers.AppLogger.Error("Unmarshal error:", err)
			continue
		}
		// 再收二进制分片数据
		mt, rawData, err := conn.ReadMessage()
		if err != nil {
			helpers.AppLogger.Error("分片数据读取失败:", err)
			continue
		}
		if mt != websocket.BinaryMessage {
			helpers.AppLogger.Error("预期分片二进制帧，收到非二进制帧")
			continue
		}
		relPath := filepath.Dir(chunk.FileName)
		fileName := filepath.Base(chunk.FileName)
		targetPath := filepath.Join(helpers.UPLOAD_ROOT_DIR, relPath)
		targetFile := filepath.Join(targetPath, fileName)
		if err := os.MkdirAll(targetPath, 0755); err != nil {
			helpers.AppLogger.Error("MkdirAll error:", err)
			continue
		}
		// 每个chunk保存独立临时文件，先校验hash
		chunkTempFile := fmt.Sprintf("%s.chunk%d", targetFile, chunk.ChunkIndex)
		actualHash := helpers.BytesSHA256(rawData)
		if actualHash != chunk.ChunkHash {
			helpers.AppLogger.Errorf("分片校验失败: index=%d, 期望=%s, 实际=%s", chunk.ChunkIndex, chunk.ChunkHash, actualHash)
			resp := APIResponse[any]{Code: BadRequest, Message: "分片校验失败", Data: map[string]any{"index": chunk.ChunkIndex}}
			msg, _ := json.Marshal(resp)
			_ = conn.WriteMessage(websocket.TextMessage, msg)
			continue
		}
		if err := os.WriteFile(chunkTempFile, rawData, 0644); err != nil {
			helpers.AppLogger.Error("Chunk写入失败:", err)
			continue
		}
		// 检查是否所有chunk都已上传
		complete := true
		for i := 0; i < chunk.ChunkCount; i++ {
			tempName := fmt.Sprintf("%s.chunk%d", targetFile, i)
			if _, err := os.Stat(tempName); err != nil {
				complete = false
				break
			}
		}
		if complete {
			helpers.AppLogger.Infof("所有分片上传完成，开始合并文件: %s", targetFile)
			// 合并所有chunk
			out, err := os.OpenFile(targetFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
			if err != nil {
				helpers.AppLogger.Error("合并文件失败:", err)
				continue
			}
			for i := 0; i < chunk.ChunkCount; i++ {
				tempName := fmt.Sprintf("%s.chunk%d", targetFile, i)
				part, err := os.Open(tempName)
				if err != nil {
					helpers.AppLogger.Error("读取chunk失败:", err)
					out.Close()
					continue
				}
				io.Copy(out, part)
				part.Close()
				os.Remove(tempName)
			}
			out.Close()
			helpers.AppLogger.Infof("文件 %s 上传完成.", targetFile)
			// 插入数据库
			photoType := chunk.Type
			livePhotoVideoPath := chunk.LivePhotoVideoPath
			if err := models.InsertPhoto(fileName, chunk.FileName, chunk.Size, photoType, livePhotoVideoPath, chunk.FileURI, chunk.MTime, chunk.CTime); err != nil {
				helpers.AppLogger.Error("照片写入数据库错误:", err)
			}
			// 通知客户端上传完成
			resp := APIResponse[map[string]string]{Code: Success, Message: "上传完成", Data: map[string]string{"path": chunk.FileName}}
			msg, _ := json.Marshal(resp)
			_ = conn.WriteMessage(websocket.TextMessage, msg)
		}
	}
}
