package helpers

import (
	"io"
	"os"
	"path/filepath"

	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"github.com/sirupsen/logrus"
)

var AppLogger *logrus.Logger

func NewLogger(logFileName string) *logrus.Logger {
	logger := logrus.New()
	logDir := filepath.Join(RootDir, "config", "logs")
	logFile := filepath.Join(logDir, logFileName)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		logger.Warnf("Failed to create log dir: %v", err)
	}
	writer, err := rotatelogs.New(
		logFile+".%Y%m%d%H%M",
		rotatelogs.WithLinkName(logFile),
		rotatelogs.WithRotationSize(10*1024*1024), // 10MB
		rotatelogs.WithRotationCount(5),
	)
	if err != nil {
		logger.Warnf("Failed to create rotatelogs: %v", err)
	} else {
		// 同时写入文件和控制台
		logger.SetOutput(io.MultiWriter(writer, os.Stdout))
	}
	return logger
}
