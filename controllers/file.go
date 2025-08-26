package controllers

import (
	"os"

	"github.com/qicfan/backup-server/helpers"

	"github.com/gin-gonic/gin"
)

type FileChunk struct {
	FileName   string `json:"fileName"`
	ChunkIndex int    `json:"chunkIndex"`
	Data       []byte `json:"data"`
	IsLast     bool   `json:"isLast"`
}

type PathRequest struct {
	Path string `json:"path"`
}
type ExistsResponse struct {
	Exists bool `json:"exists"`
}
type ListDirResponse struct {
	Entries []string `json:"entries"`
}

func HandleExists(c *gin.Context) {
	var req PathRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helpers.AppLogger.Warnf("Invalid request: %v", err)
		c.JSON(400, gin.H{"error": "Invalid request"})
		return
	}
	_, err := os.Stat(req.Path)
	c.JSON(200, ExistsResponse{Exists: err == nil})
}

func HandleListDir(c *gin.Context) {
	var req PathRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helpers.AppLogger.Warnf("Invalid request: %v", err)
		c.JSON(400, gin.H{"error": "Invalid request"})
		return
	}
	entries, err := os.ReadDir(req.Path)
	if err != nil {
		helpers.AppLogger.Errorf("ReadDir error: %v", err)
		c.JSON(400, gin.H{"error": "ReadDir error"})
		return
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	c.JSON(200, ListDirResponse{Entries: names})
}
