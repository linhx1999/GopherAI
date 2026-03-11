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
func GetByUserRefID(userRefID uint) ([]model.File, error) {
	var files []model.File
	err := postgres.DB.Where("user_ref_id = ?", userRefID).Order("created_at DESC").Find(&files).Error
	return files, err
}

// GetByRefID 根据数据库内部 ID 获取文件
func GetByRefID(id uint) (*model.File, error) {
	var file model.File
	err := postgres.DB.First(&file, id).Error
	if err != nil {
		return nil, err
	}
	return &file, nil
}

// GetByFileID 根据业务 FileID 获取文件
func GetByFileID(fileID string) (*model.File, error) {
	var file model.File
	err := postgres.DB.Where("file_id = ?", fileID).First(&file).Error
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

// DeleteByRefID 根据内部 ID 删除文件记录（软删除）
func DeleteByRefID(id uint) error {
	return postgres.DB.Delete(&model.File{}, id).Error
}

// DeleteByUserRefID 删除用户的所有文件记录（软删除）
func DeleteByUserRefID(userRefID uint) error {
	return postgres.DB.Where("user_ref_id = ?", userRefID).Delete(&model.File{}).Error
}

// IsExistByObjectName 检查对象名是否已存在
func IsExistByObjectName(objectName string) bool {
	var count int64
	postgres.DB.Model(&model.File{}).Where("object_name = ?", objectName).Count(&count)
	return count > 0
}

// GetByFileIDAndUserRefID 根据业务 FileID 和用户内部 ID 获取文件（用于权限校验）
func GetByFileIDAndUserRefID(fileID string, userRefID uint) (*model.File, error) {
	var file model.File
	err := postgres.DB.Where("file_id = ? AND user_ref_id = ?", fileID, userRefID).First(&file).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &file, nil
}

// UpdateIndexStatus 更新文件索引状态
func UpdateIndexStatus(fileRefID uint, status string, message string) error {
	updates := map[string]interface{}{
		"index_status": status,
	}
	if message != "" {
		updates["index_message"] = message
	}
	return postgres.DB.Model(&model.File{}).Where("id = ?", fileRefID).Updates(updates).Error
}

// GetIndexedFileRefIDsByUserRefID 获取用户已索引文件的内部 ID 列表
func GetIndexedFileRefIDsByUserRefID(userRefID uint) ([]uint, error) {
	var ids []uint
	err := postgres.DB.Model(&model.File{}).
		Where("user_ref_id = ? AND index_status = ?", userRefID, "indexed").
		Pluck("id", &ids).Error
	return ids, err
}
