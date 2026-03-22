package agent

import (
	"context"
	"reflect"
	"testing"

	agenttools "GopherAI/common/agent/tools"
)

func TestBuildToolSignaturePreservesRequestOrder(t *testing.T) {
	signature := buildToolSignature([]string{
		" sequentialthinking ",
		"knowledge_search",
		"sequentialthinking",
	})

	if signature != "sequentialthinking,knowledge_search" {
		t.Fatalf("unexpected tool signature: %q", signature)
	}
}

func TestBuildToolInstancesPreservesRequestOrder(t *testing.T) {
	toolInstances, err := buildToolInstances(context.Background(), []string{
		agenttools.SequentialThinkingToolName(),
		"knowledge_search",
	}, nil)
	if err != nil {
		t.Fatalf("buildToolInstances returned error: %v", err)
	}

	actualNames := make([]string, 0, len(toolInstances))
	for _, toolInstance := range toolInstances {
		info, infoErr := toolInstance.Info(context.Background())
		if infoErr != nil {
			t.Fatalf("tool info error: %v", infoErr)
		}
		actualNames = append(actualNames, info.Name)
	}

	expectedNames := []string{agenttools.SequentialThinkingToolName(), "knowledge_search"}
	if !reflect.DeepEqual(actualNames, expectedNames) {
		t.Fatalf("unexpected tool order: %#v", actualNames)
	}
}

func TestBuildToolInstancesReturnsUnknownToolError(t *testing.T) {
	_, err := buildToolInstances(context.Background(), []string{"missing_tool"}, nil)
	if !agenttools.IsUnknownToolError(err) {
		t.Fatalf("expected unknown tool error, got %T", err)
	}
}
