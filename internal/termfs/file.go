package termfs

import (
	"io"
	"io/fs"
	"sort"
	"strings"
	"time"
)

// File represents a file or directory in the terminal filesystem.
type File struct {
	name     string
	isDir    bool
	content  []byte
	modTime  time.Time
	children map[string]*File
}

// openFile implements fs.File and fs.ReadDirFile.
type openFile struct {
	file   *File
	fs     *FS
	path   string
	offset int64
}

// Stat implements fs.File.
func (of *openFile) Stat() (fs.FileInfo, error) {
	return &FileInfo{file: of.file}, nil
}

// Read implements fs.File.
func (of *openFile) Read(b []byte) (int, error) {
	if of.file.isDir {
		return 0, &fs.PathError{Op: "read", Path: of.path, Err: fs.ErrInvalid}
	}
	
	if of.offset >= int64(len(of.file.content)) {
		return 0, io.EOF
	}
	
	n := copy(b, of.file.content[of.offset:])
	of.offset += int64(n)
	return n, nil
}

// Close implements fs.File.
func (of *openFile) Close() error {
	return nil
}

// ReadDir implements fs.ReadDirFile.
func (of *openFile) ReadDir(n int) ([]fs.DirEntry, error) {
	if !of.file.isDir {
		return nil, &fs.PathError{Op: "readdir", Path: of.path, Err: fs.ErrInvalid}
	}

	var entries []fs.DirEntry

	// Find all files that are children of this directory
	for filePath, file := range of.fs.files {
		if of.path == "" {
			// Root directory - include files directly in root
			if !strings.Contains(filePath, "/") && filePath != "" {
				entries = append(entries, &dirEntry{file: file})
			}
		} else {
			// Check if this file is a direct child of the current directory
			if after, ok := strings.CutPrefix(filePath, of.path+"/"); ok {
				remaining := after
				if !strings.Contains(remaining, "/") {
					entries = append(entries, &dirEntry{file: file})
				}
			}
		}
	}

	sort.Slice(entries, func(i, j int) bool {
		return strings.Compare(entries[i].Name(), entries[j].Name()) < 0
	})

	if n > 0 && len(entries) > n {
		entries = entries[:n]
	}

	return entries, nil
}

// dirEntry implements fs.DirEntry.
type dirEntry struct {
	file *File
}

// Name implements fs.DirEntry.
func (de *dirEntry) Name() string {
	return de.file.name
}

// IsDir implements fs.DirEntry.
func (de *dirEntry) IsDir() bool {
	return de.file.isDir
}

// Type implements fs.DirEntry.
func (de *dirEntry) Type() fs.FileMode {
	if de.file.isDir {
		return fs.ModeDir
	}
	return 0
}

// Info implements fs.DirEntry.
func (de *dirEntry) Info() (fs.FileInfo, error) {
	return &FileInfo{file: de.file}, nil
}

