package controllers

import (
	"encoding/json"
	"fmt"
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
	Checksum           string           `json:"checksum"`              // 可选，照片的sha1校验和，如果有该字段则上传完成后会检查是否存在相同checksum的照片，如果存在则删除已上传的文件
}

var upgrader = websocket.Upgrader{
	ReadBufferSize: 500 * 1024, // 500KB
	CheckOrigin:    func(r *http.Request) bool { return true },
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
	var targetFileFd *os.File
	for {
		// 设置读取超时时间
		_ = conn.SetReadDeadline(time.Now().Add(time.Duration(timeoutSec) * time.Second))
		// 先收JSON帧
		mt, msg, err := conn.ReadMessage()
		if err != nil {
			helpers.AppLogger.Error("分片信息帧读取失败，中断连接:", err)
			resp := APIResponse[any]{Code: TerminalConnection, Message: fmt.Sprintf("分片信息帧读取失败: %s", err.Error()), Data: nil}
			msg, _ := json.Marshal(resp)
			_ = conn.WriteMessage(websocket.TextMessage, msg)
			break
		}
		if mt != websocket.TextMessage {
			helpers.AppLogger.Error("预期JSON帧，收到非文本帧")
			resp := APIResponse[any]{Code: TerminalConnection, Message: "预期JSON帧，收到非文本帧", Data: nil}
			msg, _ := json.Marshal(resp)
			_ = conn.WriteMessage(websocket.TextMessage, msg)
			continue
		}
		helpers.AppLogger.Debugf("Received message: %s", string(msg))
		var chunk FileChunk
		if err := json.Unmarshal(msg, &chunk); err != nil {
			helpers.AppLogger.Error("Unmarshal error:", err)
			continue
		}
		helpers.AppLogger.Infof("收到文件传输信息：%d/%d => %s", chunk.ChunkIndex+1, chunk.ChunkCount, chunk.FileName)
		// 再收二进制分片数据
		mt, rawData, err := conn.ReadMessage()
		if err != nil {
			helpers.AppLogger.Error("分片数据帧读取失败:", err)
			resp := APIResponse[any]{Code: TerminalConnection, Message: fmt.Sprintf("分片数据帧读取失败: %s", err.Error()), Data: nil}
			msg, _ := json.Marshal(resp)
			_ = conn.WriteMessage(websocket.TextMessage, msg)
			break
		}
		if mt != websocket.BinaryMessage {
			helpers.AppLogger.Error("预期分片二进制帧，收到非二进制帧")
			resp := APIResponse[any]{Code: TerminalConnection, Message: "预期分片二进制帧，收到非二进制帧", Data: nil}
			msg, _ := json.Marshal(resp)
			_ = conn.WriteMessage(websocket.TextMessage, msg)
			break
		}
		helpers.AppLogger.Debugf("Received binary data for chunk %d/%d => %s", chunk.ChunkIndex+1, chunk.ChunkCount, chunk.FileName)
		relPath := filepath.Dir(chunk.FileName)
		fileName := filepath.Base(chunk.FileName)
		targetPath := filepath.Join(helpers.UPLOAD_ROOT_DIR, relPath)
		targetFile := filepath.Join(targetPath, fileName)
		if err := os.MkdirAll(targetPath, 0755); err != nil {
			helpers.AppLogger.Error("MkdirAll error:", err)
			continue
		}
		if targetFileFd == nil {
			targetFileFd, err = os.OpenFile(targetFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
			if err != nil {
				helpers.AppLogger.Error("Open target file error:", err)
				resp := APIResponse[any]{Code: BadRequest, Message: fmt.Sprintf("打开目标文件失败: %s", err.Error()), Data: nil}
				msg, _ := json.Marshal(resp)
				_ = conn.WriteMessage(websocket.TextMessage, msg)
				continue
			}
		}

		if _, err := targetFileFd.Write(rawData); err != nil {
			helpers.AppLogger.Error("Chunk写入失败:", err)
			targetFileFd.Close()
			targetFileFd = nil
			continue
		}
		// 检查是否所有chunk都已上传
		complete := chunk.ChunkCount == chunk.ChunkIndex+1
		if complete {
			targetFileFd.Close()
			targetFileFd = nil
			helpers.AppLogger.Infof("文件 %s 上传完成.", targetFile)
			// 插入数据库
			photoType := chunk.Type
			livePhotoVideoPath := chunk.LivePhotoVideoPath
			// 计算sha1
			// checksum, _ := helpers.FileSHA1(targetFile)
			checksum := chunk.Checksum
			if checksum == "" {
				checksum, _ = helpers.FileSHA1(targetFile)
			}
			// helpers.AppLogger.Infof("照片哈希值: %s => %s", chunk.FileName, checksum)
			// 检查是否存在checksum相同的照片
			if exists, _ := models.CheckPhotoChecksum(checksum); exists {
				helpers.AppLogger.Infof("Checksum exists:%s => %s", chunk.FileName, checksum)
				// 删除已上传的文件
				os.Remove(targetFile)
			} else {
				helpers.AppLogger.Infof("Checksum not exists: %s => %s", chunk.FileName, checksum)
				if err := models.InsertPhoto(fileName, chunk.FileName, chunk.Size, photoType, livePhotoVideoPath, chunk.FileURI, chunk.MTime, chunk.CTime, checksum, 0); err != nil {
					helpers.AppLogger.Error("照片写入数据库错误:", err)
				}
				// 修改文件的ctime和mtime
				mtime := time.Unix(chunk.MTime, 0)
				ctime := time.Unix(chunk.CTime, 0)
				os.Chtimes(targetFile, mtime, ctime)
			}
			// 通知客户端上传完成
			resp := APIResponse[map[string]string]{Code: Success, Message: "上传完成", Data: map[string]string{"path": chunk.FileName}}
			msg, _ := json.Marshal(resp)
			_ = conn.WriteMessage(websocket.TextMessage, msg)
		}
	}
}
