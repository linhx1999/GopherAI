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

func GetSessionByID(sessionID string) (*model.Session, error) {
	var session model.Session
	err := postgres.DB.Where("id = ?", sessionID).First(&session).Error
	return &session, err
}

func DeleteSession(sessionID string, userName string) error {
	result := postgres.DB.Where("id = ? AND user_name = ?", sessionID, userName).Delete(&model.Session{})
	return result.Error
}

func UpdateSessionTitle(sessionID string, userName string, title string) error {
	result := postgres.DB.Model(&model.Session{}).
		Where("id = ? AND user_name = ?", sessionID, userName).
		Update("title", title)
	log.Printf("UpdateSessionTitle: sessionID=%s, userName=%s, title=%s, rowsAffected=%d, error=%v\n",
		sessionID, userName, title, result.RowsAffected, result.Error)
	return result.Error
}
