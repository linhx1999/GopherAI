package deepagent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	adkfs "github.com/cloudwego/eino/adk/filesystem"
)

type RootBackend struct {
	root string
}

func NewRootBackend(root string) *RootBackend {
	return &RootBackend{root: filepath.Clean(root)}
}

func (b *RootBackend) resolvePath(path string) (string, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" || trimmed == "/" || trimmed == "." {
		return b.root, nil
	}

	trimmed = strings.TrimPrefix(trimmed, "/")
	cleaned := filepath.Clean(trimmed)
	if cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("path escapes workspace root")
	}

	absPath := filepath.Join(b.root, cleaned)
	if !isSameOrChild(absPath, b.root) {
		return "", fmt.Errorf("path escapes workspace root")
	}
	return absPath, nil
}

func (b *RootBackend) relPath(absPath string) string {
	rel, err := filepath.Rel(b.root, absPath)
	if err != nil || rel == "." {
		return "."
	}
	return filepath.ToSlash(rel)
}

func (b *RootBackend) LsInfo(ctx context.Context, req *adkfs.LsInfoRequest) ([]adkfs.FileInfo, error) {
	dirPath, err := b.resolvePath(req.Path)
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	result := make([]adkfs.FileInfo, 0, len(entries))
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			return nil, err
		}
		result = append(result, adkfs.FileInfo{
			Path:       entry.Name(),
			IsDir:      entry.IsDir(),
			Size:       info.Size(),
			ModifiedAt: info.ModTime().UTC().Format("2006-01-02T15:04:05Z07:00"),
		})
	}
	return result, nil
}

func (b *RootBackend) Read(ctx context.Context, req *adkfs.ReadRequest) (*adkfs.FileContent, error) {
	filePath, err := b.resolvePath(req.FilePath)
	if err != nil {
		return nil, err
	}

	contentBytes, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(contentBytes), "\n")
	offset := req.Offset
	if offset < 1 {
		offset = 1
	}
	limit := req.Limit
	if limit <= 0 {
		limit = 2000
	}

	start := offset - 1
	if start >= len(lines) {
		return &adkfs.FileContent{}, nil
	}
	end := start + limit
	if end > len(lines) {
		end = len(lines)
	}

	return &adkfs.FileContent{
		Content: strings.Join(lines[start:end], "\n"),
	}, nil
}

func (b *RootBackend) GrepRaw(ctx context.Context, req *adkfs.GrepRequest) ([]adkfs.GrepMatch, error) {
	pattern := req.Pattern
	if req.CaseInsensitive {
		pattern = "(?i)" + pattern
	}
	if req.EnableMultiline {
		pattern = "(?s)" + pattern
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}

	searchRoot, err := b.resolvePath(req.Path)
	if err != nil {
		return nil, err
	}

	matches := make([]adkfs.GrepMatch, 0)
	err = filepath.WalkDir(searchRoot, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}

		relPath := b.relPath(path)
		if req.Glob != "" {
			ok, globErr := doublestar.Match(req.Glob, relPath)
			if globErr != nil {
				return globErr
			}
			if !ok {
				return nil
			}
		}
		if req.FileType != "" && filepath.Ext(path) != "."+strings.TrimPrefix(req.FileType, ".") {
			return nil
		}

		contentBytes, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		content := string(contentBytes)

		if req.EnableMultiline {
			indexes := re.FindAllStringIndex(content, -1)
			for _, idx := range indexes {
				line := 1 + strings.Count(content[:idx[0]], "\n")
				lineText := extractLine(content, idx[0])
				matches = append(matches, adkfs.GrepMatch{
					Content: lineText,
					Path:    relPath,
					Line:    line,
				})
			}
			return nil
		}

		lines := strings.Split(content, "\n")
		for lineIndex, lineText := range lines {
			if re.MatchString(lineText) {
				matches = append(matches, adkfs.GrepMatch{
					Content: lineText,
					Path:    relPath,
					Line:    lineIndex + 1,
				})
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return matches, nil
}

func (b *RootBackend) GlobInfo(ctx context.Context, req *adkfs.GlobInfoRequest) ([]adkfs.FileInfo, error) {
	searchRoot, err := b.resolvePath(req.Path)
	if err != nil {
		return nil, err
	}

	result := make([]adkfs.FileInfo, 0)
	err = filepath.WalkDir(searchRoot, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == searchRoot {
			return nil
		}

		relPath := b.relPath(path)
		matched, matchErr := doublestar.Match(req.Pattern, relPath)
		if matchErr != nil {
			return matchErr
		}
		if !matched {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return err
		}

		result = append(result, adkfs.FileInfo{
			Path:       relPath,
			IsDir:      d.IsDir(),
			Size:       info.Size(),
			ModifiedAt: info.ModTime().UTC().Format("2006-01-02T15:04:05Z07:00"),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (b *RootBackend) Write(ctx context.Context, req *adkfs.WriteRequest) error {
	filePath, err := b.resolvePath(req.FilePath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(filePath, []byte(req.Content), 0o644)
}

func (b *RootBackend) Edit(ctx context.Context, req *adkfs.EditRequest) error {
	if req.OldString == "" {
		return fmt.Errorf("old string cannot be empty")
	}

	filePath, err := b.resolvePath(req.FilePath)
	if err != nil {
		return err
	}

	contentBytes, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	content := string(contentBytes)

	count := strings.Count(content, req.OldString)
	if count == 0 {
		return fmt.Errorf("old string not found")
	}
	if !req.ReplaceAll && count != 1 {
		return fmt.Errorf("old string must appear exactly once when replace_all is false")
	}

	updated := strings.ReplaceAll(content, req.OldString, req.NewString)
	if !req.ReplaceAll {
		updated = strings.Replace(content, req.OldString, req.NewString, 1)
	}
	return os.WriteFile(filePath, []byte(updated), 0o644)
}

func extractLine(content string, index int) string {
	start := strings.LastIndex(content[:index], "\n")
	if start == -1 {
		start = 0
	} else {
		start++
	}
	endOffset := strings.Index(content[index:], "\n")
	if endOffset == -1 {
		return content[start:]
	}
	return content[start : index+endOffset]
}
