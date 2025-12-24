package termfs

import (
	"io/fs"
	"time"
)

// FileInfo implements fs.FileInfo for File.
type FileInfo struct {
	file *File
}

// Name implements fs.FileInfo.
func (fi *FileInfo) Name() string {
	return fi.file.name
}

// Size implements fs.FileInfo.
func (fi *FileInfo) Size() int64 {
	return int64(len(fi.file.content))
}

// Mode implements fs.FileInfo.
func (fi *FileInfo) Mode() fs.FileMode {
	if fi.file.isDir {
		return fs.ModeDir | 0o755
	}
	return 0o644
}

// ModTime implements fs.FileInfo.
func (fi *FileInfo) ModTime() time.Time {
	return fi.file.modTime
}

// IsDir implements fs.FileInfo.
func (fi *FileInfo) IsDir() bool {
	return fi.file.isDir
}

// Sys implements fs.FileInfo.
func (fi *FileInfo) Sys() any {
	return nil
}