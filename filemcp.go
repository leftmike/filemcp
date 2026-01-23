/*
To Do:
- Authenticate with a shared secret
*/

package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
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

func setupTransport(logProto string, t mcp.Transport) mcp.Transport {
	if logProto != "" {
		file, err := os.OpenFile(logProto, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			slog.Error("open file", slog.String("logproto", logProto),
				slog.String("error", err.Error()))
			return t
		}

		return &mcp.LoggingTransport{
			Transport: t,
			Writer:    file,
		}
	}

	return t
}

func fatal(err error) {
	slog.Error(err.Error())
	fmt.Fprintf(os.Stderr, "%s: %s", os.Args[0], err)
	os.Exit(1)
}

func main() {
	var log bool
	var logfile string
	var logProto string
	var useStdio bool
	var useSSE bool
	var useHTTP bool
	var httpsAddr string
	var tlsCert string
	var tlsKey string

	flag.BoolVar(&log, "log", false, "enable logging")
	flag.StringVar(&logfile, "logfile", "", "log file path")
	flag.StringVar(&logProto, "logproto", "", "protocol log file path")
	flag.BoolVar(&useStdio, "stdio", false, "use stdio transport")
	flag.BoolVar(&useSSE, "sse", false, "use SSE transport at /sse (requires -cert and -key)")
	flag.BoolVar(&useHTTP, "http", false, "use streaming HTTP transport at /mcp (requires -cert and -key)")
	flag.StringVar(&httpsAddr, "addr", ":8443", "HTTPS server address")
	flag.StringVar(&tlsCert, "cert", "", "TLS certificate file (required for -sse or -http)")
	flag.StringVar(&tlsKey, "key", "", "TLS key file (required for -sse or -http)")
	flag.Parse()

	if !useStdio && !useSSE && !useHTTP {
		fmt.Fprintf(os.Stderr, "at least one of -stdio, -sse, or -http must be specified\n")
		flag.Usage()
		os.Exit(1)
	}

	if (useSSE || useHTTP) && (tlsCert == "" || tlsKey == "") {
		fmt.Fprintf(os.Stderr, "-cert and -key are required for -sse or -http transport\n")
		flag.Usage()
		os.Exit(1)
	}

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

	useHTTPS := useSSE || useHTTP
	numTransports := 0
	if useHTTPS {
		numTransports++
	}
	if useStdio {
		numTransports++
	}

	errChan := make(chan error, numTransports)

	if useHTTPS {
		mux := http.NewServeMux()
		if useSSE {
			slog.Info("adding SSE handler", slog.String("path", "/sse"))
			mux.Handle("/sse", mcp.NewSSEHandler(func(r *http.Request) *mcp.Server {
				return srvr
			}, nil))
		}
		if useHTTP {
			slog.Info("adding streaming HTTP handler", slog.String("path", "/mcp"))
			mux.Handle("/mcp", mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
				return srvr
			}, nil))
		}

		go func() {
			slog.Info("starting HTTPS server", slog.String("addr", httpsAddr))
			errChan <- http.ListenAndServeTLS(httpsAddr, tlsCert, tlsKey, mux)
		}()
	}

	if useStdio {
		go func() {
			slog.Info("starting stdio transport")
			errChan <- srvr.Run(ctx, setupTransport(logProto, &mcp.StdioTransport{}))
		}()
	}

	// Wait for any transport to fail
	err = <-errChan

	if err != nil {
		fatal(err)
	}

	slog.Info("exiting", slog.String("cmd", os.Args[0]),
		slog.String("args", strings.Join(os.Args[1:], " ")), slog.Int("pid", os.Getpid()))
}
