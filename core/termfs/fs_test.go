package termfs

import (
	"errors"
	"io"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zorcal/its-a-me-zorcal/pkg/github"
)

func testRepos() []github.Repository {
	return []github.Repository{
		{
			Name:        "test-repo",
			Description: "A test repository",
			Language:    "Go",
			Stars:       42,
			URL:         "https://github.com/test/test-repo",
			UpdatedAt:   "2024-01-01T00:00:00Z",
		},
		{
			Name:        "another-repo",
			Description: "Another test repository",
			Language:    "JavaScript",
			Stars:       100,
			URL:         "https://github.com/test/another-repo",
			UpdatedAt:   "2024-01-02T00:00:00Z",
		},
	}
}

func TestNew(t *testing.T) {
	repos := testRepos()
	tfs := New(repos)

	if tfs == nil {
		t.Fatal("New() = nil, want non-nil filesystem")
	}

	t.Run("basic directories exist", func(t *testing.T) {
		paths := []string{
			".",
			"home",
			"home/zorcal",
			"home/guest",
			"home/zorcal/projects",
		}
		for _, p := range paths {
			f, err := tfs.Open(p)
			if err != nil {
				t.Fatalf("Open(%q) failed: %v", p, err)
			}
			defer f.Close()

			info, err := f.Stat()
			if err != nil {
				t.Fatalf("Stat(%q) failed: %v", p, err)
			}

			if !info.IsDir() {
				t.Errorf("IsDir() = false for %q, want true", p)
			}
		}
	})

	t.Run("repository files created", func(t *testing.T) {
		paths := []string{
			"home/zorcal/projects/test-repo.md",
			"home/zorcal/projects/another-repo.md",
		}
		for _, p := range paths {
			f, err := tfs.Open(p)
			if err != nil {
				t.Fatalf("Open(%q) failed: %v", p, err)
			}
			defer f.Close()

			info, err := f.Stat()
			if err != nil {
				t.Fatalf("Stat(%q) failed: %v", p, err)
			}

			if info.IsDir() {
				t.Errorf("IsDir() = true for %q, want false", p)
			}

			if info.Size() == 0 {
				t.Errorf("Size() = 0 for %q, want > 0", p)
			}
		}
	})
}

func TestFS_open(t *testing.T) {
	tfs := New([]github.Repository{})

	t.Run("existing paths", func(t *testing.T) {
		tests := []struct {
			path  string
			isDir bool
		}{
			{".", true},
			{"home", true},
			{"home/zorcal", true},
			{"home/guest", true},
			{"home/zorcal/projects", true},
		}
		for _, tt := range tests {
			t.Run(tt.path, func(t *testing.T) {
				f, err := tfs.Open(tt.path)
				if err != nil {
					t.Fatalf("Open(%q) failed: %v", tt.path, err)
				}
				defer f.Close()

				info, err := f.Stat()
				if err != nil {
					t.Fatalf("Stat() failed: %v", err)
				}

				if got := info.IsDir(); got != tt.isDir {
					t.Errorf("IsDir() = %v, want %v", got, tt.isDir)
				}
			})
		}
	})

	t.Run("nonexistent paths", func(t *testing.T) {
		paths := []string{
			"nonexistent",
			"home/nonexistent",
			"../invalid",
		}
		for _, path := range paths {
			t.Run(path, func(t *testing.T) {
				f, err := tfs.Open(path)
				if err == nil {
					t.Errorf("Open(%q) = nil, want error", path)
					if f != nil {
						f.Close()
					}
				}
			})
		}
	})
}

func TestFS_readDir(t *testing.T) {
	repos := []github.Repository{
		{Name: "repo1", Description: "First repo", Language: "Go", Stars: 10, URL: "url1", UpdatedAt: "2024-01-01T00:00:00Z"},
		{Name: "repo2", Description: "Second repo", Language: "JS", Stars: 20, URL: "url2", UpdatedAt: "2024-01-02T00:00:00Z"},
	}

	tfs := New(repos)

	tests := []struct {
		name      string
		path      string
		wantFiles []string
	}{
		{
			name:      "root directory",
			path:      ".",
			wantFiles: []string{"home"},
		},
		{
			name:      "home directory",
			path:      "home",
			wantFiles: []string{"guest", "zorcal"},
		},
		{
			name:      "zorcal directory",
			path:      "home/zorcal",
			wantFiles: []string{".secret.txt", "projects"},
		},
		{
			name:      "projects directory",
			path:      "home/zorcal/projects",
			wantFiles: []string{"repo1.md", "repo2.md"},
		},
		{
			name:      "empty directory",
			path:      "home/guest",
			wantFiles: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := tfs.Open(tt.path)
			if err != nil {
				t.Fatalf("Open(%q) failed: %v", tt.path, err)
			}
			defer f.Close()

			rdf, ok := f.(fs.ReadDirFile)
			if !ok {
				t.Fatalf("Open(%q) does not implement ReadDirFile", tt.path)
			}

			entries, err := rdf.ReadDir(-1)
			if err != nil {
				t.Fatalf("ReadDir() failed: %v", err)
			}

			if got, want := len(entries), len(tt.wantFiles); got != want {
				t.Fatalf("len(entries) = %d, want %d", got, want)
			}

			for i, entry := range entries {
				if got, want := entry.Name(), tt.wantFiles[i]; got != want {
					t.Errorf("entries[%d].Name() = %q, want %q", i, got, want)
				}
			}
		})
	}
}

