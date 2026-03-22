package deepagent

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const workspaceDirName = "workspace"

func workspaceRootPath() (string, error) {
	return filepath.Abs(workspaceDirName)
}

func workspacePathForUser(userUUID string) (string, error) {
	if strings.TrimSpace(userUUID) == "" {
		return "", fmt.Errorf("user uuid is required")
	}
	root, err := workspaceRootPath()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, userUUID), nil
}

func containerNameForUser(userRefID uint) string {
	return fmt.Sprintf("gopherai-deepagent-u%d", userRefID)
}

func ensureWorkspace(workspacePath string) error {
	if info, err := os.Stat(workspacePath); err == nil && info.IsDir() {
		return nil
	}
	return rebuildWorkspace(workspacePath)
}

func rebuildWorkspace(workspacePath string) error {
	if err := os.RemoveAll(workspacePath); err != nil {
		return err
	}
	return os.MkdirAll(workspacePath, 0o755)
}

func isSameOrChild(path, root string) bool {
	cleanPath := filepath.Clean(path)
	cleanRoot := filepath.Clean(root)
	if cleanPath == cleanRoot {
		return true
	}
	return strings.HasPrefix(cleanPath, cleanRoot+string(os.PathSeparator))
}
