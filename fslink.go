// =============================================================================
// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
// =============================================================================

// Package fslink is an implementation of the proposal to add a fs.ReadLinkFS
// interface: https://github.com/golang/go/issues/49580
//
// This package is intended to be deprecated once #49580 is merged.
package fslink

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"strings"
)

// ReadLinkFS is an extension of the fs.FS interface implemented by file systems
// which support symbolic links.
type ReadLinkFS interface {
	fs.FS
	ReadLink(string) (string, error)
}

// ReadLink reads the value of the symbolic link at the given name in fsys.
func ReadLink(fsys fs.FS, name string) (string, error) {
	f, ok := fsys.(ReadLinkFS)
	if !ok {
		err := fmt.Errorf("symlink found in file system which does not implement fs.ReadLinkFS: %T (%w)", fsys, fs.ErrInvalid)
		return "", &fs.PathError{"readlink", name, err}
	}
	link, err := f.ReadLink(name)
	if err != nil {
		return "", err
	}
	// Note: the current proposal from #49580 states that the ReadLink method
	// should error if the link being read is absolute; we enforce it here to
	// the best we can.
	switch {
	case link == "..":
	case strings.HasPrefix(link, "../"):
	case fs.ValidPath(link):
	default:
		return "", &fs.PathError{"readlink", name, fmt.Errorf("malformed link target: %q (%w)", link, fs.ErrInvalid)}
	}
	return link, nil
}

// TODO: the code below is copied from the Go standard library to add the
// ReadLink method to subFS. We should remove it when ReadLinkFS has been added.

func (f *subFS) ReadLink(name string) (string, error) {
	full, err := f.fullName("readlink", name)
	if err != nil {
		return "", err
	}
	return ReadLink(f.fsys, full)
}

var (
	_ ReadLinkFS = (*subFS)(nil)
)

// Sub is like fs.Sub but it returns a type that implements ReadLinkFS.
func Sub(fsys fs.FS, dir string) (fs.FS, error) {
	if !fs.ValidPath(dir) {
		return nil, &fs.PathError{Op: "sub", Path: dir, Err: errors.New("invalid name")}
	}
	if dir == "." {
		return fsys, nil
	}
	if fsys, ok := fsys.(fs.SubFS); ok {
		return fsys.Sub(dir)
	}
	return &subFS{fsys, dir}, nil
}

type subFS struct {
	fsys fs.FS
	dir  string
}

// fullName maps name to the fully-qualified name dir/name.
func (f *subFS) fullName(op string, name string) (string, error) {
	if !fs.ValidPath(name) {
		return "", &fs.PathError{Op: op, Path: name, Err: errors.New("invalid name")}
	}
	return path.Join(f.dir, name), nil
}

// shorten maps name, which should start with f.dir, back to the suffix after f.dir.
func (f *subFS) shorten(name string) (rel string, ok bool) {
	if name == f.dir {
		return ".", true
	}
	if len(name) >= len(f.dir)+2 && name[len(f.dir)] == '/' && name[:len(f.dir)] == f.dir {
		return name[len(f.dir)+1:], true
	}
	return "", false
}

// fixErr shortens any reported names in PathErrors by stripping f.dir.
func (f *subFS) fixErr(err error) error {
	if e, ok := err.(*fs.PathError); ok {
		if short, ok := f.shorten(e.Path); ok {
			e.Path = short
		}
	}
	return err
}

func (f *subFS) Open(name string) (fs.File, error) {
	full, err := f.fullName("open", name)
	if err != nil {
		return nil, err
	}
	file, err := f.fsys.Open(full)
	return file, f.fixErr(err)
}

func (f *subFS) ReadDir(name string) ([]fs.DirEntry, error) {
	full, err := f.fullName("read", name)
	if err != nil {
		return nil, err
	}
	dir, err := fs.ReadDir(f.fsys, full)
	return dir, f.fixErr(err)
}

func (f *subFS) ReadFile(name string) ([]byte, error) {
	full, err := f.fullName("read", name)
	if err != nil {
		return nil, err
	}
	data, err := fs.ReadFile(f.fsys, full)
	return data, f.fixErr(err)
}

func (f *subFS) Sub(dir string) (fs.FS, error) {
	if dir == "." {
		return f, nil
	}
	full, err := f.fullName("sub", dir)
	if err != nil {
		return nil, err
	}
	return &subFS{f.fsys, full}, nil
}
