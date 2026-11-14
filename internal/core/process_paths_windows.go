//go:build windows

package core

import (
	"os"
	"strings"
)

func protectedProcessDirectory(dir string) bool {
	root := processPath(os.Getenv("SystemRoot"))
	return root != "" && (dir == root || strings.HasPrefix(dir, root+"/"))
}
