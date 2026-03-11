package session

import (
	"GopherAI/common/postgres"
	"GopherAI/model"
)

func GetSessionsByUserRefID(userRefID uint) ([]model.Session, error) {
	var sessions []model.Session
	err := postgres.DB.Where("user_ref_id = ?", userRefID).Order("created_at DESC").Find(&sessions).Error
	return sessions, err
}

func CreateSession(session *model.Session) (*model.Session, error) {
	err := postgres.DB.Create(session).Error
	return session, err
}

func GetSessionBySessionIDAndUserRefID(sessionID string, userRefID uint) (*model.Session, error) {
	var session model.Session
	err := postgres.DB.
		Where("session_id = ? AND user_ref_id = ?", sessionID, userRefID).
		First(&session).Error
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func DeleteSessionBySessionIDAndUserRefID(sessionID string, userRefID uint) (bool, error) {
	result := postgres.DB.
		Where("session_id = ? AND user_ref_id = ?", sessionID, userRefID).
		Delete(&model.Session{})
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

func UpdateSessionTitleBySessionIDAndUserRefID(sessionID string, userRefID uint, title string) (bool, error) {
	result := postgres.DB.Model(&model.Session{}).
		Where("session_id = ? AND user_ref_id = ?", sessionID, userRefID).
		Update("title", title)
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}
