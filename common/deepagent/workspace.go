package deepagent

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"GopherAI/config"
)

func workspacePathForUser(userRefID uint) (string, error) {
	root, err := filepath.Abs(config.GetConfig().DeepAgentConfig.WorkspaceRoot)
	if err != nil {
		return "", err
	}
	return filepath.Join(root, strconv.FormatUint(uint64(userRefID), 10), "workspace"), nil
}

func containerNameForUser(userRefID uint) string {
	return fmt.Sprintf("gopherai-deepagent-u%d", userRefID)
}

func ensureWorkspace(userRefID uint, workspacePath string) error {
	if info, err := os.Stat(workspacePath); err == nil && info.IsDir() {
		return nil
	}
	return rebuildWorkspace(userRefID, workspacePath)
}

func rebuildWorkspace(userRefID uint, workspacePath string) error {
	if err := os.RemoveAll(workspacePath); err != nil {
		return err
	}

	parentDir := filepath.Dir(workspacePath)
	if err := os.MkdirAll(parentDir, 0o755); err != nil {
		return err
	}

	templateDir, err := filepath.Abs(config.GetConfig().DeepAgentConfig.TemplateDir)
	if err != nil {
		return err
	}
	workspaceRoot, err := filepath.Abs(config.GetConfig().DeepAgentConfig.WorkspaceRoot)
	if err != nil {
		return err
	}

	return copyTemplateDir(templateDir, workspacePath, workspaceRoot)
}

func copyTemplateDir(srcRoot, dstRoot, workspaceRoot string) error {
	skipRelPrefixes := map[string]struct{}{
		".git":                  {},
		".env":                  {},
		"frontend/node_modules": {},
		"frontend/dist":         {},
	}

	if rel, err := filepath.Rel(srcRoot, workspaceRoot); err == nil && rel != "." && !strings.HasPrefix(rel, "..") {
		skipRelPrefixes[filepath.ToSlash(rel)] = struct{}{}
	}

	return filepath.WalkDir(srcRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		absPath, err := filepath.Abs(path)
		if err != nil {
			return err
		}
		if isSameOrChild(absPath, workspaceRoot) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		relPath, err := filepath.Rel(srcRoot, absPath)
		if err != nil {
			return err
		}
		if relPath == "." {
			return os.MkdirAll(dstRoot, 0o755)
		}

		relPath = filepath.ToSlash(relPath)
		for prefix := range skipRelPrefixes {
			if relPath == prefix || strings.HasPrefix(relPath, prefix+"/") {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		if d.Type()&os.ModeSymlink != 0 {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		targetPath := filepath.Join(dstRoot, filepath.FromSlash(relPath))
		if d.IsDir() {
			return os.MkdirAll(targetPath, 0o755)
		}

		info, err := d.Info()
		if err != nil {
			return err
		}
		return copyFile(absPath, targetPath, info.Mode())
	})
}

func copyFile(srcPath, dstPath string, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
		return err
	}

	srcFile, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.OpenFile(dstPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode.Perm())
	if err != nil {
		return err
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return err
	}
	return nil
}

func isSameOrChild(path, root string) bool {
	cleanPath := filepath.Clean(path)
	cleanRoot := filepath.Clean(root)
	if cleanPath == cleanRoot {
		return true
	}
	return strings.HasPrefix(cleanPath, cleanRoot+string(os.PathSeparator))
}
