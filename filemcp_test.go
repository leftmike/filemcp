package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRootDirectory(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() failed with %s", err)
	}

	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "file.txt")
	err = os.WriteFile(tempFile, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("WriteFile(%s) failed with %s", tempFile, err)
	}
	tempSubdir := filepath.Join(tempDir, "subdir")
	err = os.Mkdir(tempSubdir, 0755)
	if err != nil {
		t.Fatalf("Mkdir(%s) failed with %s", tempSubdir, err)
	}
	tempSymlink := filepath.Join(tempDir, "symlink")
	err = os.Symlink(tempSubdir, tempSymlink)
	if err != nil {
		t.Fatalf("Symlink(%s) failed with %s", tempSymlink, err)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("UserHomeDir() failed with %s", err)
	}

	cases := []struct {
		args []string
		r    string
		fail bool
	}{
		{args: []string{}, r: cwd},
		{args: []string{"."}, r: cwd},
		{args: []string{tempDir}, r: tempDir},
		{args: []string{"arg1", "arg2"}, fail: true},
		{args: []string{"arg1", "arg2", "arg3"}, fail: true},
		{args: []string{"/bad/dog/food"}, fail: true},
		{args: []string{tempFile}, fail: true},
		{args: []string{"/"}, r: "/"},
		{args: []string{homeDir}, r: homeDir},
		{args: []string{"/", homeDir}, fail: true},
		{args: []string{homeDir, "/"}, fail: true},
		{args: []string{tempSymlink}, r: tempSymlink},
		{args: []string{tempSubdir}, r: tempSubdir},
		{args: []string{"./."}, r: cwd},
	}

	for _, c := range cases {
		r, err := rootDirectory(c.args)
		if err != nil {
			if !c.fail {
				t.Errorf("rootDirectory(%s) failed with %s", c.args, err)
			}
		} else if c.fail {
			t.Errorf("rootDirectory(%s) did not fail", c.args)
		} else if r != c.r {
			t.Errorf("rootDirectory(%s) got %s want %s", c.args, r, c.r)
		}
	}
}
