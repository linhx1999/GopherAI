package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"GopherAI/common/code"
	commonmcp "GopherAI/common/mcp"
	mcpDAO "GopherAI/dao/mcp"
	"GopherAI/model"
)

const testTimeout = 15 * time.Second

type HeaderInput struct {
	Key          string `json:"key"`
	Value        string `json:"value"`
	KeepExisting bool   `json:"keep_existing"`
}

type HeaderDetail struct {
	Key         string `json:"key"`
	MaskedValue string `json:"masked_value"`
	HasValue    bool   `json:"has_value"`
}

type ToolSnapshotItem = commonmcp.ToolSnapshotItem

type ServerSummary struct {
	ServerID        string             `json:"server_id"`
	Name            string             `json:"name"`
	TransportType   string             `json:"transport_type"`
	Endpoint        string             `json:"endpoint"`
	LastTestStatus  string             `json:"last_test_status"`
	LastTestMessage string             `json:"last_test_message"`
	LastTestedAt    string             `json:"last_tested_at"`
	CreatedAt       string             `json:"created_at"`
	Tools           []ToolSnapshotItem `json:"tools"`
}

type ServerDetail struct {
	ServerSummary
	Headers []HeaderDetail `json:"headers"`
}

type UpsertServerInput struct {
	Name          string        `json:"name"`
	TransportType string        `json:"transport_type"`
	Endpoint      string        `json:"endpoint"`
	Headers       []HeaderInput `json:"headers"`
}

func normalizeServerIDs(serverIDs []string) []string {
	normalized := make([]string, 0, len(serverIDs))
	seen := make(map[string]struct{}, len(serverIDs))
	for _, rawID := range serverIDs {
		serverID := strings.TrimSpace(rawID)
		if serverID == "" {
			continue
		}
		if _, ok := seen[serverID]; ok {
			continue
		}
		seen[serverID] = struct{}{}
		normalized = append(normalized, serverID)
	}
	return normalized
}

func featureDisabledCode() code.Code {
	if commonmcp.FeatureEnabled() {
		return code.CodeSuccess
	}
	return code.CodeMCPFeatureDisabled
}

func parseToolSnapshot(raw json.RawMessage) []ToolSnapshotItem {
	if len(raw) == 0 {
		return []ToolSnapshotItem{}
	}

	var items []ToolSnapshotItem
	if err := json.Unmarshal(raw, &items); err != nil {
		log.Printf("parseToolSnapshot error: %v", err)
		return []ToolSnapshotItem{}
	}
	return items
}

func serializeToolSnapshot(items []ToolSnapshotItem) json.RawMessage {
	if len(items) == 0 {
		return json.RawMessage("[]")
	}

	payload, err := json.Marshal(items)
	if err != nil {
		log.Printf("serializeToolSnapshot error: %v", err)
		return json.RawMessage("[]")
	}
	return payload
}

func formatTimePtr(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.Format(time.RFC3339)
}

func buildSummary(server *model.MCPServer) ServerSummary {
	if server == nil {
		return ServerSummary{}
	}

	return ServerSummary{
		ServerID:        server.MCPServerID,
		Name:            server.Name,
		TransportType:   server.TransportType,
		Endpoint:        server.Endpoint,
		LastTestStatus:  server.LastTestStatus,
		LastTestMessage: server.LastTestMessage,
		LastTestedAt:    formatTimePtr(server.LastTestedAt),
		CreatedAt:       server.CreatedAt.Format(time.RFC3339),
		Tools:           parseToolSnapshot(server.ToolSnapshot),
	}
}

func buildHeaderDetails(ciphertext string) ([]HeaderDetail, error) {
	headers, err := commonmcp.DecryptHeaders(ciphertext)
	if err != nil {
		return nil, err
	}

	keys := commonmcp.SortedHeaderKeys(headers)
	result := make([]HeaderDetail, 0, len(keys))
	for _, key := range keys {
		value := headers[key]
		result = append(result, HeaderDetail{
			Key:         key,
			MaskedValue: commonmcp.MaskSecret(value),
			HasValue:    strings.TrimSpace(value) != "",
		})
	}

	return result, nil
}

func buildDetail(server *model.MCPServer) (*ServerDetail, error) {
	if server == nil {
		return nil, nil
	}

	headers, err := buildHeaderDetails(server.HeadersCiphertext)
	if err != nil {
		return nil, err
	}

	summary := buildSummary(server)
	return &ServerDetail{
		ServerSummary: summary,
		Headers:       headers,
	}, nil
}

