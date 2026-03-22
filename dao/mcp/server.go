package mcp

import (
	"GopherAI/common/postgres"
	"GopherAI/model"
)

func Create(server *model.MCPServer) error {
	return postgres.DB.Create(server).Error
}

func Update(server *model.MCPServer) error {
	return postgres.DB.Save(server).Error
}

func GetByServerIDAndUserRefID(serverID string, userRefID uint) (*model.MCPServer, error) {
	var server model.MCPServer
	err := postgres.DB.
		Where("mcp_server_id = ? AND user_ref_id = ?", serverID, userRefID).
		First(&server).Error
	if err != nil {
		return nil, err
	}
	return &server, nil
}

func ListByUserRefID(userRefID uint) ([]model.MCPServer, error) {
	var servers []model.MCPServer
	err := postgres.DB.
		Where("user_ref_id = ?", userRefID).
		Order("created_at DESC").
		Find(&servers).Error
	return servers, err
}

func ListByServerIDsAndUserRefID(serverIDs []string, userRefID uint) ([]model.MCPServer, error) {
	var servers []model.MCPServer
	err := postgres.DB.
		Where("user_ref_id = ? AND mcp_server_id IN ?", userRefID, serverIDs).
		Find(&servers).Error
	return servers, err
}

func DeleteByServerIDAndUserRefID(serverID string, userRefID uint) (bool, error) {
	result := postgres.DB.
		Where("mcp_server_id = ? AND user_ref_id = ?", serverID, userRefID).
		Delete(&model.MCPServer{})
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}
