/*
To Do:
- Support either stdio or http
- Add logging to a file
- Use https
- Authenticate with a shared secret
*/

package main

import (
	"context"
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
		var err error
		rootDir, err = os.UserHomeDir()
		if err != nil {
			return "", err
		}
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

	root, err := os.OpenRoot(rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s", os.Args[0], err)
		os.Exit(1)
	}
	defer root.Close()

	srvr := mcp.NewServer(&mcp.Implementation{
		Name:    "filemcp",
		Version: "0.1.0",
	}, nil)

	ft := fileTools{
		fs: root.FS(),
	}
	ft.registerTools(srvr)

	ctx := context.Background()
	err = srvr.Run(ctx, &mcp.StdioTransport{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s", os.Args[0], err)
		os.Exit(1)
	}
}