func loadOwnedServer(userRefID uint, serverID string) (*model.MCPServer, code.Code) {
	server, err := mcpDAO.GetByServerIDAndUserRefID(serverID, userRefID)
	if err == nil {
		return server, code.CodeSuccess
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, code.CodeMCPServerNotFound
	}

	log.Printf("loadOwnedServer error: %v", err)
	return nil, code.CodeServerBusy
}

func normalizeTransportType(transportType string) string {
	normalized := strings.TrimSpace(strings.ToLower(transportType))
	if normalized == "" {
		return model.MCPTransportSSE
	}
	return normalized
}

func mergeHeaders(existing map[string]string, headerInputs []HeaderInput) (map[string]string, error) {
	headers := make(map[string]string)
	seen := make(map[string]struct{}, len(headerInputs))

	for _, item := range headerInputs {
		key := strings.TrimSpace(item.Key)
		value := strings.TrimSpace(item.Value)
		if key == "" && value == "" {
			continue
		}
		if key == "" {
			return nil, fmt.Errorf("header key is required")
		}
		if _, ok := seen[key]; ok {
			return nil, fmt.Errorf("duplicate header key: %s", key)
		}
		seen[key] = struct{}{}

		if item.KeepExisting {
			existingValue, ok := existing[key]
			if !ok || strings.TrimSpace(existingValue) == "" {
				return nil, fmt.Errorf("header %s has no existing value", key)
			}
			headers[key] = existingValue
			continue
		}

		if value == "" {
			return nil, fmt.Errorf("header %s value is required", key)
		}
		headers[key] = value
	}

	return headers, nil
}

func validateUpsertInput(existing *model.MCPServer, input UpsertServerInput) (string, string, string, string, code.Code) {
	if featureCode := featureDisabledCode(); featureCode != code.CodeSuccess {
		return "", "", "", "", featureCode
	}

	name := strings.TrimSpace(input.Name)
	if name == "" {
		return "", "", "", "", code.CodeInvalidParams
	}

	transportType := normalizeTransportType(input.TransportType)
	if transportType != model.MCPTransportSSE && transportType != model.MCPTransportHTTP {
		return "", "", "", "", code.CodeInvalidParams
	}

	endpoint := strings.TrimSpace(input.Endpoint)
	if endpoint == "" {
		return "", "", "", "", code.CodeInvalidParams
	}

	existingHeaders := map[string]string{}
	if existing != nil {
		headers, err := commonmcp.DecryptHeaders(existing.HeadersCiphertext)
		if err != nil {
			log.Printf("validateUpsertInput decrypt existing headers error: %v", err)
			return "", "", "", "", code.CodeServerBusy
		}
		existingHeaders = headers
	}

	headers, err := mergeHeaders(existingHeaders, input.Headers)
	if err != nil {
		log.Printf("validateUpsertInput header error: %v", err)
		return "", "", "", "", code.CodeInvalidParams
	}

	ciphertext, err := commonmcp.EncryptHeaders(headers)
	if err != nil {
		if errors.Is(err, commonmcp.ErrFeatureDisabled) {
			return "", "", "", "", code.CodeMCPFeatureDisabled
		}
		log.Printf("validateUpsertInput encrypt headers error: %v", err)
		return "", "", "", "", code.CodeServerBusy
	}

	return name, transportType, endpoint, ciphertext, code.CodeSuccess
}

func ListServers(userRefID uint) ([]ServerSummary, code.Code) {
	if featureCode := featureDisabledCode(); featureCode != code.CodeSuccess {
		return nil, featureCode
	}

	servers, err := mcpDAO.ListByUserRefID(userRefID)
	if err != nil {
		log.Printf("ListServers error: %v", err)
		return nil, code.CodeServerBusy
	}

	result := make([]ServerSummary, 0, len(servers))
	for i := range servers {
		result = append(result, buildSummary(&servers[i]))
	}

	return result, code.CodeSuccess
}

