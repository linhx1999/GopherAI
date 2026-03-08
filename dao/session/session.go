package session

import (
	"GopherAI/common/postgres"
	"GopherAI/model"
	"log"
)

func GetSessionsByUserName(userName string) ([]model.Session, error) {
	var sessions []model.Session
	err := postgres.DB.Where("user_name = ?", userName).Order("created_at DESC").Find(&sessions).Error
	return sessions, err
}

func CreateSession(session *model.Session) (*model.Session, error) {
	err := postgres.DB.Create(session).Error
	return session, err
}

func GetSessionByIDAndUserName(sessionID string, userName string) (*model.Session, error) {
	var session model.Session
	err := postgres.DB.
		Where("id = ? AND user_name = ?", sessionID, userName).
		First(&session).Error
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func DeleteSessionByIDAndUserName(sessionID string, userName string) (bool, error) {
	result := postgres.DB.
		Where("id = ? AND user_name = ?", sessionID, userName).
		Delete(&model.Session{})
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

func UpdateSessionTitleByIDAndUserName(sessionID string, userName string, title string) (bool, error) {
	result := postgres.DB.Model(&model.Session{}).
		Where("id = ? AND user_name = ?", sessionID, userName).
		Update("title", title)
	log.Printf("UpdateSessionTitle: sessionID=%s, userName=%s, title=%s, rowsAffected=%d, error=%v\n",
		sessionID, userName, title, result.RowsAffected, result.Error)
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}
