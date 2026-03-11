package session

import (
	"log"
	"time"

	"GopherAI/common/code"
	messageDAO "GopherAI/dao/message"
	"GopherAI/dao/session"
	"GopherAI/model"
)

// GetUserSessionsByUserName 获取用户会话列表
func GetUserSessionsByUserName(userName string) ([]model.SessionInfo, error) {
	sessions, err := session.GetSessionsByUserName(userName)
	if err != nil {
		log.Println("GetUserSessionsByUserName error:", err)
		return nil, err
	}

	sessionInfos := make([]model.SessionInfo, 0, len(sessions))
	for _, s := range sessions {
		sessionInfos = append(sessionInfos, model.SessionInfo{
			SessionID: s.SessionID,
			Title:     s.Title,
			CreatedAt: s.CreatedAt.Format(time.RFC3339),
		})
	}

	return sessionInfos, nil
}

// DeleteSession 删除会话
func DeleteSession(userName string, sessionID string) code.Code {
	deleted, err := session.DeleteSessionByIDAndUserName(sessionID, userName)
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
func UpdateSessionTitle(userName string, sessionID string, title string) code.Code {
	updated, err := session.UpdateSessionTitleByIDAndUserName(sessionID, userName, title)
	if err != nil {
		log.Println("UpdateSessionTitle error:", err)
		return code.CodeServerBusy
	}
	if !updated {
		return code.CodeSessionNotFound
	}

	return code.CodeSuccess
}