func ListToolCatalogServers(userRefID uint) ([]ServerSummary, bool, code.Code) {
	if !commonmcp.FeatureEnabled() {
		return []ServerSummary{}, false, code.CodeSuccess
	}

	servers, err := mcpDAO.ListByUserRefID(userRefID)
	if err != nil {
		log.Printf("ListToolCatalogServers error: %v", err)
		return nil, true, code.CodeServerBusy
	}

	result := make([]ServerSummary, 0, len(servers))
	for i := range servers {
		result = append(result, buildSummary(&servers[i]))
	}
	return result, true, code.CodeSuccess
}

func GetServerDetail(userRefID uint, serverID string) (*ServerDetail, code.Code) {
	if featureCode := featureDisabledCode(); featureCode != code.CodeSuccess {
		return nil, featureCode
	}

	server, code_ := loadOwnedServer(userRefID, serverID)
	if code_ != code.CodeSuccess {
		return nil, code_
	}

	detail, err := buildDetail(server)
	if err != nil {
		log.Printf("GetServerDetail build detail error: %v", err)
		if errors.Is(err, commonmcp.ErrFeatureDisabled) {
			return nil, code.CodeMCPFeatureDisabled
		}
		return nil, code.CodeServerBusy
	}

	return detail, code.CodeSuccess
}

func CreateServer(userRefID uint, input UpsertServerInput) (*ServerDetail, code.Code) {
	name, transportType, endpoint, ciphertext, code_ := validateUpsertInput(nil, input)
	if code_ != code.CodeSuccess {
		return nil, code_
	}

	server := &model.MCPServer{
		MCPServerID:       uuid.New().String(),
		UserRefID:         userRefID,
		Name:              name,
		TransportType:     transportType,
		Endpoint:          endpoint,
		HeadersCiphertext: ciphertext,
		ToolSnapshot:      json.RawMessage("[]"),
		LastTestStatus:    model.MCPTestStatusUntested,
	}

	if err := mcpDAO.Create(server); err != nil {
		log.Printf("CreateServer error: %v", err)
		return nil, code.CodeServerBusy
	}

	detail, err := buildDetail(server)
	if err != nil {
		log.Printf("CreateServer build detail error: %v", err)
		return nil, code.CodeServerBusy
	}

	return detail, code.CodeSuccess
}

func UpdateServer(userRefID uint, serverID string, input UpsertServerInput) (*ServerDetail, code.Code) {
	server, code_ := loadOwnedServer(userRefID, serverID)
	if code_ != code.CodeSuccess {
		return nil, code_
	}

	name, transportType, endpoint, ciphertext, code_ := validateUpsertInput(server, input)
	if code_ != code.CodeSuccess {
		return nil, code_
	}

	server.Name = name
	server.TransportType = transportType
	server.Endpoint = endpoint
	server.HeadersCiphertext = ciphertext

	if err := mcpDAO.Update(server); err != nil {
		log.Printf("UpdateServer error: %v", err)
		return nil, code.CodeServerBusy
	}

	detail, err := buildDetail(server)
	if err != nil {
		log.Printf("UpdateServer build detail error: %v", err)
		return nil, code.CodeServerBusy
	}

	return detail, code.CodeSuccess
}

func DeleteServer(userRefID uint, serverID string) code.Code {
	if featureCode := featureDisabledCode(); featureCode != code.CodeSuccess {
		return featureCode
	}

	deleted, err := mcpDAO.DeleteByServerIDAndUserRefID(serverID, userRefID)
	if err != nil {
		log.Printf("DeleteServer error: %v", err)
		return code.CodeServerBusy
	}
	if !deleted {
		return code.CodeMCPServerNotFound
	}

	return code.CodeSuccess
}

func buildConnectionConfig(server *model.MCPServer) (commonmcp.ServerConfig, error) {
	headers, err := commonmcp.DecryptHeaders(server.HeadersCiphertext)
	if err != nil {
		return commonmcp.ServerConfig{}, err
	}

	return commonmcp.ServerConfig{
		TransportType: server.TransportType,
		Endpoint:      server.Endpoint,
		Headers:       headers,
	}, nil
}

func updateTestResult(server *model.MCPServer, status, message string, toolSnapshot []ToolSnapshotItem) error {
	now := time.Now()
	server.LastTestStatus = status
	server.LastTestMessage = strings.TrimSpace(message)
	server.LastTestedAt = &now
	if toolSnapshot != nil {
		server.ToolSnapshot = serializeToolSnapshot(toolSnapshot)
	}
	return mcpDAO.Update(server)
}

