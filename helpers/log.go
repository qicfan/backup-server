package helpers

import (
	"os"

	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"github.com/sirupsen/logrus"
)

var AppLogger = newLogger()

func newLogger() *logrus.Logger {
	logger := logrus.New()
	logDir := "config/logs"
	logFile := logDir + "/app.log"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		logger.Warnf("Failed to create log dir: %v", err)
	}
	writer, err := rotatelogs.New(
		logFile+".%Y%m%d%H%M",
		rotatelogs.WithLinkName(logFile),
		rotatelogs.WithRotationSize(1*1024*1024), // 1MB
		rotatelogs.WithRotationCount(5),
	)
	if err != nil {
		logger.Warnf("Failed to create rotatelogs: %v", err)
	} else {
		logger.SetOutput(writer)
	}
	return logger
}
