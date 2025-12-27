package termfs

import (
	"fmt"
	"io/fs"
	"path"
	"time"

	"github.com/zorcal/its-a-me-zorcal/pkg/github"
)

// FS implements fs.FS, providing an in-memory filesystem.
type FS struct {
	files map[string]*File
}

// New creates a new filesystem with basic directories and repository files.
func New(repos []github.Repository) *FS {
	fs := &FS{
		files: make(map[string]*File),
	}

	setupFS(fs, repos)

	return fs
}

func setupFS(fs *FS, repos []github.Repository) {
	fs.AddDir("") // root dir
	fs.AddDir("home")
	fs.AddDir("home/zorcal")
	fs.AddDir("home/guest")
	fs.AddDir("home/zorcal/projects")

	for _, repo := range repos {
		content := fmt.Sprintf(`# %s

%s

**Language:** %s
**Stars:** %d
**URL:** %s
**Last Updated:** %s
`, repo.Name, repo.Description, repo.Language, repo.Stars, repo.URL, repo.UpdatedAt)

		fs.AddFile(fmt.Sprintf("home/zorcal/projects/%s.md", repo.Name), []byte(content))
	}

	welcomeMessage := `Welcome to Zorcal's Terminal Interface!

This is a web-based terminal emulator that simulates a Unix-like environment.
You can navigate the filesystem, view files, and execute commands just like
in a real terminal.

Try exploring with 'ls' and 'cd projects' to see my work!
Run 'help' for a full list of available commands.

Happy exploring!

- Zorcal`

	fs.AddFile("home/guest/welcome.txt", []byte(welcomeMessage))

	// Easter egg.
	secretMessage := `🎉 Congratulations! You found the secret file! 🎉

You're clearly a true hacker at heart. Welcome to the matrix!

Fun fact: This filesystem exists entirely in memory and disappears
when you close your browser. It's like Schrödinger's file system -
it both exists and doesn't exist at the same time.

Keep exploring! There might be more secrets hidden in the code...

- Zorcal`

	fs.AddFile("home/zorcal/.secret.txt", []byte(secretMessage))
}

// Open implements fs.FS.
func (f *FS) Open(name string) (fs.File, error) {
	if name != "." && !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrInvalid}
	}

	name = path.Clean(name)
	if name == "." {
		name = ""
	}

	file, exists := f.files[name]
	if !exists {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
	}

	return &openFile{file: file, fs: f, path: name}, nil
}

// AddDir creates a new directory in the filesystem.
func (f *FS) AddDir(name string) {
	name = path.Clean(name)
	if name == "." {
		name = ""
	}

	f.files[name] = &File{
		name:     path.Base(name),
		isDir:    true,
		modTime:  time.Now(),
		children: make(map[string]*File),
	}
}

// AddFile creates a new file with the given content.
func (f *FS) AddFile(name string, content []byte) {
	name = path.Clean(name)
	dir := path.Dir(name)
	if dir == "." {
		dir = ""
	}

	f.files[name] = &File{
		name:    path.Base(name),
		isDir:   false,
		content: content,
		modTime: time.Now(),
	}

	if parent, exists := f.files[dir]; exists && parent.isDir {
		parent.children[path.Base(name)] = f.files[name]
	}
}
