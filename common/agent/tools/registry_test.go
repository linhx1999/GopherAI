package tools

import (
	"context"
	"reflect"
	"testing"
)

func TestNormalizeToolNamesPreservesRequestOrder(t *testing.T) {
	names := NormalizeToolNames([]string{
		" sequential_thinking ",
		"knowledge_search",
		"sequential_thinking",
		"",
		"knowledge_search",
	})

	expected := []string{"sequential_thinking", "knowledge_search"}
	if !reflect.DeepEqual(names, expected) {
		t.Fatalf("unexpected normalized names: %#v", names)
	}
}

func TestBuildRequestedToolsReturnsUnknownToolError(t *testing.T) {
	_, err := BuildRequestedTools(context.Background(), []string{"missing_tool"}, nil)
	if err == nil {
		t.Fatal("expected unknown tool error")
	}
	if !IsUnknownToolError(err) {
		t.Fatalf("expected unknown tool error, got %T", err)
	}
}

func TestBuildRequestedToolsPreservesRequestOrder(t *testing.T) {
	builtTools, err := BuildRequestedTools(context.Background(), []string{
		"sequential_thinking",
		"knowledge_search",
		"sequential_thinking",
	}, nil)
	if err != nil {
		t.Fatalf("BuildRequestedTools returned error: %v", err)
	}

	expectedNames := []string{"sequential_thinking", "knowledge_search"}
	actualNames := make([]string, 0, len(builtTools))
	for _, builtTool := range builtTools {
		info, infoErr := builtTool.Info(context.Background())
		if infoErr != nil {
			t.Fatalf("tool info error: %v", infoErr)
		}
		actualNames = append(actualNames, info.Name)
	}

	if !reflect.DeepEqual(actualNames, expectedNames) {
		t.Fatalf("unexpected tool order: %#v", actualNames)
	}
}

func TestBuildRequestedToolsInjectsKnowledgeSearchWithoutIndexedFiles(t *testing.T) {
	builtTools, err := BuildRequestedTools(context.Background(), []string{"knowledge_search"}, nil)
	if err != nil {
		t.Fatalf("BuildRequestedTools returned error: %v", err)
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

func TestListAvailableToolsReturnsGlobalToolMap(t *testing.T) {
	toolList := ListAvailableTools()
	if len(toolList) != 2 {
		t.Fatalf("unexpected tool count: %d", len(toolList))
	}

	actualNames := []string{toolList[0].Name, toolList[1].Name}
	expectedNames := []string{"sequential_thinking", "knowledge_search"}
	if !reflect.DeepEqual(actualNames, expectedNames) {
		t.Fatalf("unexpected tool list: %#v", actualNames)
	}
}
