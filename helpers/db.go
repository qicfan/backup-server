package helpers

import (
	"fmt"
	"path/filepath"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// 写入请求结构
type DBWriteRequest struct {
	ExecFunc func(db *gorm.DB) error
	ResultCh chan error
}

var dbWriteQueue chan DBWriteRequest
var Db *gorm.DB

// 启动写入调度 goroutine
func StartDBWriteWorker() {
	dbWriteQueue = make(chan DBWriteRequest, 100)
	go func() {
		for req := range dbWriteQueue {
			err := req.ExecFunc(Db)
			if req.ResultCh != nil {
				req.ResultCh <- err
			}
		}
	}()
}

// 提交写入请求（异步）
func EnqueueDBWrite(exec func(db *gorm.DB) error) {
	dbWriteQueue <- DBWriteRequest{ExecFunc: exec, ResultCh: nil}
}

// 提交写入请求（同步，等待结果）
func EnqueueDBWriteSync(exec func(db *gorm.DB) error) error {
	ch := make(chan error, 1)
	dbWriteQueue <- DBWriteRequest{ExecFunc: exec, ResultCh: ch}
	return <-ch
}

// 获取一个数据库连接
func GetDb() {
	dbFile := filepath.Join(RootDir, "config", "master.db")
	if Db != nil {
		return
	}
	var err error
	Db, err = gorm.Open(sqlite.Open(dbFile + "?cache=shared&_journal_mode=WAL"))
	if err != nil {
		panic(fmt.Sprintf("failed to connect database: %v", err))
	}
}

func InitDb() {
	// 建立数据库连接
	GetDb()
	StartDBWriteWorker()
	AppLogger.Info("成功初始化数据库组件和写入队列")
}
