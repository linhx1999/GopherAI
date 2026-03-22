package session

import (
	"log"
	"time"

	"github.com/google/uuid"

	"GopherAI/common/code"
	messageDAO "GopherAI/dao/message"
	"GopherAI/dao/session"
	"GopherAI/model"
)

func buildSessionInfo(s model.Session) model.SessionInfo {
	return model.SessionInfo{
		SessionID: s.SessionID,
		Title:     s.Title,
		CreatedAt: s.CreatedAt.Format(time.RFC3339),
	}
}

// GetUserSessionsByUserRefID 获取用户会话列表
func GetUserSessionsByUserRefID(userRefID uint) ([]model.SessionInfo, error) {
	sessions, err := session.GetSessionsByUserRefID(userRefID)
	if err != nil {
		log.Println("GetUserSessionsByUserRefID error:", err)
		return nil, err
	}

	sessionInfos := make([]model.SessionInfo, 0, len(sessions))
	for _, s := range sessions {
		sessionInfos = append(sessionInfos, buildSessionInfo(s))
	}

	return sessionInfos, nil
}

// CreateSession 创建空会话
func CreateSession(userRefID uint) (*model.SessionInfo, code.Code) {
	newSession := &model.Session{
		SessionID: uuid.New().String(),
		UserRefID: userRefID,
		Title:     model.DefaultSessionTitle,
	}

	createdSession, err := session.CreateSession(newSession)
	if err != nil {
		log.Println("CreateSession error:", err)
		return nil, code.CodeServerBusy
	}

	sessionInfo := buildSessionInfo(*createdSession)
	return &sessionInfo, code.CodeSuccess
}

// DeleteSession 删除会话
func DeleteSession(userRefID uint, sessionID string) code.Code {
	targetSession, err := session.GetSessionBySessionIDAndUserRefID(sessionID, userRefID)
	if err != nil {
		log.Println("DeleteSession lookup error:", err)
		return code.CodeServerBusy
	}
	if targetSession == nil {
		return code.CodeSessionNotFound
	}

	if err := messageDAO.DeleteBySessionRefID(targetSession.ID); err != nil {
		log.Println("DeleteSession delete messages error:", err)
		return code.CodeServerBusy
	}

	deleted, err := session.DeleteSessionBySessionIDAndUserRefID(sessionID, userRefID)
	if err != nil {
		log.Println("DeleteSession error:", err)
		return code.CodeServerBusy
	}
	if !deleted {
		return code.CodeSessionNotFound
	}

	_ = messageDAO.DeleteCachedMessages(sessionID)

	return code.CodeSuccess
}

// UpdateSessionTitle 更新会话标题
func UpdateSessionTitle(userRefID uint, sessionID string, title string) code.Code {
	updated, err := session.UpdateSessionTitleBySessionIDAndUserRefID(sessionID, userRefID, title)
	if err != nil {
		log.Println("UpdateSessionTitle error:", err)
		return code.CodeServerBusy
	}
	if !updated {
		return code.CodeSessionNotFound
	}

	return code.CodeSuccess
}
