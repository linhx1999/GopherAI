package session

import (
	"log"

	"GopherAI/common/code"
	redis_cache "GopherAI/common/redis"
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
			SessionID: s.ID,
			Title:     s.Title,
		})
	}

	return sessionInfos, nil
}

// DeleteSession 删除会话
func DeleteSession(userName string, sessionID string) code.Code {
	_ = redis_cache.DeleteMessages(sessionID)

	if err := session.DeleteSession(sessionID, userName); err != nil {
		log.Println("DeleteSession error:", err)
		return code.CodeServerBusy
	}

	return code.CodeSuccess
}

// UpdateSessionTitle 更新会话标题
func UpdateSessionTitle(userName string, sessionID string, title string) code.Code {
	if err := session.UpdateSessionTitle(sessionID, userName, title); err != nil {
		log.Println("UpdateSessionTitle error:", err)
		return code.CodeServerBusy
	}

	return code.CodeSuccess
}
