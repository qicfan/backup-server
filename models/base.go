package models

type BaseModel struct {
	ID        uint  `gorm:"primary" json:"id"`
	CreatedAt int64 `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt int64 `gorm:"autoUpdateTime" json:"updated_at"`
}
