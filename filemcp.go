/*
To Do:
- Support either stdio or sse or http
- Use https
- Authenticate with a shared secret
*/

package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
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

type discardHandler struct{}

func (discardHandler) Enabled(context.Context, slog.Level) bool  { return false }
func (discardHandler) Handle(context.Context, slog.Record) error { return nil }
func (dh discardHandler) WithAttrs([]slog.Attr) slog.Handler     { return dh }
func (dh discardHandler) WithGroup(string) slog.Handler          { return dh }

func setupLogging(log bool, logfile string) {
	if log {
		if logfile != "" {
			file, err := os.OpenFile(logfile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s: %s", os.Args[0], err)
				os.Exit(1)
			}

			slog.SetDefault(slog.New(slog.NewTextHandler(file, nil)))
		} else {
			slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, nil)))
		}
	} else {
		slog.SetDefault(slog.New(discardHandler{}))
	}
}

func fatal(err error) {
	slog.Error(err.Error())
	fmt.Fprintf(os.Stderr, "%s: %s", os.Args[0], err)
	os.Exit(1)
}

func main() {
	var log bool
	var logfile string

	flag.BoolVar(&log, "log", false, "enable logging")
	flag.StringVar(&logfile, "logfile", "", "log file path")
	flag.Parse()

	setupLogging(log, logfile)
	slog.Info("starting", slog.String("cmd", os.Args[0]),
		slog.String("args", strings.Join(os.Args[1:], " ")), slog.Int("pid", os.Getpid()))

	rootDir, err := rootDirectory(flag.Args())
	if err != nil {
		fatal(err)
	}

	root, err := os.OpenRoot(rootDir)
	if err != nil {
		fatal(err)
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
		fatal(err)
	}

	slog.Info("exiting", slog.String("cmd", os.Args[0]),
		slog.String("args", strings.Join(os.Args[1:], " ")), slog.Int("pid", os.Getpid()))
}
