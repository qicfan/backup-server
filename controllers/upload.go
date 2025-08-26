package controllers

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/qicfan/backup-server/helpers"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

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
		relPath := filepath.Base(chunk.FileName)
		targetPath := filepath.Join(UPLOAD_ROOT_DIR, relPath)
		tempFileName := targetPath + ".uploading"
		lock, _ := fileLocks.LoadOrStore(tempFileName, &sync.Mutex{})
		mu := lock.(*sync.Mutex)
		mu.Lock()
		if err := os.MkdirAll(UPLOAD_ROOT_DIR, 0755); err != nil {
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
			if err := os.Rename(tempFileName, targetPath); err != nil {
				helpers.AppLogger.Error("Rename error:", err)
			} else {
				helpers.AppLogger.Infof("File %s upload complete.", targetPath)
			}
			fileLocks.Delete(tempFileName)
		}
	}
}

func CleanupUploadingFiles() {
	entries, err := os.ReadDir(".")
	if err != nil {
		helpers.AppLogger.Error("ReadDir error:", err)
		return
	}
	for _, entry := range entries {
		if entry.Type().IsRegular() && len(entry.Name()) > 9 && entry.Name()[len(entry.Name())-9:] == ".uploading" {
			err := os.Remove(entry.Name())
			if err != nil {
				helpers.AppLogger.Errorf("Failed to remove temp file %s: %v", entry.Name(), err)
			} else {
				helpers.AppLogger.Infof("Removed temp file: %s", entry.Name())
			}
		}
	}
}
