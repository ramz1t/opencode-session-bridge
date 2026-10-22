package bridge

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"opencode-session-bridge/internal/opencode"
)

// resolveDirectory turns a project name or absolute path into a directory.
// Known session directories are checked before the configured roots.
func resolveDirectory(project string, known []opencode.Session, roots []string) (string, error) {
	project = strings.TrimSpace(project)
	if filepath.IsAbs(project) {
		if !isDir(project) {
			return "", fmt.Errorf("directory %s does not exist", project)
		}
		return filepath.Clean(project), nil
	}

	for _, s := range known {
		if strings.EqualFold(filepath.Base(s.Directory), project) && isDir(s.Directory) {
			return s.Directory, nil
		}
	}
	for _, root := range roots {
		if candidate := filepath.Join(root, project); isDir(candidate) {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("could not resolve project %q: pass an absolute path or add a root to SESSION_BRIDGE_PROJECT_ROOTS", project)
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
