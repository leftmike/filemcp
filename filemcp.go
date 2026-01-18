package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var (
	rootDir string
)

func sanitizePath(rootDir, path string) (string, error) {
	if filepath.IsAbs(path) {
		return "", fmt.Errorf("bad path: %s", path)
	}
	path = filepath.Join(rootDir, filepath.Clean(path))
	if !strings.HasPrefix(path, rootDir) {
		return "", fmt.Errorf("bad path: %s", path)
	}
	return path, nil
}

func main() {
	switch len(os.Args) {
	case 1:
		rootDir = "."
	case 2:
		rootDir = os.Args[1]
	default:
		fmt.Fprintf(os.Stderr, "%s: too many arguments", os.Args[0])
		os.Exit(1)
	}

	var err error
	rootDir, err = filepath.Abs(rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s", os.Args[0], err)
		os.Exit(1)
	}

}
