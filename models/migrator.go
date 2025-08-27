package models

import (
	"github.com/qicfan/backup-server/helpers"
)

type Migrator struct {
	BaseModel
	VersionCode int `json:"version_code"` // 版本号
}

func (*Migrator) TableName() string {
	return "migrator"
}

// 数据库迁移
// 如果没有数据则创建
// 如果已有数据库则从数据库中获取版本，根据版本执行变更
func Migrate() {
	// 如果不存在则初始化所有表
	var migrator Migrator = Migrator{}
	if !helpers.Db.Migrator().HasTable(Migrator{}) {
		helpers.Db.AutoMigrate(Migrator{})
		migrator = Migrator{BaseModel: BaseModel{ID: 1}, VersionCode: 1} // 初始版本为1
		helpers.Db.Create(&migrator)
		helpers.AppLogger.Info("初始化数据库版本表")
	}
	err := helpers.Db.Model(&migrator).First(&migrator).Error
	if err != nil {
		helpers.AppLogger.Errorf("获取数据库版本表失败：%v", err)
	} else {
		helpers.AppLogger.Infof("数据库版本表存在，当前数据库版本：%d", migrator.VersionCode)
	}
	if migrator.VersionCode == 1 {
		helpers.Db.AutoMigrate(Photo{})

		migrator.updateVersion()
	}
}

func (m *Migrator) updateVersion() {
	m.VersionCode++
	helpers.Db.Save(m)
	helpers.AppLogger.Infof("数据库版本更新完毕，当前数据库版本：%d", m.VersionCode)
}
