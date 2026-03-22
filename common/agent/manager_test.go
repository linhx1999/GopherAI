package agent

import (
	"context"
	"testing"
)

func TestGetAgentManagerReturnsSingleton(t *testing.T) {
	firstManager := GetAgentManager()
	secondManager := GetAgentManager()
	if firstManager != secondManager {
		t.Fatal("expected shared agent manager instance")
	}
}

func TestBuildAgentRequiresModel(t *testing.T) {
	_, err := GetAgentManager().buildAgent(context.Background(), &AgentSessionConfig{})
	if err == nil {
		t.Fatal("expected buildAgent to reject missing model")
	}
}