func TestServer(ctx context.Context, userRefID uint, serverID string) (*ServerDetail, code.Code) {
	if featureCode := featureDisabledCode(); featureCode != code.CodeSuccess {
		return nil, featureCode
	}

	server, code_ := loadOwnedServer(userRefID, serverID)
	if code_ != code.CodeSuccess {
		return nil, code_
	}

	cfg, err := buildConnectionConfig(server)
	if err != nil {
		log.Printf("TestServer build connection config error: %v", err)
		return nil, code.CodeServerBusy
	}

	testCtx, cancel := context.WithTimeout(ctx, testTimeout)
	defer cancel()

	conn, err := commonmcp.Connect(testCtx, cfg)
	if err != nil {
		log.Printf("TestServer connect error: %v", err)
		if updateErr := updateTestResult(server, model.MCPTestStatusFailed, err.Error(), nil); updateErr != nil {
			log.Printf("TestServer update failure status error: %v", updateErr)
		}
		detail, buildErr := buildDetail(server)
		if buildErr != nil {
			log.Printf("TestServer build detail after failure error: %v", buildErr)
		}
		return detail, code.CodeMCPConnectionFailed
	}
	defer func() {
		_ = conn.Close()
	}()

	toolSnapshot, err := commonmcp.BuildToolSnapshot(testCtx, conn.Tools())
	if err != nil {
		log.Printf("TestServer snapshot error: %v", err)
		if updateErr := updateTestResult(server, model.MCPTestStatusFailed, err.Error(), nil); updateErr != nil {
			log.Printf("TestServer update failure status error: %v", updateErr)
		}
		detail, buildErr := buildDetail(server)
		if buildErr != nil {
			log.Printf("TestServer build detail after snapshot failure error: %v", buildErr)
		}
		return detail, code.CodeMCPConnectionFailed
	}

	if err := updateTestResult(server, model.MCPTestStatusSuccess, fmt.Sprintf("发现 %d 个工具", len(toolSnapshot)), toolSnapshot); err != nil {
		log.Printf("TestServer update success status error: %v", err)
		return nil, code.CodeServerBusy
	}

	detail, err := buildDetail(server)
	if err != nil {
		log.Printf("TestServer build detail error: %v", err)
		return nil, code.CodeServerBusy
	}

	return detail, code.CodeSuccess
}

func ResolveEnabledTools(ctx context.Context, userRefID uint, serverIDs []string) ([]tool.BaseTool, func(), code.Code) {
	normalizedIDs := normalizeServerIDs(serverIDs)
	if len(normalizedIDs) == 0 {
		return nil, func() {}, code.CodeSuccess
	}

	if featureCode := featureDisabledCode(); featureCode != code.CodeSuccess {
		return nil, func() {}, featureCode
	}

	servers, err := mcpDAO.ListByServerIDsAndUserRefID(normalizedIDs, userRefID)
	if err != nil {
		log.Printf("ResolveEnabledTools list servers error: %v", err)
		return nil, func() {}, code.CodeServerBusy
	}
	if len(servers) != len(normalizedIDs) {
		return nil, func() {}, code.CodeMCPServerNotFound
	}

	serverByID := make(map[string]*model.MCPServer, len(servers))
	for i := range servers {
		serverByID[servers[i].MCPServerID] = &servers[i]
	}

	resolvedTools := make([]tool.BaseTool, 0)
	cleanupFns := make([]func(), 0, len(normalizedIDs))
	cleanup := func() {
		for i := len(cleanupFns) - 1; i >= 0; i-- {
			cleanupFns[i]()
		}
	}

	for _, serverID := range normalizedIDs {
		server := serverByID[serverID]
		if server == nil {
			cleanup()
			return nil, func() {}, code.CodeMCPServerNotFound
		}

		cfg, err := buildConnectionConfig(server)
		if err != nil {
			log.Printf("ResolveEnabledTools build config error: %v", err)
			cleanup()
			return nil, func() {}, code.CodeServerBusy
		}

		conn, err := commonmcp.Connect(ctx, cfg)
		if err != nil {
			log.Printf("ResolveEnabledTools connect error: %v", err)
			cleanup()
			return nil, func() {}, code.CodeMCPConnectionFailed
		}

		cleanupFns = append(cleanupFns, func() {
			_ = conn.Close()
		})
		resolvedTools = append(resolvedTools, conn.Tools()...)
	}

	return resolvedTools, cleanup, code.CodeSuccess
}
