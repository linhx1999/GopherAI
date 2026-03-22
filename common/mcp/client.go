package mcp

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	einomcp "github.com/cloudwego/eino-ext/components/tool/mcp"
	"github.com/cloudwego/eino/components/tool"
	mcpclient "github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
)

const clientVersion = "1.0.0"

type ServerConfig struct {
	TransportType string
	Endpoint      string
	Headers       map[string]string
}

type ToolSnapshotItem struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
}

type Connection struct {
	client *mcpclient.Client
	tools  []tool.BaseTool
}

func validateEndpoint(endpoint string) error {
	parsedURL, err := url.ParseRequestURI(strings.TrimSpace(endpoint))
	if err != nil {
		return fmt.Errorf("invalid mcp endpoint: %w", err)
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("mcp endpoint must use http or https")
	}

	return nil
}

func normalizeTransportType(transportType string) string {
	normalized := strings.TrimSpace(strings.ToLower(transportType))
	if normalized == "" {
		return "sse"
	}
	return normalized
}

func Connect(ctx context.Context, cfg ServerConfig) (*Connection, error) {
	if !FeatureEnabled() {
		return nil, ErrFeatureDisabled
	}
	if err := validateEndpoint(cfg.Endpoint); err != nil {
		return nil, err
	}

	transportType := normalizeTransportType(cfg.TransportType)

	var cli *mcpclient.Client
	var err error
	switch transportType {
	case "sse":
		options := make([]transport.ClientOption, 0, 1)
		if len(cfg.Headers) > 0 {
			options = append(options, transport.WithHeaders(cfg.Headers))
		}

		cli, err = mcpclient.NewSSEMCPClient(cfg.Endpoint, options...)
		if err != nil {
			return nil, fmt.Errorf("create sse mcp client: %w", err)
		}
	case "http":
		options := make([]transport.StreamableHTTPCOption, 0, 1)
		if len(cfg.Headers) > 0 {
			options = append(options, transport.WithHTTPHeaders(cfg.Headers))
		}

		httpTransport, transportErr := transport.NewStreamableHTTP(cfg.Endpoint, options...)
		if transportErr != nil {
			return nil, fmt.Errorf("create http mcp transport: %w", transportErr)
		}
		cli = mcpclient.NewClient(httpTransport)
	default:
		return nil, fmt.Errorf("unsupported mcp transport type: %s", transportType)
	}

	cleanup := func() {
		_ = cli.Close()
	}

	if err := cli.Start(ctx); err != nil {
		cleanup()
		return nil, fmt.Errorf("start mcp client: %w", err)
	}

	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "GopherAI",
		Version: clientVersion,
	}
	initRequest.Params.Capabilities = mcp.ClientCapabilities{}

	if _, err := cli.Initialize(ctx, initRequest); err != nil {
		cleanup()
		return nil, fmt.Errorf("initialize mcp client: %w", err)
	}

	mcpTools, err := einomcp.GetTools(ctx, &einomcp.Config{
		Cli:           cli,
		CustomHeaders: cfg.Headers,
	})
	if err != nil {
		cleanup()
		return nil, fmt.Errorf("load mcp tools: %w", err)
	}

	return &Connection{
		client: cli,
		tools:  mcpTools,
	}, nil
}

func (c *Connection) Tools() []tool.BaseTool {
	if c == nil {
		return nil
	}
	return c.tools
}

func (c *Connection) Close() error {
	if c == nil || c.client == nil {
		return nil
	}
	return c.client.Close()
}

func BuildToolSnapshot(ctx context.Context, tools []tool.BaseTool) ([]ToolSnapshotItem, error) {
	snapshot := make([]ToolSnapshotItem, 0, len(tools))
	for _, currentTool := range tools {
		if currentTool == nil {
			continue
		}

		info, err := currentTool.Info(ctx)
		if err != nil {
			return nil, fmt.Errorf("read mcp tool info: %w", err)
		}
		if info == nil {
			continue
		}

		snapshot = append(snapshot, ToolSnapshotItem{
			Name:        info.Name,
			DisplayName: info.Name,
			Description: strings.TrimSpace(info.Desc),
		})
	}

	return snapshot, nil
}

func CloneToolSnapshot(items []ToolSnapshotItem) []ToolSnapshotItem {
	cloned := make([]ToolSnapshotItem, 0, len(items))
	cloned = append(cloned, items...)
	return cloned
}
