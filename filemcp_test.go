package main

import (
	"testing"
)

func TestSanitizePath(t *testing.T) {
	rootDir := "/home/mike"
	cases := []struct {
		p, r string
		fail bool
	}{
		{p: "file.txt", r: "/home/mike/file.txt"},
		{p: "subdir/file.txt", r: "/home/user/mike/subdir/file.txt"},
		{p: "/etc/passwd", fail: true},
		{p: "../etc/passwd", fail: true},
		{p: "../../etc/passwd", fail: true},
		{p: "../../../etc/passwd", fail: true},
		{p: "subdir/../file.txt", r: "/home/mike/file.txt"},
		{p: "subdir//file.txt", r: "/home/mike/subdir/file.txt"},
		{p: "./file.txt", r: "/home/mike/file.txt"},
		{p: "", r: "/home/mike"},
		{p: ".", r: "/home/mike"},
		{p: "/file.txt", fail: true},
		{p: "subdir/../../file.txt", fail: true},
	}

	for _, c := range cases {
		r, err := sanitizePath(rootDir, c.p)
		if err == nil {
			if c.fail {
				t.Errorf("sanitizePath(%s) did not fail", c.p)
			}
		} else if !c.fail {
			t.Errorf("sanitizePath(%s) failed with %s", c.p, err)
		} else if r != c.r {
			t.Errorf("sanitizePath(%s) got %s want %s", c.p, r, c.r)
		}
	}
}
