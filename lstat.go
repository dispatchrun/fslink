package fslink

import (
	"errors"
	"io"
	"io/fs"
	"path"
)

// Lstat is like fs.Stat but if the name is a symbolic link, it returns
// information about the link instead of the target file.
//
// At this time, the implementation emulates the behavior by scanning the parent
// directory looking for an entry with a matching name since this is the only
// way to get file information of a symbolic link using a fs.FS file system.
func Lstat(fsys fs.FS, name string) (fs.FileInfo, error) {
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{"lstat", name, fs.ErrNotExist}
	}
	if name == "." {
		return fs.Stat(fsys, name)
	}
	f, err := fsys.Open(path.Dir(name))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	d, ok := f.(fs.ReadDirFile)
	if !ok {
		return nil, &fs.PathError{"lstat", name, fs.ErrInvalid}
	}
	name = path.Base(name)
	for {
		entries, err := d.ReadDir(100)
		for _, entry := range entries {
			if entry.Name() == name {
				return entry.Info()
			}
		}
		if err == io.EOF {
			return nil, &fs.PathError{"lstat", name, fs.ErrNotExist}
		}
		if err != nil {
			return nil, &fs.PathError{"lstat", name, errors.Unwrap(err)}
		}
	}
}