func TestFS_readFile(t *testing.T) {
	repos := []github.Repository{
		{
			Name:        "test-repo",
			Description: "A test repository",
			Language:    "Go",
			Stars:       42,
			URL:         "https://github.com/test/test-repo",
			UpdatedAt:   "2024-01-01T00:00:00Z",
		},
	}

	tfs := New(repos)

	f, err := tfs.Open("home/zorcal/projects/test-repo.md")
	if err != nil {
		t.Fatalf("Open() failed: %v", err)
	}
	defer f.Close()

	content := make([]byte, 1024)
	n, err := f.Read(content)
	if err != nil && !errors.Is(err, io.EOF) {
		t.Fatalf("Read() failed: %v", err)
	}

	contentStr := string(content[:n])

	wantContents := []string{
		"# test-repo",
		"A test repository",
		"**Language:** Go",
		"**Stars:** 42",
		"**URL:** https://github.com/test/test-repo",
		"**Last Updated:** 2024-01-01T00:00:00Z",
	}
	for _, want := range wantContents {
		if !strings.Contains(contentStr, want) {
			t.Errorf("content missing %q", want)
		}
	}
}

func TestFS_addOperations(t *testing.T) {
	tfs := New([]github.Repository{})

	t.Run("add directory", func(t *testing.T) {
		tfs.AddDir("test/dir")

		f, err := tfs.Open("test/dir")
		if err != nil {
			t.Fatalf("Open() failed: %v", err)
		}
		defer f.Close()

		info, err := f.Stat()
		if err != nil {
			t.Fatalf("Stat() failed: %v", err)
		}

		if !info.IsDir() {
			t.Error("IsDir() = false, want true")
		}

		if got, want := info.Name(), "dir"; got != want {
			t.Errorf("Name() = %q, want %q", got, want)
		}
	})

	t.Run("add file", func(t *testing.T) {
		tfs.AddDir("test")

		content := []byte("Hello, World!")
		tfs.AddFile("test/hello.txt", content)

		f, err := tfs.Open("test/hello.txt")
		if err != nil {
			t.Fatalf("Open() failed: %v", err)
		}
		defer f.Close()

		info, err := f.Stat()
		if err != nil {
			t.Fatalf("Stat() failed: %v", err)
		}

		if info.IsDir() {
			t.Error("IsDir() = true, want false")
		}

		if got, want := info.Name(), "hello.txt"; got != want {
			t.Errorf("Name() = %q, want %q", got, want)
		}

		if got, want := info.Size(), int64(len(content)); got != want {
			t.Errorf("Size() = %d, want %d", got, want)
		}

		readContent := make([]byte, len(content))
		n, err := f.Read(readContent)
		if err != nil {
			t.Fatalf("Read() failed: %v", err)
		}

		if n != len(content) {
			t.Errorf("Read() = %d bytes, want %d", n, len(content))
		}

		if got, want := string(readContent), string(content); got != want {
			t.Errorf("content = %q, want %q", got, want)
		}
	})
}

func TestFS_walkDir(t *testing.T) {
	repos := []github.Repository{
		{Name: "repo1", Description: "First repo", Language: "Go", Stars: 10, URL: "url1", UpdatedAt: "2024-01-01T00:00:00Z"},
	}

	tfs := New(repos)

	var paths []string
	err := fs.WalkDir(tfs, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		paths = append(paths, path)
		return nil
	})
	if err != nil {
		t.Fatalf("WalkDir() failed: %v", err)
	}

	wantPaths := []string{
		".",
		"home",
		"home/guest",
		"home/zorcal",
		"home/zorcal/.secret.txt",
		"home/zorcal/projects",
		"home/zorcal/projects/repo1.md",
	}

	if got, want := len(paths), len(wantPaths); got != want {
		t.Errorf("walked %d paths, want %d. got: %v", got, want, paths)
	}

	pathSet := make(map[string]bool)
	for _, p := range paths {
		pathSet[filepath.ToSlash(p)] = true
	}

	for _, want := range wantPaths {
		if !pathSet[want] {
			t.Errorf("missing path %q in walked paths", want)
		}
	}
}

func TestFileInfo(t *testing.T) {
	tfs := New([]github.Repository{})
	tfs.AddFile("test.txt", []byte("hello"))

	t.Run("file info", func(t *testing.T) {
		f, err := tfs.Open("test.txt")
		if err != nil {
			t.Fatalf("Open() failed: %v", err)
		}
		defer f.Close()

		info, err := f.Stat()
		if err != nil {
			t.Fatalf("Stat() failed: %v", err)
		}

		if got, want := info.Name(), "test.txt"; got != want {
			t.Errorf("Name() = %q, want %q", got, want)
		}

		if got, want := info.Size(), int64(5); got != want {
			t.Errorf("Size() = %d, want %d", got, want)
		}

		if info.IsDir() {
			t.Error("IsDir() = true, want false for file")
		}

		if info.Mode()&fs.ModeDir != 0 {
			t.Error("Mode() has directory bit set for file")
		}

		if info.Sys() != nil {
			t.Error("Sys() = non-nil, want nil")
		}
	})

	t.Run("directory info", func(t *testing.T) {
		f, err := tfs.Open("home")
		if err != nil {
			t.Fatalf("Open() failed: %v", err)
		}
		defer f.Close()

		info, err := f.Stat()
		if err != nil {
			t.Fatalf("Stat() failed: %v", err)
		}

		if !info.IsDir() {
			t.Error("IsDir() = false, want true for directory")
		}

		if info.Mode()&fs.ModeDir == 0 {
			t.Error("Mode() missing directory bit for directory")
		}
	})
}
