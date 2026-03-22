package deepagent

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWorkspacePathForUserUsesBusinessUUID(t *testing.T) {
	expected, err := filepath.Abs(filepath.Join(workspaceDirName, "user-uuid-1"))
	if err != nil {
		t.Fatalf("filepath.Abs failed: %v", err)
	}

	got, err := workspacePathForUser("user-uuid-1")
	if err != nil {
		t.Fatalf("workspacePathForUser failed: %v", err)
	}
	if got != expected {
		t.Fatalf("expected %q, got %q", expected, got)
	}
}

func TestEnsureWorkspaceCreatesEmptyDirectory(t *testing.T) {
	workspacePath := filepath.Join(t.TempDir(), "workspace", "user-uuid-2")
	if err := ensureWorkspace(workspacePath); err != nil {
		t.Fatalf("ensureWorkspace failed: %v", err)
	}

	entries, err := os.ReadDir(workspacePath)
	if err != nil {
		t.Fatalf("ReadDir failed: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected empty workspace, got %d entries", len(entries))
	}
}

func TestRebuildWorkspaceClearsDirectory(t *testing.T) {
	workspacePath := filepath.Join(t.TempDir(), "workspace", "user-uuid-3")
	if err := os.MkdirAll(workspacePath, 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workspacePath, "hello.txt"), []byte("world"), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	if err := rebuildWorkspace(workspacePath); err != nil {
		t.Fatalf("rebuildWorkspace failed: %v", err)
	}

	entries, err := os.ReadDir(workspacePath)
	if err != nil {
		t.Fatalf("ReadDir failed: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected empty workspace after rebuild, got %d entries", len(entries))
	}
}
