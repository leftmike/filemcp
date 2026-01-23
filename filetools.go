package main

import (
	"context"
	"io"
	"io/fs"
	"log/slog"
	"path/filepath"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type fileTools struct {
	fs fs.FS
}

type readFileInput struct {
	Path string `json:"path" jsonschema:"path to the file relative to root directory"`
}

type readFileOutput struct {
	Content string `json:"content" jsonschema:"the file contents"`
	Size    int    `json:"size" jsonschema:"size of the file in bytes"`
	Path    string `json:"path" jsonschema:"the path that was read"`
}

func (ft fileTools) handleReadFile(ctx context.Context, req *mcp.CallToolRequest,
	args readFileInput) (*mcp.CallToolResult, readFileOutput, error) {

	slog.Info("read file", slog.Any("req", req), slog.Any("args", args))

	cnt, err := ft.readFile(ctx, args.Path)
	if err != nil {
		return nil, readFileOutput{}, err
	}

	return nil, readFileOutput{
		Content: string(cnt),
		Size:    len(cnt),
		Path:    args.Path,
	}, nil
}

func (ft fileTools) readFile(ctx context.Context, path string) ([]byte, error) {
	fh, err := ft.fs.Open(path)
	if err != nil {
		return nil, err
	}
	defer fh.Close()

	cnt, err := io.ReadAll(fh)
	if err != nil {
		return nil, err
	}
	return cnt, nil
}

type listDirectoryInput struct {
	Path string `json:"path,omitempty" jsonschema:"path to directory relative to root (empty for root)"`
}

type directoryEntry struct {
	Name  string `json:"name" jsonschema:"name of the file or directory"`
	Size  int64  `json:"size" jsonschema:"size in bytes (0 for directories)"`
	IsDir bool   `json:"isDir" jsonschema:"true if this is a directory"`
}

type listDirectoryOutput struct {
	Path    string           `json:"path" jsonschema:"the directory path that was listed"`
	Entries []directoryEntry `json:"entries" jsonschema:"list of directory entries"`
	Count   int              `json:"count" jsonschema:"number of entries"`
}

func (ft fileTools) handleListDirectory(ctx context.Context, req *mcp.CallToolRequest,
	args listDirectoryInput) (*mcp.CallToolResult, listDirectoryOutput, error) {

	slog.Info("list directory", slog.Any("req", req), slog.Any("args", args))

	entries, err := ft.listDirectory(ctx, args.Path)
	if err != nil {
		return nil, listDirectoryOutput{}, err
	}

	return nil, listDirectoryOutput{
		Path:    args.Path,
		Entries: entries,
		Count:   len(entries),
	}, nil
}

func (ft fileTools) listDirectory(ctx context.Context, path string) ([]directoryEntry, error) {
	if path == "" {
		path = "."
	}
	lst, err := fs.ReadDir(ft.fs, path)
	if err != nil {
		return nil, err
	}

	var entries []directoryEntry
	for _, de := range lst {
		var sz int64
		if !de.IsDir() {
			if fi, err := de.Info(); err == nil {
				sz = fi.Size()
			}
		}

		entries = append(entries, directoryEntry{
			Name:  de.Name(),
			Size:  sz,
			IsDir: de.IsDir(),
		})
	}

	return entries, nil
}

type searchFilesInput struct {
	Pattern string `json:"pattern" jsonschema:"glob pattern to match files, e.g. '*.txt'"`
}

type searchFilesOutput struct {
	Pattern string   `json:"pattern" jsonschema:"the pattern that was searched"`
	Matches []string `json:"matches" jsonschema:"list of matching file paths"`
	Count   int      `json:"count" jsonschema:"number of matches found"`
}

func (ft fileTools) handleSearchFiles(ctx context.Context, req *mcp.CallToolRequest,
	args searchFilesInput) (*mcp.CallToolResult, searchFilesOutput, error) {

	slog.Info("search files", slog.Any("req", req), slog.Any("args", args))

	matches, err := ft.searchFiles(ctx, args.Pattern)
	if err != nil {
		return nil, searchFilesOutput{}, err
	}

	return nil, searchFilesOutput{
		Pattern: args.Pattern,
		Matches: matches,
		Count:   len(matches),
	}, nil
}

func (ft fileTools) searchFiles(ctx context.Context, pattern string) ([]string, error) {
	var matches []string
	err := fs.WalkDir(ft.fs, ".", func(path string, de fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if de.IsDir() {
			return nil
		}

		matched, err := filepath.Match(pattern, de.Name())
		if err != nil {
			return err
		}

		if matched {
			matches = append(matches, path)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}
	return matches, nil
}

type getFileInfoInput struct {
	Path string `json:"path" jsonschema:"path to the file relative to root directory"`
}

type getFileInfoOutput struct {
	Path    string `json:"path" jsonschema:"the file path"`
	Size    int64  `json:"size" jsonschema:"size in bytes"`
	IsDir   bool   `json:"isDir" jsonschema:"true if this is a directory"`
	ModTime string `json:"modTime" jsonschema:"last modification time"`
	Mode    string `json:"mode" jsonschema:"file permissions"`
}

func (ft fileTools) handleGetFileInfo(ctx context.Context, req *mcp.CallToolRequest,
	args getFileInfoInput) (*mcp.CallToolResult, getFileInfoOutput, error) {

	slog.Info("get file info", slog.Any("req", req), slog.Any("args", args))

	fi, err := ft.getFileInfo(ctx, args.Path)
	if err != nil {
		return nil, getFileInfoOutput{}, err
	}
	return nil, getFileInfoOutput{
		Path:    args.Path,
		Size:    fi.Size(),
		IsDir:   fi.IsDir(),
		ModTime: fi.ModTime().Format("2006-01-02T15:04:05Z07:00"),
		Mode:    fi.Mode().String(),
	}, nil
}

func (ft fileTools) getFileInfo(ctx context.Context, path string) (fs.FileInfo, error) {
	return fs.Stat(ft.fs, path)
}

func (ft fileTools) registerTools(srvr *mcp.Server) {
	mcp.AddTool(srvr, &mcp.Tool{
		Name:        "read_file",
		Description: "Read the contents of a file. Returns the file content as text.",
	}, ft.handleReadFile)

	mcp.AddTool(srvr, &mcp.Tool{
		Name:        "list_directory",
		Description: "List the contents of a directory. Returns file names, types, and sizes.",
	}, ft.handleListDirectory)

	mcp.AddTool(srvr, &mcp.Tool{
		Name:        "search_files",
		Description: "Search for files matching a glob pattern (e.g., '*.go', 'test*', '*.md').",
	}, ft.handleSearchFiles)

	mcp.AddTool(srvr, &mcp.Tool{
		Name:        "get_file_info",
		Description: "Get detailed information about a file or directory.",
	}, ft.handleGetFileInfo)
}
