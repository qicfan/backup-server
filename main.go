package main

import (
	"fmt"
	"os"

	"github.com/qicfan/backup-server/controllers"

	"github.com/gin-gonic/gin"
	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"github.com/sirupsen/logrus"
	ginlogrus "github.com/toorop/gin-logrus"
)

func main() {
	controllers.CleanupUploadingFiles()
	// 启动时判断并创建上传根目录
	if _, err := os.Stat(controllers.UPLOAD_ROOT_DIR); os.IsNotExist(err) {
		if err := os.MkdirAll(controllers.UPLOAD_ROOT_DIR, 0755); err != nil {
			fmt.Printf("Failed to create upload root dir: %v\n", err)
		} else {
			fmt.Printf("Created upload root dir: %s\n", controllers.UPLOAD_ROOT_DIR)
		}
	}
	logger := logrus.New()
	logDir := "config/logs"
	logFile := logDir + "/web.log"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		fmt.Printf("Failed to create log dir: %v\n", err)
	}
	writer, err := rotatelogs.New(
		logFile+".%Y%m%d%H%M",
		rotatelogs.WithLinkName(logFile),
		rotatelogs.WithRotationSize(1*1024*1024), // 1MB
		rotatelogs.WithRotationCount(5),
	)
	if err != nil {
		fmt.Printf("Failed to create rotatelogs: %v\n", err)
	} else {
		logger.SetOutput(writer)
	}
	r := gin.New()
	r.Use(ginlogrus.Logger(logger), gin.Recovery())
	r.POST("/login", controllers.HandleLogin)
	r.POST("/exists", controllers.HandleExists)
	r.POST("/listdir", controllers.HandleListDir)
	r.GET("/upload", controllers.HandleUpload)
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
	weberr := r.Run(":8080")
	if weberr != nil {
		fmt.Println("ListenAndServe error:", weberr)
	}
}
