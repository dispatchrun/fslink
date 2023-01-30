package fslink_test

import (
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/stealthrocket/fslink"
)

func TestLstat(t *testing.T) {
	fsys := fstest.MapFS{
		"file": &fstest.MapFile{Mode: 0600},
		"link": &fstest.MapFile{Mode: 0666 | fs.ModeSymlink, Data: []byte("file")},
	}

	info, err := fslink.Lstat(fsys, "link")
	if err != nil {
		t.Fatal(err)
	}
	if info.Name() != "link" {
		t.Errorf("wrong link name: %q", info.Name())
	}
	if info.Mode() != (0666 | fs.ModeSymlink) {
		t.Errorf("wrong link mode: %s", info.Mode())
	}
}
