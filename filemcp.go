package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func rootDirectory(args []string) (string, error) {
	var rootDir string
	switch len(args) {
	case 0:
		rootDir = "."
	case 1:
		rootDir = args[0]
	default:
		return "", fmt.Errorf("too many arguments: %s", strings.Join(os.Args, " "))
	}

	rootDir, err := filepath.Abs(rootDir)
	if err != nil {
		return "", err
	}

	fi, err := os.Stat(rootDir)
	if err != nil {
		return "", err
	} else if !fi.IsDir() {
		return "", fmt.Errorf("not a directory: %s", rootDir)
	}

	return rootDir, nil
}

func main() {
	rootDir, err := rootDirectory(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s", os.Args[0], err)
		os.Exit(1)
	}

	srvr := mcp.NewServer(&mcp.Implementation{
		Name:    "filemcp",
		Version: "0.1.0",
	}, nil)

	ft := fileTools{
		rootDir: rootDir,
	}
	ft.registerTools(srvr)

	// XXX: run server
}
