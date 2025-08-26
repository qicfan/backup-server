package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
)

const UPLOAD_ROOT_DIR = "/upload"

var Version string = "v0.0.1"
var PublishDate string = "2025-08-26"

// 路径请求结构体
type PathRequest struct {
	Path string `json:"path"`
}

// 路径存在响应结构体
type ExistsResponse struct {
	Exists bool `json:"exists"`
}

// 目录列表响应结构体
type ListDirResponse struct {
	Entries []string `json:"entries"`
}

// 判断路径是否存在
func handleExists(c *gin.Context) {
	var req PathRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request"})
		return
	}
	_, err := os.Stat(req.Path)
	c.JSON(200, ExistsResponse{Exists: err == nil})
}

// 返回目录下的子项
func handleListDir(c *gin.Context) {
	var req PathRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request"})
		return
	}
	entries, err := os.ReadDir(req.Path)
	if err != nil {
		c.JSON(400, gin.H{"error": "ReadDir error"})
		return
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	c.JSON(200, ListDirResponse{Entries: names})
}

// 登录请求结构体
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// 登录响应结构体
type LoginResponse struct {
	Token string `json:"token"`
}

// 登录处理
func handleLogin(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request"})
		return
	}
	envUser := os.Getenv("USERNAME")
	envPass := os.Getenv("PASSWORD")
	if req.Username != envUser || req.Password != envPass {
		c.JSON(401, gin.H{"error": "Unauthorized"})
		return
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"username": req.Username,
		"exp":      jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
	})
	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		c.JSON(500, gin.H{"error": "Token generation error"})
		return
	}
	c.JSON(200, LoginResponse{Token: tokenString})
}

// JWT 密钥（请替换为你的密钥）
var jwtSecret = []byte("your-secret-key")

// 校验 JWT Token
func validateJWT(tokenString string) error {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})
	if err != nil || !token.Valid {
		return fmt.Errorf("invalid token")
	}
	return nil
}

// 文件块结构体
type FileChunk struct {
	FileName   string `json:"fileName"`
	ChunkIndex int    `json:"chunkIndex"`
	Data       []byte `json:"data"`
	IsLast     bool   `json:"isLast"`
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// 用于管理文件写入的锁
var fileLocks sync.Map

func handleUpload(c *gin.Context) {
	// 从 header 获取 JWT（可根据实际情况调整 header 名）
	tokenString := c.GetHeader("Sec-WebSocket-Protocol")
	if tokenString == "" {
		c.String(401, "Missing JWT token")
		return
	}
	if err := validateJWT(tokenString); err != nil {
		c.String(401, "Invalid JWT token")
		return
	}
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		fmt.Println("WebSocket upgrade error:", err)
		return
	}
	defer conn.Close()

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			fmt.Println("Read error:", err)
			break
		}
		var chunk FileChunk
		if err := json.Unmarshal(msg, &chunk); err != nil {
			fmt.Println("Unmarshal error:", err)
			continue
		}

		// 文件锁，防止并发写入
		relPath := filepath.Base(chunk.FileName)
		targetPath := filepath.Join(UPLOAD_ROOT_DIR, relPath)
		tempFileName := targetPath + ".uploading"
		lock, _ := fileLocks.LoadOrStore(tempFileName, &sync.Mutex{})
		mu := lock.(*sync.Mutex)
		mu.Lock()
		// 创建根目录
		if err := os.MkdirAll(UPLOAD_ROOT_DIR, 0755); err != nil {
			fmt.Println("MkdirAll error:", err)
			mu.Unlock()
			continue
		}
		f, err := os.OpenFile(tempFileName, os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			fmt.Println("File open error:", err)
			mu.Unlock()
			continue
		}
		// 按块序号定位写入位置
		offset := int64(chunk.ChunkIndex) * int64(len(chunk.Data))
		if _, err := f.Seek(offset, io.SeekStart); err != nil {
			fmt.Println("Seek error:", err)
			f.Close()
			mu.Unlock()
			continue
		}
		if _, err := f.Write(chunk.Data); err != nil {
			fmt.Println("Write error:", err)
		}
		f.Close()
		mu.Unlock()

		// 最后一块，重命名临时文件为目标文件，清理锁
		if chunk.IsLast {
			if err := os.Rename(tempFileName, targetPath); err != nil {
				fmt.Println("Rename error:", err)
			} else {
				fmt.Printf("File %s upload complete.\n", targetPath)
			}
			fileLocks.Delete(tempFileName)
		}
	}
}

func cleanupUploadingFiles() {
	entries, err := os.ReadDir(".")
	if err != nil {
		fmt.Println("ReadDir error:", err)
		return
	}
	for _, entry := range entries {
		if entry.Type().IsRegular() && len(entry.Name()) > 9 && entry.Name()[len(entry.Name())-9:] == ".uploading" {
			err := os.Remove(entry.Name())
			if err != nil {
				fmt.Printf("Failed to remove temp file %s: %v\n", entry.Name(), err)
			} else {
				fmt.Printf("Removed temp file: %s\n", entry.Name())
			}
		}
	}
}

func main() {
	cleanupUploadingFiles()
	// 启动时判断并创建上传根目录
	if _, err := os.Stat(UPLOAD_ROOT_DIR); os.IsNotExist(err) {
		if err := os.MkdirAll(UPLOAD_ROOT_DIR, 0755); err != nil {
			fmt.Printf("Failed to create upload root dir: %v\n", err)
		} else {
			fmt.Printf("Created upload root dir: %s\n", UPLOAD_ROOT_DIR)
		}
	}
	r := gin.Default()
	r.POST("/login", handleLogin)
	r.POST("/exists", handleExists)
	r.POST("/listdir", handleListDir)
	r.GET("/upload", handleUpload)
	fmt.Println("WebSocket server started at :8080 (SSL supported)")
	certFile := "config/server.crt"
	keyFile := "config/server.key"
	if _, err := os.Stat(certFile); err == nil {
		if _, err := os.Stat(keyFile); err == nil {
			err := r.RunTLS(":8080", certFile, keyFile)
			if err != nil {
				fmt.Println("ListenAndServeTLS error:", err)
			}
			return
		}
	}
	// 没有证书则回退到普通 HTTP
	err := r.Run(":8080")
	if err != nil {
		fmt.Println("ListenAndServe error:", err)
	}
}
