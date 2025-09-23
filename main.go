package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/qicfan/backup-server/controllers"
	"github.com/qicfan/backup-server/helpers"
	"github.com/qicfan/backup-server/models"

	"github.com/gin-gonic/gin"
	ginlogrus "github.com/toorop/gin-logrus"
)

var Version string = "v0.0.1"
var PublishDate string = "2025-08-26"
var IsRelease bool = false

func main() {
	cstZone := time.FixedZone("CST", 8*3600)
	time.Local = cstZone
	getRootDir()
	logger := helpers.NewLogger("web.log")
	helpers.AppLogger = helpers.NewLogger("app.log")
	initUploadDir()
	helpers.AppLogger.Infof("Backup Server %s (%s) starting...\n", Version, PublishDate)
	helpers.AppLogger.Infof("运行目录: %s\n", helpers.RootDir)
	helpers.AppLogger.Infof("上传目录: %s\n", helpers.UPLOAD_ROOT_DIR)
	helpers.InitDb()                // 初始化数据库组件
	models.Migrate()                // 执行数据库迁移
	helpers.CleanupUploadingFiles() // 清理所有未完成的上传临时文件
	models.RefreshPhotoCollection() // 先执行一遍
	models.InitCron()               // 初始化定时任务
	if IsRelease {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()
	r.Use(ginlogrus.Logger(logger), gin.Recovery())
	r.POST("/login", controllers.HandleLogin)
	api := r.Group("/api")
	api.Use(controllers.JWTAuthMiddleware())
	{
		api.POST("/exists", controllers.HandleExists)
		api.POST("/exists-checksum", controllers.HandleChecksumExists)
		api.POST("/listdir", controllers.HandleListDir)
		api.POST("/createdir", controllers.HandleCreateDir)
	}
	photoApi := r.Group("/photo")
	photoApi.Use(controllers.JWTAuthMiddleware())
	{
		photoApi.GET("/thumbnail/:path/:size", controllers.HandleGetThumbnail) // 缩略图查看
		photoApi.GET("/download", controllers.HandlePhotoDownload)             // 文件下载
		photoApi.GET("/list", controllers.HandlePhotoList)                     // 照片列表
		photoApi.POST("/update", controllers.HandlePhotoUpdate)                // 照片信息更新
	}
	r.GET("/upload", controllers.HandleUpload)
	// r.GET("/upload/status", controllers.HandleUploadStatus)
	port := os.Getenv("PORT")
	if port == "" {
		port = "12334"
	}
	addr := ":" + port

	certFile := filepath.Join(helpers.RootDir, "config", "server.crt")
	keyFile := filepath.Join(helpers.RootDir, "config", "server.key")
	if helpers.FileExists(certFile) && helpers.FileExists(keyFile) {
		fmt.Printf("WebSocket server started at %s (SSL supported)\n", addr)
		err := r.RunTLS(addr, certFile, keyFile)
		if err != nil {
			fmt.Println("ListenAndServeTLS error:", err)
		}
		return
	}
	fmt.Printf("WebSocket server started at %s (SSL not supported)\n", addr)
	// 没有证书则回退到普通 HTTP
	weberr := r.Run(addr)
	if weberr != nil {
		fmt.Println("ListenAndServe error:", weberr)
	}
}

func initUploadDir() {
	// 启动时判断并创建上传根目录
	if !helpers.FileExists(helpers.UPLOAD_ROOT_DIR) {
		if err := os.MkdirAll(helpers.UPLOAD_ROOT_DIR, 0755); err != nil {
			helpers.AppLogger.Errorf("Failed to create upload root dir: %v\n", err)
		} else {
			helpers.AppLogger.Infof("Created upload root dir: %s\n", helpers.UPLOAD_ROOT_DIR)
		}
	}
}

func checkRelease() {
	arg1 := strings.ToLower(os.Args[0])
	fmt.Printf("arg1=%s\n", arg1)
	name := strings.ToLower(filepath.Base(arg1))
	IsRelease = strings.Index(name, "backup-server") == 0 && !strings.Contains(arg1, "go-build")
}

func getRootDir() string {
	var exPath string = "/app" // 默认使用docker的路径
	checkRelease()
	fmt.Printf("isRelease=%v\n", IsRelease)
	if IsRelease {
		ex, err := os.Executable()
		if err != nil {
			panic(err)
		}
		exPath = filepath.Dir(ex)
	} else {
		if runtime.GOOS == "windows" {
			exPath = "D:\\Dev\\backup-server"
		} else {
			exPath = "/home/samba/shares/dev/backup-server"
		}
	}
	helpers.RootDir = exPath // 获取当前工作目录
	if !IsRelease {
		if runtime.GOOS == "windows" {
			helpers.UPLOAD_ROOT_DIR = "D:\\Dev\\backup-server\\config\\upload"
		} else {
			helpers.UPLOAD_ROOT_DIR = "/home/samba/shares/dev/backup-server/config/upload"
		}
	} else {
		envDir := os.Getenv("UPLOAD_ROOT_DIR")
		if envDir != "" {
			helpers.UPLOAD_ROOT_DIR = envDir
		}
	}
	return exPath
}
