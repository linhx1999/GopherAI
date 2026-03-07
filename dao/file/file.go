package file

import (
	"GopherAI/common/postgres"
	"GopherAI/model"

	"gorm.io/gorm"
)

// Create 创建文件元数据记录
func Create(file *model.File) error {
	return postgres.DB.Create(file).Error
}

// GetByUserName 根据用户名获取所有文件
func GetByUserName(userName string) ([]model.File, error) {
	var files []model.File
	err := postgres.DB.Where("user_name = ?", userName).Order("created_at DESC").Find(&files).Error
	return files, err
}

// GetByID 根据 ID 获取文件
func GetByID(id uint) (*model.File, error) {
	var file model.File
	err := postgres.DB.First(&file, id).Error
	if err != nil {
		return nil, err
	}
	return &file, nil
}

// GetByObjectName 根据对象名获取文件
func GetByObjectName(objectName string) (*model.File, error) {
	var file model.File
	err := postgres.DB.Where("object_name = ?", objectName).First(&file).Error
	if err != nil {
		return nil, err
	}
	return &file, nil
}

// Delete 根据 ID 删除文件记录（软删除）
func Delete(id uint) error {
	return postgres.DB.Delete(&model.File{}, id).Error
}

// DeleteByUserName 删除用户的所有文件记录（软删除）
func DeleteByUserName(userName string) error {
	return postgres.DB.Where("user_name = ?", userName).Delete(&model.File{}).Error
}

// IsExistByObjectName 检查对象名是否已存在
func IsExistByObjectName(objectName string) bool {
	var count int64
	postgres.DB.Model(&model.File{}).Where("object_name = ?", objectName).Count(&count)
	return count > 0
}

// GetByIDAndUserName 根据 ID 和用户名获取文件（用于权限校验）
func GetByIDAndUserName(id uint, userName string) (*model.File, error) {
	var file model.File
	err := postgres.DB.Where("id = ? AND user_name = ?", id, userName).First(&file).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &file, nil
}

// UpdateIndexStatus 更新文件索引状态
func UpdateIndexStatus(id uint, status string, message string) error {
	updates := map[string]interface{}{
		"index_status": status,
	}
	if message != "" {
		updates["index_message"] = message
	}
	return postgres.DB.Model(&model.File{}).Where("id = ?", id).Updates(updates).Error
}
