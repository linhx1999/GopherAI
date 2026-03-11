package model

import (
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	UserID   string `gorm:"column:user_id;type:varchar(36);uniqueIndex;not null" json:"user_id"`
	Name     string `gorm:"type:varchar(50)" json:"name"`
	Email    string `gorm:"type:varchar(100);index" json:"email"`
	Username string `gorm:"type:varchar(50);uniqueIndex" json:"username"` // 唯一索引
	Password string `gorm:"type:varchar(255)" json:"-"`                   // 不返回给前端
}
