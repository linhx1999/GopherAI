package tools

import (
	"context"
	"reflect"
	"testing"
)

func TestResolveRequestedToolsPreservesRequestOrder(t *testing.T) {
	resolvedTools, err := ResolveRequestedTools(context.Background(), []string{
		" sequentialthinking ",
		"knowledge_search",
		"sequentialthinking",
		"",
		"knowledge_search",
	})
	if err != nil {
		t.Fatalf("ResolveRequestedTools returned error: %v", err)
	}

	actualNames := make([]string, 0, len(resolvedTools))
	for _, resolvedTool := range resolvedTools {
		info, infoErr := resolvedTool.Info(context.Background())
		if infoErr != nil {
			t.Fatalf("resolved tool info error: %v", infoErr)
		}
		actualNames = append(actualNames, info.Name)
	}

	expected := []string{"sequentialthinking", "knowledge_search"}
	if !reflect.DeepEqual(actualNames, expected) {
		t.Fatalf("unexpected resolved tools: %#v", actualNames)
	}
}

func TestListAvailableToolsReturnsGlobalToolMap(t *testing.T) {
	toolList := ListAvailableTools()
	if len(toolList) != 2 {
		t.Fatalf("unexpected tool count: %d", len(toolList))
	}

	actualNames := []string{toolList[0].Name, toolList[1].Name}
	expectedNames := []string{"knowledge_search", "sequentialthinking"}
	if !reflect.DeepEqual(actualNames, expectedNames) {
		t.Fatalf("unexpected tool list: %#v", actualNames)
	}

	sequentialThinkingTool := toolList[1]
	if toolList[0].Name == SequentialThinkingToolName() {
		sequentialThinkingTool = toolList[0]
	}
	if sequentialThinkingTool.Name != SequentialThinkingToolName() {
		t.Fatalf("unexpected sequentialthinking tool name: %q", sequentialThinkingTool.Name)
	}
	if sequentialThinkingTool.DisplayName == "" || sequentialThinkingTool.Description == "" {
		t.Fatalf("expected sequentialthinking tool catalog fields, got %#v", sequentialThinkingTool)
	}
}

func TestGetSequentialThinkingToolReturnsFreshInstances(t *testing.T) {
	firstTool, err := GetSequentialThinkingTool()
	if err != nil {
		t.Fatalf("GetSequentialThinkingTool returned error: %v", err)
	}

	secondTool, err := GetSequentialThinkingTool()
	if err != nil {
		t.Fatalf("GetSequentialThinkingTool returned error: %v", err)
	}

	if reflect.ValueOf(firstTool).Pointer() == reflect.ValueOf(secondTool).Pointer() {
		t.Fatal("expected sequentialthinking tool instances to be isolated per build")
	}

	firstInfo, err := firstTool.Info(context.Background())
	if err != nil {
		t.Fatalf("first tool info error: %v", err)
	}
	secondInfo, err := secondTool.Info(context.Background())
	if err != nil {
		t.Fatalf("second tool info error: %v", err)
	}

	if firstInfo.Name != SequentialThinkingToolName() || secondInfo.Name != SequentialThinkingToolName() {
		t.Fatalf("unexpected sequentialthinking runtime name: %q, %q", firstInfo.Name, secondInfo.Name)
	}
}

func TestSequentialThinkingToolDefinitionKeepsInitializedTool(t *testing.T) {
	if SequentialThinkingTool.tool == nil {
		t.Fatal("expected sequentialthinking definition to keep initialized upstream tool")
	}
}

func TestResolveRequestedToolsReturnsUnknownToolError(t *testing.T) {
	_, err := ResolveRequestedTools(context.Background(), []string{"missing_tool"})
	if !IsUnknownToolError(err) {
		t.Fatalf("expected unknown tool error, got %T", err)
	}
}

func TestResolveRequestedToolsInjectsKnowledgeSearchWithoutFileScope(t *testing.T) {
	builtTools, err := ResolveRequestedTools(context.Background(), []string{"knowledge_search"})
	if err != nil {
		t.Fatalf("ResolveRequestedTools returned error: %v", err)
	}
	if len(builtTools) != 1 {
		t.Fatalf("unexpected tool count: %d", len(builtTools))
	}

	info, err := builtTools[0].Info(context.Background())
	if err != nil {
		t.Fatalf("tool info error: %v", err)
	}
	if info.Name != "knowledge_search" {
		t.Fatalf("unexpected tool name: %q", info.Name)
	}
}
