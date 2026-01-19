package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
)

func mustWriteFile(t *testing.T, path string, cnt []byte) {
	t.Helper()

	err := os.MkdirAll(filepath.Dir(path), 0755)
	if err != nil {
		t.Fatalf("MkdirAll(%s) failed with %s", filepath.Dir(path), err)
	}
	err = os.WriteFile(path, cnt, 0644)
	if err != nil {
		t.Fatalf("WriteFile(%s) failed with %s", path, err)
	}
}

func TestReadFile(t *testing.T) {
	tempDir := t.TempDir()

	mustWriteFile(t, filepath.Join(tempDir, "simple.txt"), []byte("simple content"))
	mustWriteFile(t, filepath.Join(tempDir, "empty.txt"), []byte{})
	mustWriteFile(t, filepath.Join(tempDir, "subdir", "nested.txt"), []byte("nested content"))
	mustWriteFile(t, filepath.Join(tempDir, "special.txt"),
		[]byte("line1\nline2\ttab\r\nwindows"))

	largeContent := make([]byte, 1024*1024)
	for i := range largeContent {
		largeContent[i] = byte(i % 256)
	}
	mustWriteFile(t, filepath.Join(tempDir, "large.bin"), largeContent)

	cases := []struct {
		path string
		cnt  []byte
		fail bool
	}{
		{path: "simple.txt", cnt: []byte("simple content")},
		{path: "empty.txt", cnt: []byte{}},
		{path: "subdir/nested.txt", cnt: []byte("nested content")},
		{path: "special.txt", cnt: []byte("line1\nline2\ttab\r\nwindows")},
		{path: "large.bin", cnt: largeContent},
		{path: "missing.txt", fail: true},
		{path: "nodir/file.txt", fail: true},
	}

	ft := fileTools{fs: os.DirFS(tempDir)}
	ctx := context.Background()

	for _, c := range cases {
		cnt, err := ft.readFile(ctx, c.path)
		if err != nil {
			if !c.fail {
				t.Errorf("readFile(%s) failed with %s", c.path, err)
			}
		} else if c.fail {
			t.Errorf("readFile(%s) did not fail", c.path)
		} else if !bytes.Equal(cnt, c.cnt) {
			t.Errorf("readFile(%s) got %v want %v", c.path, cnt, c.cnt)
		}
	}
}

func TestReadFileEscape(t *testing.T) {
	tempDir := t.TempDir()

	mustWriteFile(t, filepath.Join(tempDir, "inside.txt"), []byte("inside"))

	outsideFile := filepath.Join(filepath.Dir(tempDir), "outside.txt")
	mustWriteFile(t, outsideFile, []byte("outside"))
	defer os.Remove(outsideFile)

	ft := fileTools{fs: os.DirFS(tempDir)}
	ctx := context.Background()

	cnt, err := ft.readFile(ctx, "inside.txt")
	if err != nil {
		t.Errorf("readFile(inside.txt) failed with %s", err)
	} else if string(cnt) != "inside" {
		t.Errorf("readFile(inside.txt) got %s, want inside", cnt)
	}

	mustFailPaths := []string{
		"../outside.txt",
		"../../outside.txt",
		"../../../etc/passwd",
		"/etc/passwd",
		"subdir/../../outside.txt",
		"./../../outside.txt",
		"..\\outside.txt",
		"./../outside.txt",
	}

	for _, path := range mustFailPaths {
		_, err := ft.readFile(ctx, path)
		if err == nil {
			t.Errorf("readFile(%s) did not fail", path)
		}
	}
}

func TestListDirectory(t *testing.T) {
	tempDir := t.TempDir()

	mustWriteFile(t, filepath.Join(tempDir, "file1.txt"), []byte("content1"))
	mustWriteFile(t, filepath.Join(tempDir, "file2.txt"), []byte("content2"))
	mustWriteFile(t, filepath.Join(tempDir, "subdir", "nested.txt"), []byte("nested"))
	mustWriteFile(t, filepath.Join(tempDir, "subdir", "another.txt"), []byte("another"))
	mustWriteFile(t, filepath.Join(tempDir, "empty", ".gitkeep"), []byte{})

	cases := []struct {
		path    string
		entries []directoryEntry
		fail    bool
	}{
		{
			path: ".",
			entries: []directoryEntry{
				{Name: "empty", IsDir: true},
				{Name: "file1.txt", Size: 8},
				{Name: "file2.txt", Size: 8},
				{Name: "subdir", IsDir: true},
			},
		},
		{
			path: "",
			entries: []directoryEntry{
				{Name: "empty", IsDir: true},
				{Name: "file1.txt", Size: 8},
				{Name: "file2.txt", Size: 8},
				{Name: "subdir", IsDir: true},
			},
		},
		{
			path: "subdir",
			entries: []directoryEntry{
				{Name: "another.txt", Size: 7},
				{Name: "nested.txt", Size: 6},
			},
		},
		{
			path: "empty",
			entries: []directoryEntry{
				{Name: ".gitkeep"},
			},
		},
		{path: "nonexistent", fail: true},
		{path: "file1.txt", fail: true},
	}

	ft := fileTools{fs: os.DirFS(tempDir)}
	ctx := context.Background()

	for _, c := range cases {
		entries, err := ft.listDirectory(ctx, c.path)
		if err != nil {
			if !c.fail {
				t.Errorf("listDirectory(%s) failed with %s", c.path, err)
			}
		} else if c.fail {
			t.Errorf("listDirectory(%s) did not fail", c.path)
		} else {
			slices.SortFunc(entries, func(a, b directoryEntry) int {
				return strings.Compare(a.Name, b.Name)
			})

			if !reflect.DeepEqual(entries, c.entries) {
				t.Errorf("listDirectory(%s) got %v, want %v", c.path, entries, c.entries)
			}
		}
	}
}

func TestListDirectoryEscape(t *testing.T) {
	tempDir := t.TempDir()

	mustWriteFile(t, filepath.Join(tempDir, "inside", "file.txt"), []byte("inside"))

	outsideDir := filepath.Join(filepath.Dir(tempDir), "outsidedir")
	err := os.Mkdir(outsideDir, 0755)
	if err != nil {
		t.Fatalf("Mkdir(%s) failed with %s", outsideDir, err)
	}
	defer os.RemoveAll(outsideDir)

	ft := fileTools{fs: os.DirFS(tempDir)}
	ctx := context.Background()

	entries, err := ft.listDirectory(ctx, "inside")
	if err != nil {
		t.Errorf("listDirectory(inside) failed with %s", err)
	} else if len(entries) != 1 || entries[0].Name != "file.txt" {
		t.Errorf("listDirectory(inside) got unexpected entries: %v", entries)
	}

	mustFailPaths := []string{
		"..",
		"../..",
		"../../../etc",
		"/etc",
		"inside/../..",
		"./../../",
	}

	for _, path := range mustFailPaths {
		_, err := ft.listDirectory(ctx, path)
		if err == nil {
			t.Errorf("listDirectory(%s) did not fail", path)
		}
	}
}
