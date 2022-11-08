package artifacts

import (
	"os"
	"path/filepath"
	"strings"
)

// resolveFullPath resolves the full path of s if s is a relative path.
func resolveFullPath(s string) string {
	fullPath := s
	if !strings.HasPrefix(s, "/") {
		cwd, _ := os.Getwd()
		fullPath = filepath.Join(cwd, s)
	}
	return fullPath
}
